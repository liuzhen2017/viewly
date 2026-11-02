package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	"viewly/internal/model"
)

// ffmpeg slicer: cut one episode into N short clips for the publish matrix.
// Stream copy (-c copy, input seek) — no re-encode, cuts land on keyframes,
// which is good enough for MVP and the only thing the 2GB ARM box can afford.
// Jobs are DB-backed (tk_slice_jobs), so unlike remux's in-memory channel they
// survive restarts; errors are recorded, never swallowed.

var (
	sliceMu      sync.Mutex
	sliceRunning bool
)

func kickSlicer(h *Handler) { go runSlicer(h) }

func runSlicer(h *Handler) {
	sliceMu.Lock()
	if sliceRunning {
		sliceMu.Unlock()
		return
	}
	sliceRunning = true
	sliceMu.Unlock()
	defer func() { sliceMu.Lock(); sliceRunning = false; sliceMu.Unlock() }()

	for {
		var job model.TKSliceJob
		if err := h.DB.Where("status = ?", "queued").Order("id ASC").First(&job).Error; err != nil {
			return // queue drained
		}
		h.processSliceJob(&job)
	}
}

func (h *Handler) processSliceJob(j *model.TKSliceJob) {
	failJob := func(format string, args ...interface{}) {
		h.DB.Model(j).Updates(map[string]interface{}{
			"status": "failed", "error": truncStr(fmt.Sprintf(format, args...), 490)})
		log.Printf("[slicer] job %d failed: %s", j.ID, fmt.Sprintf(format, args...))
	}
	h.DB.Model(j).Update("status", "running")

	var ep model.Episode
	if err := h.DB.Where("id = ? AND tenant_id = ?", j.EpisodeID, j.TenantID).First(&ep).Error; err != nil {
		failJob("episode missing")
		return
	}
	if ep.VideoURL == "" {
		failJob("episode has no video")
		return
	}
	var drama model.Drama
	dramaTitle := ""
	if err := h.DB.First(&drama, ep.DramaID).Error; err == nil {
		dramaTitle = drama.Title
	}

	seg := j.SegSec
	if seg < 10 {
		seg = 10
	}
	if seg > 600 {
		seg = 600
	}
	maxN := j.MaxCount
	if maxN < 1 {
		maxN = 1
	}
	if maxN > 50 {
		maxN = 50
	}
	n := maxN
	if ep.DurationSec > 0 {
		full := (ep.DurationSec + seg - 1) / seg
		if full < n {
			n = full
		}
	}
	if n < 1 {
		failJob("nothing to cut (duration %ds, seg %ds)", ep.DurationSec, seg)
		return
	}

	tmp, err := os.MkdirTemp(h.Cfg.FFmpeg.TmpDir, "slice-*")
	if err != nil {
		failJob("temp dir: %v", err)
		return
	}
	defer os.RemoveAll(tmp)
	in := filepath.Join(tmp, "in.mp4")

	// fetch source via CDN — bucket policy only serves Cloudflare IPs,
	// same constraint remux works around.
	dl, err := http.Get(ep.VideoURL)
	if err != nil {
		failJob("download: %v", err)
		return
	}
	if dl.StatusCode != http.StatusOK {
		dl.Body.Close()
		failJob("download: HTTP %d", dl.StatusCode)
		return
	}
	f, err := os.Create(in)
	if err != nil {
		dl.Body.Close()
		failJob("temp file: %v", err)
		return
	}
	_, cerr := io.Copy(f, dl.Body)
	f.Close()
	dl.Body.Close()
	if cerr != nil {
		failJob("download: %v", cerr)
		return
	}

	baseTitle := fmt.Sprintf("E%d", ep.EpIndex)
	if dramaTitle != "" {
		baseTitle = fmt.Sprintf("%s E%d", dramaTitle, ep.EpIndex)
	}

	created := 0
	for i := 0; i < n; i++ {
		out := filepath.Join(tmp, fmt.Sprintf("out-%d.mp4", i))
		start := int64(i) * int64(seg)
		cmd := exec.Command(h.Cfg.FFmpeg.Path, "-nostdin", "-y", "-loglevel", "error",
			"-ss", fmt.Sprintf("%d", start), "-t", fmt.Sprintf("%d", seg),
			"-i", in, "-c", "copy", "-movflags", "+faststart", out)
		if o, err := cmd.CombinedOutput(); err != nil {
			if i > 0 {
				break // short tail: keep the clips we already got
			}
			failJob("ffmpeg: %v %s", err, truncStr(string(o), 200))
			return
		}
		st, err := os.Stat(out)
		if err != nil || st.Size() < 1024 {
			break // empty output = past end of file
		}

		key := fmt.Sprintf("videos/%s/%s-s%d.mp4", time.Now().UTC().Format("200601"), randHex(8), i)
		ctx, cancel := context.WithTimeout(context.Background(), 15*60*time.Second)
		awsCfg, _ := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(h.Cfg.AWS.Region))
		of, err := os.Open(out)
		if err != nil {
			cancel()
			failJob("open clip: %v", err)
			return
		}
		_, err = s3.NewFromConfig(awsCfg).PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(h.Cfg.AWS.Bucket), Key: aws.String(key),
			Body: of, ContentType: aws.String("video/mp4"),
		})
		of.Close()
		cancel()
		if err != nil {
			failJob("upload clip: %v", err)
			return
		}

		uid := j.UserID
		clip := model.TikTokClip{
			TenantID: j.TenantID, UserID: &uid,
			Title:    fmt.Sprintf("%s #%d", baseTitle, i+1),
			VideoURL: strings.TrimRight(h.Cfg.AWS.CDNBase, "/") + "/" + key,
			SizeBytes: st.Size(), DurationSec: seg,
		}
		if err := h.DB.Create(&clip).Error; err != nil {
			failJob("save clip: %v", err)
			return
		}
		created++
	}
	h.DB.Model(j).Updates(map[string]interface{}{"status": "done", "clip_count": created, "error": ""})
	log.Printf("[slicer] job %d done: %d clips from episode %d", j.ID, created, j.EpisodeID)
}

