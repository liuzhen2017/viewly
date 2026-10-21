package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"viewly/internal/model"
)

// TKPostAI integration: hand the logged-in user a publish link for a drama.
// Our server signs (userid, dramaid, timestamp) with HMAC-SHA256 and calls
// TKPostAI's link API; the user opens the returned URL in the same tab and
// publishes promo videos through their connected social accounts. Tasks are
// attributed back to our userid/dramaid in TKPostAI's exports.
//
// Signatures are single-use (REPLAY_DETECTED on reuse), but a link stays
// valid ~15 min with 2 entries — cache the latest per (user, drama) and
// reuse it instead of re-signing on every tap.

const tkpaLinkAPI = "https://tkpostai.com/api/integrations/drama/links"

var tkpaClient = &http.Client{Timeout: 15 * time.Second}

// POST /api/user/tkpostai/link {drama_id}
func (h *Handler) TKPostAILink(c *gin.Context) {
	if h.Cfg.TkPostAI.SigningKey == "" {
		failBiz(c, 7201, "promote not configured")
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

	// reuse a link that still has time left (signature reuse would 409)
	var ent model.TKEntry
	if h.DB.Where("tenant_id = ? AND user_id = ? AND drama_id = ? AND expires_at > ?",
		h.tID(c), u.ID, d.ID, time.Now().Add(2*time.Minute)).Order("id DESC").First(&ent).Error == nil {
		ok(c, gin.H{
			"url": ent.URL, "entry_id": ent.EntryID,
			"expires_at": ent.ExpiresAt.UnixMilli(), "remaining_uses": ent.RemainingUses,
		})
		return
	}

	userid := fmt.Sprintf("vu%dt%da", u.ID, u.TenantID)
	dramaid := fmt.Sprintf("%d", d.ID)
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(h.Cfg.TkPostAI.SigningKey))
	fmt.Fprintf(mac, "%s\n%s\n%d", userid, dramaid, ts)
	token := hex.EncodeToString(mac.Sum(nil))

	payload, _ := json.Marshal(map[string]interface{}{
		"userid": userid, "dramaid": dramaid, "title": d.Title,
		"timestamp": ts, "accesstoken": token,
	})
	resp, err := tkpaClient.Post(tkpaLinkAPI, "application/json", bytes.NewReader(payload))
	if err != nil {
		fail(c, http.StatusBadGateway, "promote service unreachable")
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusCreated {
		var e struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(raw, &e)
		msg := e.Code
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		failBiz(c, 7202, "promote: "+msg)
		return
	}
	var r struct {
		URL           string `json:"url"`
		EntryID       string `json:"entryId"`
		ExpiresAt     int64  `json:"expiresAt"`
		MaxUses       int    `json:"maxUses"`
		RemainingUses int    `json:"remainingUses"`
	}
	if err := json.Unmarshal(raw, &r); err != nil || r.URL == "" {
		fail(c, http.StatusBadGateway, "promote: unexpected response")
		return
	}
	ent = model.TKEntry{
		TenantID: h.tID(c), UserID: u.ID, DramaID: d.ID,
		EntryID: r.EntryID, URL: r.URL,
		ExpiresAt: time.UnixMilli(r.ExpiresAt), RemainingUses: r.RemainingUses,
	}
	h.DB.Create(&ent) // best-effort cache row
	ok(c, gin.H{
		"url": r.URL, "entry_id": r.EntryID,
		"expires_at": r.ExpiresAt, "remaining_uses": r.RemainingUses,
	})
}

// GET /api/user/tkpostai/entries — the user's own promote-link history
func (h *Handler) TKPostAIEntries(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	var ents []model.TKEntry
	h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID).Order("id DESC").Limit(50).Find(&ents)
	ok(c, ents)
}
