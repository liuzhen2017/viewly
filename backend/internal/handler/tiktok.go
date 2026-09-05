package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"viewly/internal/model"
)

// TikTok account hosting + one-click publishing via the Content Posting API.
// Flow: admin opens the authorize URL (user.info.basic + video.upload +
// video.publish) -> callback exchanges the code for tokens -> the account is
// hosted here; publishing downloads the clip from our CDN and pushes it to
// TikTok in chunks (init -> PUT upload_url -> async processing).

const (
	ttAuthorizeURL = "https://www.tiktok.com/v2/auth/authorize/"
	ttAPIBase      = "https://open.tiktokapis.com/v2"
	ttChunkSize    = 16 * 1024 * 1024
)

// ---------- OAuth ----------

// GET /api/admin/tiktok/connect-url
func (h *Handler) TikTokConnectURL(c *gin.Context) {
	if h.Cfg.TikTok.ClientKey == "" {
		failBiz(c, 7001, "tiktok not configured: set client_key/client_secret in config.yaml")
		return
	}
	state := fmt.Sprintf("t%d.%s", h.tID(c), randHex(8))
	q := url.Values{
		"client_key":    {h.Cfg.TikTok.ClientKey},
		"scope":         {"user.info.basic,video.publish,video.upload"},
		"response_type": {"code"},
		"redirect_uri":  {h.Cfg.TikTok.RedirectURL},
		"state":         {state},
	}
	ok(c, gin.H{"url": ttAuthorizeURL + "?" + q.Encode()})
}

// GET /api/tiktok/callback (public — TikTok redirects here)
func (h *Handler) TikTokCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	back := "https://admin.likeviewly.com/#/tiktok"
	if code == "" || !strings.HasPrefix(state, "t") {
		c.Redirect(http.StatusFound, back+"?err=missing_code")
		return
	}
	var tid uint64
	fmt.Sscanf(state[1:], "%d", &tid)
	if tid == 0 {
		c.Redirect(http.StatusFound, back+"?err=bad_state")
		return
	}
	tok, err := h.ttExchange(code)
	if err != nil {
		c.Redirect(http.StatusFound, back+"?err="+url.QueryEscape(err.Error()))
		return
	}
	info, err := h.ttUserInfo(tok.AccessToken)
	if err != nil {
		info = struct {
			OpenID      string `json:"open_id"`
			DisplayName string `json:"display_name"`
			AvatarURL   string `json:"avatar_url"`
		}{OpenID: tok.OpenID}
	}
	now := time.Now().UTC()
	acc := model.TikTokAccount{
		TenantID: tid, OpenID: info.OpenID, DisplayName: info.DisplayName, Avatar: info.AvatarURL,
		AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken,
		AccessExpiresAt: now.Add(time.Duration(tok.ExpiresIn) * time.Second),
		RefreshExpiresAt: now.Add(time.Duration(tok.RefreshExpiresIn) * time.Second),
		Scope: tok.Scope, Status: 1,
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "open_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"display_name", "avatar", "access_token", "refresh_token",
			"access_expires_at", "refresh_expires_at", "scope", "status"}),
	}).Create(&acc).Error; err != nil {
		c.Redirect(http.StatusFound, back+"?err=save_failed")
		return
	}
	c.Redirect(http.StatusFound, back+"?connected=1")
}

