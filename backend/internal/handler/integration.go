package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"viewly/internal/model"
)

// Server-to-server integration API for the DramaCut/TKPost pipeline.
// Auth: Authorization: Bearer <integration token> (config integration.token,
// delivered privately — never exposed to browsers). User identity comes as
// the externalUserId we minted in the promote flows (fox%07dt%d); we parse
// it back to our user+tenant, so callers can't probe other users' data.
//
// "已授权剧目" rule: every published drama on the user's tenant is
// promotable (matches the H5 promote page). Free/paid per-episode coin
// locks are a playback concern, not a promotion concern.

var extUserRe = regexp.MustCompile(`^fox(\d{1,9})t(\d{1,6})$`)

func (h *Handler) IntegrationAuth(c *gin.Context) {
	if h.Cfg.Integration.Token == "" {
		failBiz(c, 7401, "integration api disabled")
		c.Abort()
		return
	}
	auth := c.GetHeader("Authorization")
	if auth != "Bearer "+h.Cfg.Integration.Token {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "bad integration token"})
		return
	}
	uid, tid := parseExtUser(c.Query("userid"))
	if uid == 0 {
		// TKPost-native accounts (e.g. yixi8888) have no FoxDrama user row.
		// Accept them as tenant-1 (main) callers; the Bearer token is the
		// real auth boundary, the userid is only for tenant binding.
		raw := strings.TrimSpace(c.Query("userid"))
		if raw == "" || len(raw) > 64 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "userid required"})
			return
		}
		tid = 1
	}
	c.Set("int_uid", uid)
	c.Set("int_tid", tid)
}

func parseExtUser(s string) (uint64, uint64) {
	m := extUserRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0
	}
	uid, _ := strconv.ParseUint(m[1], 10, 64)
	tid, _ := strconv.ParseUint(m[2], 10, 64)
	return uid, tid
}

func (h *Handler) intUser(c *gin.Context) (uint64, uint64) {
	return c.GetUint64("int_uid"), c.GetUint64("int_tid")
}