// ---------- API ----------

// POST /api/user/tk/slice-jobs {episode_id, seg_sec, max_count}
func (h *Handler) TKUserSliceJobCreate(c *gin.Context) {
	u := h.tkUser(c)
	var p struct {
		EpisodeID uint64 `json:"episode_id" binding:"required"`
		SegSec    int    `json:"seg_sec"`
		MaxCount  int    `json:"max_count"`
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, http.StatusBadRequest, "episode_id required")
		return
	}
	var ep model.Episode
	if err := h.DB.Where("id = ? AND tenant_id = ? AND status = 1", p.EpisodeID, h.tID(c)).First(&ep).Error; err != nil || ep.VideoURL == "" {
		failBiz(c, 7331, "episode not found or has no video")
		return
	}
	seg := p.SegSec
	if seg < 10 {
		seg = 10
	}
	if seg > 600 {
		seg = 600
	}
	maxN := p.MaxCount
	if maxN < 1 {
		maxN = 1
	}
	if maxN > 50 {
		maxN = 50
	}
	job := model.TKSliceJob{
		TenantID: h.tID(c), UserID: u.ID, EpisodeID: ep.ID,
		SegSec: seg, MaxCount: maxN, Status: "queued",
	}
	if err := h.DB.Create(&job).Error; err != nil {
		fail(c, http.StatusInternalServerError, "create job failed")
		return
	}
	kickSlicer(h)
	ok(c, job)
}

// GET /api/user/tk/slice-jobs — with episode/drama titles for display
func (h *Handler) TKUserSliceJobs(c *gin.Context) {
	u := h.tkUser(c)
	var jobs []model.TKSliceJob
	h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID).Order("id DESC").Limit(50).Find(&jobs)
	type row struct {
		model.TKSliceJob
		EpisodeTitle string `json:"episode_title"`
		DramaTitle   string `json:"drama_title"`
	}
	out := make([]row, 0, len(jobs))
	for _, j := range jobs {
		r := row{TKSliceJob: j}
		var ep model.Episode
		if err := h.DB.First(&ep, j.EpisodeID).Error; err == nil {
			r.EpisodeTitle = ep.Title
			var drama model.Drama
			if err := h.DB.First(&drama, ep.DramaID).Error; err == nil {
				r.DramaTitle = drama.Title
			}
		}
		out = append(out, r)
	}
	ok(c, out)
}