type ttTokenResp struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int64  `json:"expires_in"`
	OpenID           string `json:"open_id"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (h *Handler) ttPostForm(endpoint string, form url.Values) (*ttTokenResp, error) {
	req, _ := http.NewRequest("POST", ttAPIBase+"/oauth/"+endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tr ttTokenResp
	json.NewDecoder(resp.Body).Decode(&tr)
	if tr.Error != "" {
		return nil, fmt.Errorf("%s: %s", tr.Error, tr.ErrorDescription)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("empty token (HTTP %d)", resp.StatusCode)
	}
	return &tr, nil
}

func (h *Handler) ttExchange(code string) (*ttTokenResp, error) {
	return h.ttPostForm("token/", url.Values{
		"client_key":    {h.Cfg.TikTok.ClientKey},
		"client_secret": {h.Cfg.TikTok.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {h.Cfg.TikTok.RedirectURL},
	})
}

func (h *Handler) ttRefresh(acc *model.TikTokAccount) error {
	tr, err := h.ttPostForm("token/", url.Values{
		"client_key":    {h.Cfg.TikTok.ClientKey},
		"client_secret": {h.Cfg.TikTok.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {acc.RefreshToken},
	})
	if err != nil {
		acc.Status = 0 // token chain broken — needs reconnect
		h.DB.Model(acc).Updates(map[string]interface{}{"status": 0, "access_token": "", "refresh_token": ""})
		return fmt.Errorf("refresh failed: %v", err)
	}
	now := time.Now().UTC()
	acc.AccessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		acc.RefreshToken = tr.RefreshToken
	}
	acc.AccessExpiresAt = now.Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.RefreshExpiresIn > 0 {
		acc.RefreshExpiresAt = now.Add(time.Duration(tr.RefreshExpiresIn) * time.Second)
	}
	h.DB.Model(acc).Updates(map[string]interface{}{
		"access_token": acc.AccessToken, "refresh_token": acc.RefreshToken,
		"access_expires_at": acc.AccessExpiresAt, "refresh_expires_at": acc.RefreshExpiresAt, "status": 1,
	})
	return nil
}

func (h *Handler) ttAccessToken(acc *model.TikTokAccount) (string, error) {
	if acc.Status == 0 {
		return "", fmt.Errorf("account disconnected — reconnect required")
	}
	if time.Now().UTC().Add(5 * time.Minute).After(acc.AccessExpiresAt) {
		if err := h.ttRefresh(acc); err != nil {
			return "", err
		}
	}
	return acc.AccessToken, nil
}

func (h *Handler) ttUserInfo(token string) (struct {
	OpenID      string `json:"open_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}, error) {
	var out struct {
		OpenID      string `json:"open_id"`
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	}
	req, _ := http.NewRequest("GET", ttAPIBase+"/user/info/?fields=open_id,display_name,avatar_url", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	var r struct {
		Data struct {
			User struct {
				OpenID      string `json:"open_id"`
				DisplayName string `json:"display_name"`
				AvatarURL   string `json:"avatar_url"`
			} `json:"user"`
		} `json:"data"`
		Error struct {
			Code string `json:"code"`
			Msg  string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&r)
	if r.Error.Code != "" && r.Error.Code != "ok" {
		return out, fmt.Errorf("user info: %s", r.Error.Msg)
	}
	out = r.Data.User
	return out, nil
}

// ---------- admin: accounts ----------

// GET /api/admin/tiktok/accounts
func (h *Handler) TikTokAccounts(c *gin.Context) {
	var accs []model.TikTokAccount
	h.DB.Where("tenant_id = ?", h.tID(c)).Order("id DESC").Find(&accs)
	ok(c, accs)
}

// DELETE /api/admin/tiktok/accounts/:id
func (h *Handler) TikTokAccountDelete(c *gin.Context) {
	id := uint64(toInt(c.Param("id"), 0))
	res := h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.TikTokAccount{})
	if res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// ---------- admin: clip library ----------

// GET /api/admin/tiktok/clips
func (h *Handler) TikTokClips(c *gin.Context) {
	var clips []model.TikTokClip
	h.DB.Where("tenant_id = ?", h.tID(c)).Order("id DESC").Limit(500).Find(&clips)
	ok(c, clips)
}

// POST /api/admin/tiktok/clips {title, video_url, size_bytes, duration_sec}
func (h *Handler) TikTokClipSave(c *gin.Context) {
	var p model.TikTokClip
	if err := c.ShouldBindJSON(&p); err != nil || p.VideoURL == "" {
		fail(c, http.StatusBadRequest, "video_url required")
		return
	}
	p.ID = 0
	p.TenantID = h.tID(c)
	if p.Title == "" {
		p.Title = "clip-" + time.Now().UTC().Format("0102-150405")
	}
	if err := h.DB.Create(&p).Error; err != nil {
		fail(c, http.StatusInternalServerError, "save failed")
		return
	}
	ok(c, p)
}

// DELETE /api/admin/tiktok/clips/:id
func (h *Handler) TikTokClipDelete(c *gin.Context) {
	id := uint64(toInt(c.Param("id"), 0))
	res := h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.TikTokClip{})
	if res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// ---------- admin: publish ----------

type ttPublishReq struct {
	AccountIDs   []uint64 `json:"account_ids" binding:"required"`
	ClipIDs      []uint64 `json:"clip_ids" binding:"required"`
	Title        string   `json:"title" binding:"required"`
	PrivacyLevel string   `json:"privacy_level"`
}

// POST /api/admin/tiktok/publish — fan out posts (account × clip) into the queue
func (h *Handler) TikTokPublish(c *gin.Context) {
	var req ttPublishReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "account_ids, clip_ids, title required")
		return
	}
	if req.Title == "" {
		fail(c, http.StatusBadRequest, "title required")
		return
	}
	if len(req.Title) > 2100 {
		req.Title = req.Title[:2100]
	}
	if req.PrivacyLevel == "" {
		req.PrivacyLevel = "SELF_ONLY"
	}
	var accs []model.TikTokAccount
	h.DB.Where("id IN ? AND tenant_id = ? AND status = 1", req.AccountIDs, h.tID(c)).Find(&accs)
	var clips []model.TikTokClip
	h.DB.Where("id IN ? AND tenant_id = ?", req.ClipIDs, h.tID(c)).Find(&clips)
	if len(accs) == 0 || len(clips) == 0 {
		fail(c, http.StatusBadRequest, "no valid accounts or clips")
		return
	}
	created := 0
	for _, a := range accs {
		for _, cl := range clips {
			post := model.TikTokPost{
				TenantID: h.tID(c), AccountID: a.ID, ClipID: cl.ID,
				Title: req.Title, PrivacyLevel: req.PrivacyLevel, Status: "queued",
			}
			if h.DB.Create(&post).Error == nil {
				created++
			}
		}
	}
	kickTTPublisher(h)
	ok(c, gin.H{"queued": created})
}

