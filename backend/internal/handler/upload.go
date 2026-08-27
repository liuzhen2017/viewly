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

// allowedUploadExt maps file extensions to storage prefixes. Videos go to the
// CF-fronted bucket; images share the same bucket under images/.
var allowedUploadExt = map[string]string{
	"mp4": "videos", "m3u8": "videos", "ts": "videos", "webm": "videos", "mov": "videos",
	"jpg": "images", "jpeg": "images", "png": "images", "webp": "images", "svg": "images",
}

var defaultUploadMIME = map[string]string{
	"mp4": "video/mp4", "m3u8": "application/vnd.apple.mpegurl", "ts": "video/mp2t",
	"webm": "video/webm", "mov": "video/quicktime",
	"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png",
	"webp": "image/webp", "svg": "image/svg+xml",
}

type presignReq struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type"`
}

// POST /api/admin/uploads/presign — issue a short-lived S3 PUT URL for the
// browser to upload directly (never through this server). The response also
// carries the CDN playback URL to store on the episode/cover field.
// Credentials come from the EC2 instance role; nothing is stored server-side.
func (h *Handler) PresignUpload(c *gin.Context) {
	var req presignReq
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
	cfg := h.Cfg.AWS
	if cfg.Bucket == "" || cfg.Region == "" {
		failBiz(c, 6002, "upload storage not configured")
		return
	}

	buf := make([]byte, 8)
	rand.Read(buf)
	key := fmt.Sprintf("%s/%s/%s.%s", prefix, time.Now().UTC().Format("200601"), hex.EncodeToString(buf), ext)
	ct := req.ContentType
	if ct == "" {
		ct = defaultUploadMIME[ext]
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		fail(c, http.StatusInternalServerError, "aws config: "+err.Error())
		return
	}
	presigner := s3.NewPresignClient(s3.NewFromConfig(awsCfg))
	p, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(ct),
	}, s3.WithPresignExpires(10*time.Minute))
	if err != nil {
		fail(c, http.StatusInternalServerError, "presign: "+err.Error())
		return
	}
	cdn := strings.TrimRight(cfg.CDNBase, "/") + "/" + key
	ok(c, gin.H{"upload_url": p.URL, "cdn_url": cdn, "key": key})
}
