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

const tkpaBase = "https://tkpostai.com"
const tkpaLinkAPI = tkpaBase + "/api/integrations/drama/links"

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

	dramaid := fmt.Sprintf("%d", d.ID)

	// preferred: new auth API — auto-register + token link, zero user friction
	if h.Cfg.TkPostAI.AppID != "" {
		token, err := h.ensureTkAccount(u)
		if err != nil {
			failBiz(c, 7204, "promote account: "+err.Error())
			return
		}
		r, err := h.tkpaNewLink(token, dramaid, d.Title)
		if err != nil {
			failBiz(c, 7205, "promote link: "+err.Error())
			return
		}
		h.DB.Create(&model.TKEntry{
			TenantID: h.tID(c), UserID: u.ID, DramaID: d.ID,
			EntryID: r.EntryID, URL: r.URL,
			ExpiresAt: time.UnixMilli(r.ExpiresAt), RemainingUses: r.RemainingUses,
		})
		ok(c, gin.H{
			"url": r.URL, "entry_id": r.EntryID,
			"expires_at": r.ExpiresAt, "remaining_uses": r.RemainingUses,
		})
		return
	}

	// legacy: signed-link API (kept until every partner moves to the new one)
	userid := fmt.Sprintf("vu%dt%da", u.ID, u.TenantID)
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

// POST /api/tkpostai/synced — the user confirms they finished the one-time
// account binding on TKPostAI (their platform has no server-side register
// API, so the flag is user-confirmed; see the link flow in TKPostAILink).
func (h *Handler) TKPostAISynced(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	// require at least one promote link so the flag can't be set blindly
	var cnt int64
	h.DB.Model(&model.TKEntry{}).Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID).Count(&cnt)
	if cnt == 0 {
		failBiz(c, 7203, "open a promote link first")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", u.ID).Update("tk_synced", 1)
	ok(c, gin.H{"tk_synced": 1})
}

// ---------- new auth API (2026-10-23): server-side register + token links ----------
// When appid is configured we register every promoter automatically (username
// we generate + a strong random password), log in for a 2h token, and create
// links with it — the user never sees TKPostAI's registration form.

func tkpaGenPassword() string { return randHex(12) }

// ensureTkAccount returns a valid TKPostAI token for the user, registering
// and/or logging in as needed. Credentials and token are cached on the row.
func (h *Handler) ensureTkAccount(u *model.User) (string, error) {
	nowMs := time.Now().UnixMilli()
	if u.TkToken != "" && u.TkTokenExpires > nowMs+60_000 {
		return u.TkToken, nil
	}
	type authResp struct {
		Userid    string `json:"userid"`
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expiresAt"`
		Error     string `json:"error"`
		Code      string `json:"code"`
	}
	post := func(path, username, password string) (*authResp, int, error) {
		body, _ := json.Marshal(map[string]string{
			"appid": h.Cfg.TkPostAI.AppID, "username": username, "password": password,
		})
		resp, err := tkpaClient.Post(tkpaBase+path, "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
		var r authResp
		json.Unmarshal(raw, &r)
		return &r, resp.StatusCode, nil
	}
	password := u.TkPassword
	if u.TkUsername == "" {
		// first time: generate credentials and register (retry once on name clash)
		for i := 0; i < 2; i++ {
			username := fmt.Sprintf("fox%07dt%d", u.ID, u.TenantID)
			if i == 1 {
				username += "x" + randHex(3)
			}
			pw := tkpaGenPassword()
			r, code, err := post("/api/integrations/auth/register", username, pw)
			if err != nil {
				return "", err
			}
			if code == http.StatusCreated {
				u.TkUsername, u.TkPassword = username, pw
				break
			}
			if code == http.StatusConflict && r.Code == "USERNAME_EXISTS" && i == 0 {
				continue // name taken (shouldn't happen with our ID scheme) — retry with suffix
			}
			return "", fmt.Errorf("register: %s %s", r.Code, r.Error)
		}
		if u.TkUsername == "" {
			return "", fmt.Errorf("register: no username")
		}
		password = u.TkPassword
	}
	r, code, err := post("/api/integrations/auth/login", u.TkUsername, password)
	if err != nil {
		return "", err
	}
	if code != http.StatusOK || r.Token == "" {
		return "", fmt.Errorf("login: %s %s", r.Code, r.Error)
	}
	u.TkToken, u.TkTokenExpires = r.Token, r.ExpiresAt
	h.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"tk_username": u.TkUsername, "tk_password": u.TkPassword,
		"tk_token": u.TkToken, "tk_token_expires": u.TkTokenExpires,
		"tk_synced": 1,
	})
	return u.TkToken, nil
}

type tkLinkResp struct {
	URL           string `json:"url"`
	EntryID       string `json:"entryId"`
	ExpiresAt     int64  `json:"expiresAt"`
	RemainingUses int    `json:"remainingUses"`
	Error         string `json:"error"`
	Code          string `json:"code"`
}

// tkpaNewLink creates a publish link via the token API.
func (h *Handler) tkpaNewLink(token, dramaid, title string) (tkLinkResp, error) {
	var out tkLinkResp
	body, _ := json.Marshal(map[string]string{
		"appid": h.Cfg.TkPostAI.AppID, "token": token,
		"dramaid": dramaid, "title": title,
	})
	resp, err := tkpaClient.Post(tkpaBase+"/api/integrations/drama/links", "application/json", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	json.Unmarshal(raw, &out)
	if resp.StatusCode != http.StatusCreated || out.URL == "" {
		return out, fmt.Errorf("link: %s %s", out.Code, out.Error)
	}
	return out, nil
}
