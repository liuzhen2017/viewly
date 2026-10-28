package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"viewly/internal/model"
)

// DramaCut (editing studio) integration.
//
// The source-import credential (dramacut.source_key, delivered 2026-10-26,
// id foxdrama-source-20261026) allows exactly two calls: submitting a
// source-import and querying imports we submitted. The user's editing
// session itself is opened through TKPost's creator workspace, so this
// handler only pushes episodes + tracks the import.
//
// Source whitelist on their side is cdn.likeviewly.com only for now —
// send the raw CDN URL (e.VideoURL), not the accelerated relay URL.

var dcutClient = &http.Client{Timeout: 30 * time.Second}

func (h *Handler) dcPost(path string, body interface{}, out interface{}) (int, []byte, error) {
	raw, _ := json.Marshal(body)
	r, _ := http.NewRequest("POST", h.Cfg.DramaCut.BaseURL+path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+h.Cfg.DramaCut.SourceKey)
	resp, err := dcutClient.Do(r)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if out != nil {
		json.Unmarshal(b, out)
	}
	return resp.StatusCode, b, nil
}

type dcImportReqUser struct {
	ExternalUserID string `json:"externalUserId"`
	InternalUserID uint64 `json:"-"`
}

// externalUserID is the stable ID TKPost knows this user by (minted in the
// promote registration flow: fox%07dt%d). DramaCut requires reusing it
// verbatim — no case changes, no new ids.
func externalUserID(u *model.User) string {
	return fmt.Sprintf("fox%07dt%d", u.ID, u.TenantID)
}

// POST /api/dramacut/import {drama_id, episode_ids?} — push source episodes
// to DramaCut; they create a project per drama and download in background.
func (h *Handler) DramaCutImport(c *gin.Context) {
	if h.Cfg.DramaCut.SourceKey == "" {
		failBiz(c, 7301, "dramacut not configured")
		return
	}
	u := c.MustGet("user").(*model.User)
	var req struct {
		DramaID    uint64   `json:"drama_id" binding:"required"`
		EpisodeIDs []uint64 `json:"episode_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "drama_id required")
		return
	}
	var d model.Drama
	if err := h.DB.Where("id = ? AND tenant_id = ? AND status = 1", req.DramaID, h.tID(c)).First(&d).Error; err != nil {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var eps []model.Episode
	q := h.DB.Where("drama_id = ? AND status = ?", d.ID, 1)
	if len(req.EpisodeIDs) > 0 {
		q = q.Where("id IN ?", req.EpisodeIDs)
	}
	q.Order("ep_index ASC").Find(&eps)
	if len(eps) == 0 {
		fail(c, http.StatusBadRequest, "no episodes")
		return
	}
	h.headFill(eps, false)

	type epOut struct {
		EpisodeID     string `json:"episodeId"`
		EpisodeNumber int    `json:"episodeNumber"`
		Version       string `json:"version"`
		SizeBytes     int64  `json:"sizeBytes"`
		DownloadURL   string `json:"downloadUrl"`
	}
	epList := make([]epOut, 0, len(eps))
	for _, e := range eps {
		dl := e.VideoURL
		if base := strings.TrimRight(h.Cfg.Integration.DownloadBase, "/"); base != "" {
			if i := strings.Index(e.VideoURL, "cdn.likeviewly.com/"); i >= 0 {
				dl = base + "/" + e.VideoURL[i+len("cdn.likeviewly.com/"):]
			}
		}
		epList = append(epList, epOut{
			EpisodeID: fmt.Sprintf("foxdrama-ep-%d", e.ID), EpisodeNumber: e.EpIndex,
			Version: e.FileEtag, SizeBytes: e.FileSize,
			DownloadURL: dl, // download_base (OSS mirror) — cdn.likeviewly.com fallback
		})
	}
	reqID := fmt.Sprintf("fdi%d%d", time.Now().UnixNano(), u.ID)
	payload := map[string]interface{}{
		"tenantId":       h.Cfg.DramaCut.TenantID,
		"externalUserId": externalUserID(u),
		"requestId":      reqID,
		"source":         "foxdrama",
		"dramaId":        fmt.Sprintf("foxdrama-drama-%d", d.ID),
		"dramaName":      d.Title,
		"autoAnalyze":    false,
		"episodes":       epList,
	}
	var out struct {
		ImportID string `json:"importId"`
		TaskID   string `json:"taskId"`
		Error    string `json:"error"`
		Code     string `json:"code"`
	}
	code, raw, err := h.dcPost("/api/integration/source-imports", payload, &out)
	if err != nil {
		fail(c, http.StatusBadGateway, "dramacut unreachable")
		return
	}
	rec := model.DCImport{
		TenantID: h.tID(c), UserID: u.ID, DramaID: d.ID,
		ImportID: out.ImportID, RequestID: reqID,
		Status: "submitted", RawResponse: string(raw),
	}
	if code != http.StatusOK && code != http.StatusCreated && code != http.StatusAccepted {
		rec.Status = "rejected"
		h.DB.Create(&rec)
		failBiz(c, 7303, fmt.Sprintf("dramacut import: HTTP %d %s", code, truncateStr(string(raw), 200)))
		return
	}
	h.DB.Create(&rec)
	ok(c, gin.H{"import_id": out.ImportID, "drama_id": d.ID, "episodes": len(epList)})
}

// GET /api/dramacut/import/status?drama_id= — latest import + remote state
func (h *Handler) DramaCutImportStatus(c *gin.Context) {
	if h.Cfg.DramaCut.SourceKey == "" {
		failBiz(c, 7301, "dramacut not configured")
		return
	}
	u := c.MustGet("user").(*model.User)
	did := uint64(toInt(c.Query("drama_id"), 0))
	var rec model.DCImport
	q := h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID)
	if did > 0 {
		q = q.Where("drama_id = ?", did)
	}
	if err := q.Order("id DESC").First(&rec).Error; err != nil {
		fail(c, http.StatusNotFound, "no import yet")
		return
	}
	// ask DramaCut for the live state of our own submission
	url := fmt.Sprintf("%s/api/integration/source-imports/%s?externalUserId=%s",
		h.Cfg.DramaCut.BaseURL, rec.ImportID, externalUserID(u))
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Set("Authorization", "Bearer "+h.Cfg.DramaCut.SourceKey)
	resp, err := dcutClient.Do(r)
	remote := ""
	if err == nil {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		resp.Body.Close()
		remote = string(b)
		if resp.StatusCode == http.StatusOK {
			h.DB.Model(&rec).Updates(map[string]interface{}{"raw_response": remote, "status": "polled"})
		}
	}
	ok(c, gin.H{
		"import_id": rec.ImportID, "drama_id": rec.DramaID, "status": rec.Status,
		"created_at": rec.CreatedAt, "remote": json.RawMessage(orEmpty(remote)),
	})
}

func orEmpty(s string) string {
	if s == "" {
		return "{}"
	}
	return s
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