// GET /api/integration/dramas?userid=&page=&size=&q= — authorized drama list
func (h *Handler) IntegrationDramas(c *gin.Context) {
	uid, tid := h.intUser(c)
	if uid > 0 {
		var u model.User
		if err := h.DB.Where("id = ? AND tenant_id = ?", uid, tid).First(&u).Error; err != nil {
			fail(c, http.StatusNotFound, "user not found")
			return
		}
	}
	page := toInt(c.Query("page"), 1)
	size := toInt(c.Query("size"), 20)
	if size < 1 || size > 50 {
		size = 20
	}
	q := strings.TrimSpace(c.Query("q"))
	db := h.DB.Table("dramas").Where("dramas.tenant_id = ? AND dramas.status = ?", tid, 1)
	if q != "" {
		db = db.Where("dramas.title LIKE ?", "%"+q+"%")
	}
	var total int64
	db.Count(&total)
	type row struct {
		ID       uint64 `json:"id"`
		Title    string `json:"title"`
		Cover    string `json:"cover"`
		EpsCount int    `json:"episodes_count"`
	}
	var rows []row
	db.Select("dramas.id, dramas.title, dramas.cover, (SELECT COUNT(*) FROM episodes e WHERE e.drama_id = dramas.id AND e.status = 1) AS eps_count").
		Order("dramas.id DESC").Limit(size).Offset((page - 1) * size).Scan(&rows)
	type outRow struct {
		DramaID       string `json:"dramaId"`
		DramaName     string `json:"dramaName"`
		EpisodeCount  int    `json:"episodeCount"`
		Cover         string `json:"cover"`
	}
	list := make([]outRow, 0, len(rows))
	for _, r := range rows {
		list = append(list, outRow{
			DramaID: fmt.Sprintf("foxdrama-drama-%d", r.ID), DramaName: r.Title,
			EpisodeCount: r.EpsCount, Cover: r.Cover,
		})
	}
	ok(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

// GET /api/integration/dramas/:dramaId/episodes?userid= — full episode manifest
func (h *Handler) IntegrationEpisodes(c *gin.Context) {
	uid, tid := h.intUser(c)
	if uid > 0 {
		var u model.User
		if err := h.DB.Where("id = ? AND tenant_id = ?", uid, tid).First(&u).Error; err != nil {
			fail(c, http.StatusNotFound, "user not found")
			return
		}
	}
	did := parseDramaID(c.Param("dramaId"))
	if did == 0 {
		fail(c, http.StatusBadRequest, "bad dramaId")
		return
	}
	var d model.Drama
	if err := h.DB.Where("id = ? AND tenant_id = ? AND status = 1", did, tid).First(&d).Error; err != nil {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var eps []model.Episode
	h.DB.Where("drama_id = ? AND status = ?", did, 1).Order("ep_index ASC").Find(&eps)

	// refresh file size/etag (cached; ?refresh=1 forces re-HEAD for replaced files)
	force := c.Query("refresh") == "1"
	h.headFill(eps, force)

	type outEp struct {
		EpisodeID     string `json:"episodeId"`
		EpisodeNumber int    `json:"episodeNumber"`
		Title         string `json:"title"`
		Filename      string `json:"filename"`
		DownloadURL   string `json:"downloadUrl"`
		SizeBytes     int64  `json:"sizeBytes"`
		Version       string `json:"version"`
		DownloadHost  string `json:"downloadHost"`
	}
	list := make([]outEp, 0, len(eps))
	for _, e := range eps {
		list = append(list, outEp{
			EpisodeID: fmt.Sprintf("foxdrama-ep-%d", e.ID), EpisodeNumber: e.EpIndex,
			Title: e.Title, Filename: fmt.Sprintf("%03d.mp4", e.EpIndex),
			DownloadURL: h.dlURL(e.VideoURL),
			SizeBytes:   e.FileSize, Version: e.FileEtag,
			DownloadHost: h.Cfg.Integration.DownloadBase,
		})
	}
	ok(c, gin.H{
		"dramaId": fmt.Sprintf("foxdrama-drama-%d", d.ID), "dramaName": d.Title,
		"episodeCount": len(list), "episodes": list,
		"linkPolicy": gin.H{
			"validity": "permanent (no expiry, no renewal needed)",
			"cacheRule": "verify Content-Length + ETag before use; after a source file is replaced its ETag changes — call with refresh=1 or compare the version field",
		},
	})
}

func parseDramaID(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "foxdrama-drama-")
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}

var headClient = &http.Client{Timeout: 10 * time.Second}

// dlURL maps a cdn.likeviewly.com video URL onto the configured download
// base (OSS mirror) — the manifest and the ETag cache must describe the
// SAME copy of the file, or version checks drift (CDN/S3 etags differ from
// OSS etags even for identical bytes).
func (h *Handler) dlURL(videoURL string) string {
	base := strings.TrimRight(h.Cfg.Integration.DownloadBase, "/")
	if base == "" {
		return videoURL
	}
	if i := strings.Index(videoURL, "cdn.likeviewly.com"); i >= 0 {
		return base + videoURL[i+len("cdn.likeviewly.com"):]
	}
	return videoURL
}

// headFill lazily caches Content-Length + ETag from the download base (12 concurrent)
func (h *Handler) headFill(eps []model.Episode, force bool) {
	var need []*model.Episode
	for i := range eps {
		if force || eps[i].FileSize == 0 || eps[i].FileEtag == "" {
			need = append(need, &eps[i])
		}
	}
	if len(need) == 0 {
		return
	}
	sem := make(chan struct{}, 12)
	var wg sync.WaitGroup
	var mu sync.Mutex
	updates := map[uint64][2]interface{}{}
	for _, e := range need {
		wg.Add(1)
		go func(e *model.Episode) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			req, _ := http.NewRequest("HEAD", h.dlURL(e.VideoURL), nil)
			resp, err := headClient.Do(req)
			if err != nil || resp.StatusCode != http.StatusOK {
				return
			}
			sz := resp.ContentLength
			etag := strings.Trim(resp.Header.Get("ETag"), `"`)
			mu.Lock()
			updates[e.ID] = [2]interface{}{sz, etag}
			e.FileSize, e.FileEtag = sz, etag
			mu.Unlock()
		}(e)
	}
	wg.Wait()
	if len(updates) > 0 {
		now := time.Now().UTC()
		for id, v := range updates {
			h.DB.Model(&model.Episode{}).Where("id = ?", id).Updates(map[string]interface{}{
				"file_size": v[0], "file_etag": v[1], "file_meta_at": now,
			})
		}
	}
}
