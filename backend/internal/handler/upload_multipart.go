package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
)

// Chunked multipart upload for large videos: browser splits the file into
// ~16MB chunks and POSTs them concurrently; each chunk streams straight into
// an S3 UploadPart (no disk on this box). Keeps every request well under
// proxy body limits and lets a failed chunk retry alone.

type mpSession struct {
	Key      string
	UploadID string
	Created  time.Time
}

var (
	mpMu       sync.Mutex
	mpSessions = map[string]*mpSession{}
)

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) awsS3(c *gin.Context) (*s3.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(h.Cfg.AWS.Region))
	if err != nil {
		return nil, nil, nil, err
	}
	return s3.NewFromConfig(cfg), ctx, cancel, nil
}

type mpInitReq struct {
	Filename string `json:"filename" binding:"required"`
}

// POST /api/admin/uploads/mp/init
func (h *Handler) MpInit(c *gin.Context) {
	var req mpInitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "filename required")
		return
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(req.Filename)), ".")
	prefix, allowed := allowedUploadExt[ext]
	if !allowed {
		failBiz(c, 6001, "unsupported file type: "+ext)
		return
	}
	buf := make([]byte, 8)
	rand.Read(buf)
	key := fmt.Sprintf("%s/%s/%s.%s", prefix, time.Now().UTC().Format("200601"), hex.EncodeToString(buf), ext)

	client, ctx, cancel, err := h.awsS3(c)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer cancel()
	out, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(h.Cfg.AWS.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(defaultUploadMIME[ext]),
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "init: "+err.Error())
		return
	}
	id := fmt.Sprintf("%x", time.Now().UnixNano())[:10] + randHex(6)
	mpMu.Lock()
	mpSessions[id] = &mpSession{Key: key, UploadID: *out.UploadId, Created: time.Now()}
	// opportunistically drop sessions older than a day
	for k, v := range mpSessions {
		if time.Since(v.Created) > 24*time.Hour {
			delete(mpSessions, k)
		}
	}
	mpMu.Unlock()
	ok(c, gin.H{"upload_id": id})
}

func (h *Handler) mpSession(id string) *mpSession {
	mpMu.Lock()
	defer mpMu.Unlock()
	return mpSessions[id]
}

// POST /api/admin/uploads/mp/chunk?upload_id=&part=  (multipart "chunk")
func (h *Handler) MpChunk(c *gin.Context) {
	sess := h.mpSession(c.Query("upload_id"))
	if sess == nil {
		failBiz(c, 6101, "unknown upload_id")
		return
	}
	part := int32(toInt(c.Query("part"), 0))
	if part < 1 || part > 10000 {
		fail(c, http.StatusBadRequest, "bad part number")
		return
	}
	fh, err := c.FormFile("chunk")
	if err != nil {
		fail(c, http.StatusBadRequest, "multipart field 'chunk' required")
		return
	}
	src, err := fh.Open()
	if err != nil {
		fail(c, http.StatusInternalServerError, "open chunk failed")
		return
	}
	defer src.Close()

	client, ctx, cancel, err := h.awsS3(c)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer cancel()
	pn := part
	out, err := client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(h.Cfg.AWS.Bucket),
		Key:        aws.String(sess.Key),
		UploadId:   aws.String(sess.UploadID),
		PartNumber: &pn,
		Body:       src,
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "chunk: "+err.Error())
		return
	}
	ok(c, gin.H{"part": part, "etag": strings.Trim(*out.ETag, "\"")})
}

type mpCompleteReq struct {
	UploadID string `json:"upload_id" binding:"required"`
	Parts    []struct {
		Part int    `json:"part"`
		ETag string `json:"etag"`
	} `json:"parts" binding:"required"`
}

// POST /api/admin/uploads/mp/complete
func (h *Handler) MpComplete(c *gin.Context) {
	var req mpCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "upload_id and parts required")
		return
	}
	sess := h.mpSession(req.UploadID)
	if sess == nil {
		failBiz(c, 6101, "unknown upload_id")
		return
	}
	client, ctx, cancel, err := h.awsS3(c)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer cancel()
	parts := make([]types.CompletedPart, 0, len(req.Parts))
	for _, p := range req.Parts {
		pn := int32(p.Part)
		etag := strings.Trim(p.ETag, `\"`)
		parts = append(parts, types.CompletedPart{PartNumber: &pn, ETag: aws.String(etag)})
	}
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket: aws.String(h.Cfg.AWS.Bucket), Key: aws.String(sess.Key),
		UploadId: aws.String(sess.UploadID), MultipartUpload: &types.CompletedMultipartUpload{Parts: parts},
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "complete: "+err.Error())
		return
	}
	mpMu.Lock()
	delete(mpSessions, req.UploadID)
	mpMu.Unlock()
	cdn := strings.TrimRight(h.Cfg.AWS.CDNBase, "/") + "/" + sess.Key
	enqueueRemux(sess.Key)
	ok(c, gin.H{"cdn_url": cdn, "key": sess.Key})
}

// POST /api/admin/uploads/mp/abort {upload_id}
func (h *Handler) MpAbort(c *gin.Context) {
	var req struct {
		UploadID string `json:"upload_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "upload_id required")
		return
	}
	sess := h.mpSession(req.UploadID)
	if sess == nil {
		ok(c, gin.H{"aborted": true})
		return
	}
	client, ctx, cancel, err := h.awsS3(c)
	if err == nil {
		defer cancel()
		client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket: aws.String(h.Cfg.AWS.Bucket), Key: aws.String(sess.Key), UploadId: aws.String(sess.UploadID),
		})
	}
	mpMu.Lock()
	delete(mpSessions, req.UploadID)
	mpMu.Unlock()
	ok(c, gin.H{"aborted": true})
}