// GET /api/admin/tiktok/posts
func (h *Handler) TikTokPosts(c *gin.Context) {
	h.ttRefreshProcessing(c)
	var posts []model.TikTokPost
	q := h.DB.Where("tenant_id = ?", h.tID(c))
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	q.Order("id DESC").Limit(200).Find(&posts)
	type row struct {
		model.TikTokPost
		AccountName string `json:"account_name"`
		ClipTitle   string `json:"clip_title"`
		ClipURL     string `json:"clip_url"`
	}
	out := make([]row, 0, len(posts))
	for _, p := range posts {
		var a model.TikTokAccount
		var cl model.TikTokClip
		h.DB.First(&a, p.AccountID)
		h.DB.First(&cl, p.ClipID)
		out = append(out, row{p, a.DisplayName, cl.Title, cl.VideoURL})
	}
	ok(c, out)
}

// lazily move processing posts to their final state (TikTok is async)
func (h *Handler) ttRefreshProcessing(c *gin.Context) {
	var posts []model.TikTokPost
	h.DB.Where("tenant_id = ? AND status = ?", h.tID(c), "processing").Limit(5).Find(&posts)
	for _, p := range posts {
		var acc model.TikTokAccount
		if h.DB.First(&acc, p.AccountID).Error != nil || p.PublishID == "" {
			continue
		}
		tok, err := h.ttAccessToken(&acc)
		if err != nil {
			continue
		}
		body, _ := json.Marshal(map[string]string{"publish_id": p.PublishID})
		req, _ := http.NewRequest("POST", ttAPIBase+"/post/publish/status/fetch/", strings.NewReader(string(body)))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		var r struct {
			Data struct {
				Status    string `json:"status"`
				FailReason struct {
					Code string `json:"code"`
					Msg  string `json:"message"`
				} `json:"fail_reason"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		switch r.Data.Status {
		case "PUBLISH_COMPLETE":
			h.DB.Model(&p).Updates(map[string]interface{}{"status": "published", "error": ""})
		case "FAILED":
			h.DB.Model(&p).Updates(map[string]interface{}{"status": "failed", "error": truncStr(r.Data.FailReason.Code+" "+r.Data.FailReason.Msg, 490)})
		}
	}
}

// ---------- publish worker (serialized queue, mirrors the remux design) ----------

var (
	ttPubMu      sync.Mutex
	ttPubRunning bool
)

func kickTTPublisher(h *Handler) {
	go runTTPublisher(h)
}

func runTTPublisher(h *Handler) {
	ttPubMu.Lock()
	if ttPubRunning {
		ttPubMu.Unlock()
		return
	}
	ttPubRunning = true
	ttPubMu.Unlock()
	defer func() { ttPubMu.Lock(); ttPubRunning = false; ttPubMu.Unlock() }()

	for {
		var post model.TikTokPost
		if err := h.DB.Where("status = ?", "queued").Order("id ASC").First(&post).Error; err != nil {
			return // queue drained
		}
		h.processPost(&post)
	}
}

func (h *Handler) processPost(p *model.TikTokPost) {
	failPost := func(format string, args ...interface{}) {
		h.DB.Model(p).Updates(map[string]interface{}{
			"status": "failed", "error": truncStr(fmt.Sprintf(format, args...), 490)})
	}
	var acc model.TikTokAccount
	if h.DB.First(&acc, p.AccountID).Error != nil {
		failPost("account missing")
		return
	}
	var clip model.TikTokClip
	if h.DB.First(&clip, p.ClipID).Error != nil {
		failPost("clip missing")
		return
	}
	tok, err := h.ttAccessToken(&acc)
	if err != nil {
		failPost("%v", err)
		return
	}

	// download the clip to a temp file (chunked ranged uploads need seeking)
	tmp, err := os.CreateTemp("", "tt-clip-*.mp4")
	if err != nil {
		failPost("temp file: %v", err)
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	defer tmp.Close()
	dl, err := http.Get(clip.VideoURL)
	if err != nil {
		failPost("download clip: %v", err)
		return
	}
	if dl.StatusCode != http.StatusOK {
		dl.Body.Close()
		failPost("download clip: HTTP %d", dl.StatusCode)
		return
	}
	size, err := io.Copy(tmp, dl.Body)
	dl.Body.Close()
	if err != nil {
		failPost("download clip: %v", err)
		return
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		failPost("seek: %v", err)
		return
	}

	h.DB.Model(p).Update("status", "uploading")

	// init
	chunk := int64(ttChunkSize)
	if size < chunk {
		chunk = size
	}
	total := (size + chunk - 1) / chunk
	initBody, _ := json.Marshal(map[string]interface{}{
		"post_info": map[string]interface{}{
			"title": p.Title, "privacy_level": p.PrivacyLevel,
			"disable_comment": false, "disable_duet": false, "disable_stitch": false,
		},
		"source_info": map[string]interface{}{
			"source": "FILE_UPLOAD", "video_size": size,
			"chunk_size": chunk, "total_chunk_count": total,
		},
	})
	req, _ := http.NewRequest("POST", ttAPIBase+"/post/publish/video/init/", strings.NewReader(string(initBody)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		failPost("init: %v", err)
		return
	}
	var ir struct {
		Data struct {
			PublishID string `json:"publish_id"`
			UploadURL string `json:"upload_url"`
		} `json:"data"`
		Error struct {
			Code string `json:"code"`
			Msg  string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&ir)
	resp.Body.Close()
	if ir.Data.UploadURL == "" {
		failPost("init: %s %s (HTTP %d)", ir.Error.Code, ir.Error.Msg, resp.StatusCode)
		return
	}
	h.DB.Model(p).Update("publish_id", ir.Data.PublishID)

	// upload chunks with Content-Range
	buf := make([]byte, chunk)
	for idx := int64(0); idx < total; idx++ {
		n, err := io.ReadFull(tmp, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			failPost("read chunk %d: %v", idx+1, err)
			return
		}
		start := idx * chunk
		end := start + int64(n) - 1
		ureq, _ := http.NewRequest("PUT", ir.Data.UploadURL, strings.NewReader(string(buf[:n])))
		ureq.Header.Set("Content-Type", "video/mp4")
		ureq.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
		ures, err := http.DefaultClient.Do(ureq)
		if err != nil {
			failPost("upload chunk %d: %v", idx+1, err)
			return
		}
		ures.Body.Close()
		if ures.StatusCode >= 300 {
			failPost("upload chunk %d: HTTP %d", idx+1, ures.StatusCode)
			return
		}
	}

	h.DB.Model(p).Updates(map[string]interface{}{"status": "processing", "error": ""})
	h.DB.Model(&model.TikTokClip{ID: clip.ID}).Update("used_at", time.Now().UTC())
	log.Printf("[tiktok] post %d uploaded (publish_id=%s) — processing on TikTok", p.ID, ir.Data.PublishID)
}

func truncStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
