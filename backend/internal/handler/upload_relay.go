package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// POST /api/admin/uploads (multipart "file") — server-relay upload: the browser
// sends the video to this API (same origin, no CORS), and we stream it to S3
// using the instance role. Simpler than presigned browser PUTs and works from
// any admin origin without bucket CORS configuration.
func (h *Handler) UploadRelay(c *gin.Context) {
	cfg := h.Cfg.AWS
	if cfg.Bucket == "" || cfg.Region == "" {
		failBiz(c, 6002, "upload storage not configured")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "multipart field 'file' required")
		return
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	prefix, allowed := allowedUploadExt[ext]
	if !allowed {
		failBiz(c, 6001, "unsupported file type: "+ext)
		return
	}

	buf := make([]byte, 8)
	rand.Read(buf)
	key := fmt.Sprintf("%s/%s/%s.%s", prefix, time.Now().UTC().Format("200601"), hex.EncodeToString(buf), ext)

	src, err := fh.Open()
	if err != nil {
		fail(c, http.StatusInternalServerError, "open upload failed")
		return
	}
	defer src.Close()

	ctx, cancel := contextWithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		fail(c, http.StatusInternalServerError, "aws config: "+err.Error())
		return
	}
	_, err = s3.NewFromConfig(awsCfg).PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(defaultUploadMIME[ext]),
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "s3 put: "+err.Error())
		return
	}
	cdn := strings.TrimRight(cfg.CDNBase, "/") + "/" + key
	ok(c, gin.H{"cdn_url": cdn, "key": key, "size": fh.Size})
}

// contextWithTimeout keeps the helper local so upload_relay has no extra imports.
func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
