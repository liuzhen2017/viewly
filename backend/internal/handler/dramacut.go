package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"viewly/internal/model"
)

// DramaCut (editing studio) integration. Our server calls their ticket API
// with the logged-in user's stable ID and the drama they want to cut; the
// returned iframeUrl opens their editor with an established session. The
// source-import API (pushing our episodes into their projects) is specified
// but not yet live on their side — this file ships the entry point first.

var dcutClient = &http.Client{Timeout: 15 * time.Second}

// POST /api/dramacut/entry {drama_id} — get the editor entry URL for the user
func (h *Handler) DramaCutEntry(c *gin.Context) {
	if h.Cfg.DramaCut.BaseURL == "" || h.Cfg.DramaCut.Bearer == "" {
		failBiz(c, 7301, "dramacut not configured")
		return
	}
	u := c.MustGet("user").(*model.User)
	var req struct {
		DramaID uint64 `json:"drama_id" binding:"required"`
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

	body, _ := json.Marshal(map[string]string{
		"tenantId":       h.Cfg.DramaCut.TenantID,
		"externalUserId": fmt.Sprintf("fox%07dt%d", u.ID, u.TenantID),
		"dramaId":        fmt.Sprintf("foxdrama-drama-%d", d.ID),
		"requestId":      fmt.Sprintf("fd%d%d", time.Now().UnixMilli(), u.ID),
	})
	r, _ := http.NewRequest("POST", h.Cfg.DramaCut.BaseURL+"/api/integration/tickets", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+h.Cfg.DramaCut.Bearer)
	resp, err := dcutClient.Do(r)
	if err != nil {
		fail(c, http.StatusBadGateway, "dramacut unreachable")
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		failBiz(c, 7302, "dramacut: HTTP "+fmt.Sprint(resp.StatusCode)+" "+string(raw[:min(len(raw), 120)]))
		return
	}
	var out struct {
		IframeURL string `json:"iframeUrl"`
		URL       string `json:"url"`
	}
	json.Unmarshal(raw, &out)
	link := out.IframeURL
	if link == "" {
		link = out.URL
	}
	if link == "" {
		fail(c, http.StatusBadGateway, "dramacut: no iframe url")
		return
	}
	// their spec: concatenate with the service base address when relative
	if len(link) > 0 && link[0] == '/' {
		link = h.Cfg.DramaCut.BaseURL + link
	}
	ok(c, gin.H{"url": link, "drama_id": d.ID, "drama_title": d.Title})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
