package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"

	"viewly/internal/model"
)

// Creator workspace API (studio.likeviewly.com) — the TKPostAI-clone surface.
// Everything is scoped to the logged-in user (UserAuth middleware) inside the
// resolved tenant. Legacy admin-hosted assets have user_id NULL and never
// show up here.

const tkMaxPostsPerTask = 500

var ttPrivacyLevels = map[string]bool{
	"SELF_ONLY": true, "MUTUAL_FOLLOW": true, "FOLLOWER_OF_CREATOR": true, "PUBLIC_TO_EVERYONE": true,
}

func (h *Handler) tkUser(c *gin.Context) *model.User {
	u, _ := c.Get("user")
	if u == nil {
		return nil
	}
	return u.(*model.User)
}

// ---------- OAuth: direct connect + 7-day authorization links ----------

// POST /api/user/tk/connect-url {title?} → {url} — single-use, 30 min
func (h *Handler) TKUserConnectURL(c *gin.Context) {
	u := h.tkUser(c)
	if h.Cfg.TikTok.ClientKey == "" {
		failBiz(c, 7001, "tiktok not configured")
		return
	}
	st := model.TKAuthState{
		TenantID: h.tID(c), State: "u" + randHex(16), Kind: "user",
		OwnerUserID: u.ID, ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
	}
	if err := h.DB.Create(&st).Error; err != nil {
		failBiz(c, 7301, "create state failed")
		return
	}
	ok(c, gin.H{"url": h.ttAuthorizeLink(st.State), "expires_at": st.ExpiresAt})
}

// POST /api/user/tk/auth-links {title} → {id, url, expires_at}
// Shareable for 7 days: every account authorized through it joins the
// OWNER's matrix (see TikTokCallback).
func (h *Handler) TKUserAuthLinkCreate(c *gin.Context) {
	u := h.tkUser(c)
	var p struct {
		Title string `json:"title"`
	}
	c.ShouldBindJSON(&p)
	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" {
		p.Title = "link-" + time.Now().UTC().Format("0102-150405")
	}
	st := model.TKAuthState{
		TenantID: h.tID(c), State: "l" + randHex(16), Kind: "link",
		OwnerUserID: u.ID, Title: p.Title,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	if err := h.DB.Create(&st).Error; err != nil {
		failBiz(c, 7301, "create link failed")
		return
	}
	ok(c, gin.H{"id": st.ID, "title": st.Title, "url": h.ttAuthorizeLink(st.State), "expires_at": st.ExpiresAt})
}

// GET /api/user/tk/auth-links
func (h *Handler) TKUserAuthLinks(c *gin.Context) {
	u := h.tkUser(c)
	var links []model.TKAuthState
	h.DB.Where("tenant_id = ? AND owner_user_id = ? AND kind = ?", h.tID(c), u.ID, "link").
		Order("id DESC").Limit(100).Find(&links)
	type row struct {
		model.TKAuthState
		URL       string `json:"url"`
		Expired   bool   `json:"expired"`
		ExpiresIn int64  `json:"expires_in_sec"`
	}
	out := make([]row, 0, len(links))
	for _, l := range links {
		r := row{TKAuthState: l, URL: h.ttAuthorizeLink(l.State)}
		r.Expired = l.ExpiresAt.Before(time.Now().UTC())
		r.ExpiresIn = int64(time.Until(l.ExpiresAt).Seconds())
		if r.ExpiresIn < 0 {
			r.ExpiresIn = 0
		}
		out = append(out, r)
	}
	ok(c, out)
}

// DELETE /api/user/tk/auth-links/:id — revoke immediately (expired/revoked
// states are rejected at the callback).
func (h *Handler) TKUserAuthLinkDelete(c *gin.Context) {
	u := h.tkUser(c)
	res := h.DB.Where("id = ? AND tenant_id = ? AND owner_user_id = ?",
		toInt(c.Param("id"), 0), h.tID(c), u.ID).Delete(&model.TKAuthState{})
	if res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// ---------- accounts ----------

// GET /api/user/tk/accounts
func (h *Handler) TKUserAccounts(c *gin.Context) {
	u := h.tkUser(c)
	var accs []model.TikTokAccount
	h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID).Order("id DESC").Find(&accs)
	ok(c, accs)
}

// DELETE /api/user/tk/accounts/:id — also cancels its not-yet-run posts
func (h *Handler) TKUserAccountDelete(c *gin.Context) {
	u := h.tkUser(c)
	id := uint64(toInt(c.Param("id"), 0))
	var acc model.TikTokAccount
	if err := h.DB.Where("id = ? AND tenant_id = ? AND user_id = ?", id, h.tID(c), u.ID).First(&acc).Error; err != nil {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	h.DB.Model(&model.TikTokPost{}).
		Where("account_id = ? AND status IN ?", acc.ID, []string{"scheduled", "queued"}).
		Update("status", "canceled")
	h.DB.Delete(&acc)
	ok(c, gin.H{"deleted": true})
}

// ---------- clips ----------

// GET /api/user/tk/clips
func (h *Handler) TKUserClips(c *gin.Context) {
	u := h.tkUser(c)
	var clips []model.TikTokClip
	h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID).Order("id DESC").Limit(500).Find(&clips)
	ok(c, clips)
}

// POST /api/user/tk/clips {title, video_url, size_bytes, duration_sec}
// video_url comes from the chunked upload endpoints (same flow as admin).
func (h *Handler) TKUserClipSave(c *gin.Context) {
	u := h.tkUser(c)
	var p model.TikTokClip
	if err := c.ShouldBindJSON(&p); err != nil || p.VideoURL == "" {
		fail(c, http.StatusBadRequest, "video_url required")
		return
	}
	p.ID = 0
	p.TenantID = h.tID(c)
	uid := u.ID
	p.UserID = &uid
	if p.Title == "" {
		p.Title = "clip-" + time.Now().UTC().Format("0102-150405")
	}
	if err := h.DB.Create(&p).Error; err != nil {
		fail(c, http.StatusInternalServerError, "save failed")
		return
	}
	ok(c, p)
}

// DELETE /api/user/tk/clips/:id
func (h *Handler) TKUserClipDelete(c *gin.Context) {
	u := h.tkUser(c)
	res := h.DB.Where("id = ? AND tenant_id = ? AND user_id = ?",
		toInt(c.Param("id"), 0), h.tID(c), u.ID).Delete(&model.TikTokClip{})
	if res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// ---------- tasks (batch publish) ----------

type tkTaskReq struct {
	Name           string     `json:"name"`
	AccountIDs     []uint64   `json:"account_ids" binding:"required"`
	ClipIDs        []uint64   `json:"clip_ids" binding:"required"`
	Rule           string     `json:"rule"`
	LoopWhenShort  bool       `json:"loop_when_short"`
	MaxRepeat      int        `json:"max_repeat"`
	Title          string     `json:"title" binding:"required"`
	PrivacyLevel   string     `json:"privacy_level"`
	ScheduledAt    *time.Time `json:"scheduled_at"`
	DisableComment bool       `json:"disable_comment"`
	DisableDuet    bool       `json:"disable_duet"`
	DisableStitch  bool       `json:"disable_stitch"`
}

// loadOwnedAccounts/Clips fetch only the rows the user owns.
func (h *Handler) loadOwnedAccounts(c *gin.Context, ids []uint64) []model.TikTokAccount {
	u := h.tkUser(c)
	var accs []model.TikTokAccount
	h.DB.Where("id IN ? AND tenant_id = ? AND user_id = ? AND status = 1", ids, h.tID(c), u.ID).Find(&accs)
	return accs
}

func (h *Handler) loadOwnedClips(c *gin.Context, ids []uint64) []model.TikTokClip {
	u := h.tkUser(c)
	var clips []model.TikTokClip
	h.DB.Where("id IN ? AND tenant_id = ? AND user_id = ?", ids, h.tID(c), u.ID).Find(&clips)
	return clips
}

func ruleValid(r string) bool {
	return r == RuleOneToOne || r == RuleAllVideos || r == RuleRandomEven
}

func accIDs(accs []model.TikTokAccount) []uint64 {
	out := make([]uint64, len(accs))
	for i, a := range accs {
		out[i] = a.ID
	}
	return out
}

func clipIDs(clips []model.TikTokClip) []uint64 {
	out := make([]uint64, len(clips))
	for i, cl := range clips {
		out[i] = cl.ID
	}
	return out
}

// POST /api/user/tk/tasks/preview → {total, per_account:[{account_id,count}]}
// The wizard calls this on every selection change to show what would happen.
func (h *Handler) TKUserTaskPreview(c *gin.Context) {
	var req tkTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "account_ids, clip_ids required")
		return
	}
	accs := h.loadOwnedAccounts(c, req.AccountIDs)
	clips := h.loadOwnedClips(c, req.ClipIDs)
	if len(accs) == 0 || len(clips) == 0 {
		failBiz(c, 7311, "no valid accounts or clips")
		return
	}
	if !ruleValid(req.Rule) {
		req.Rule = RuleOneToOne
	}
	pairs := allocate(accIDs(accs), clipIDs(clips), req.Rule, req.LoopWhenShort, req.MaxRepeat)
	type pc struct {
		AccountID uint64 `json:"account_id"`
		Count     int    `json:"count"`
	}
	per := make([]pc, 0, len(accs))
	for _, a := range accs {
		n := 0
		for _, p := range pairs {
			if p.AccountID == a.ID {
				n++
			}
		}
		if n > 0 {
			per = append(per, pc{a.ID, n})
		}
	}
	ok(c, gin.H{"total": len(pairs), "per_account": per})
}

// POST /api/user/tk/tasks — create task + fan out posts via the allocation rule
func (h *Handler) TKUserTaskCreate(c *gin.Context) {
	u := h.tkUser(c)
	var req tkTaskReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		fail(c, http.StatusBadRequest, "account_ids, clip_ids, title required")
		return
	}
	if len(req.Title) > 2100 {
		req.Title = req.Title[:2100]
	}
	if !ruleValid(req.Rule) {
		req.Rule = RuleOneToOne
	}
	if req.PrivacyLevel == "" {
		req.PrivacyLevel = "SELF_ONLY"
	}
	if !ttPrivacyLevels[req.PrivacyLevel] {
		fail(c, http.StatusBadRequest, "invalid privacy_level")
		return
	}
	accs := h.loadOwnedAccounts(c, req.AccountIDs)
	clips := h.loadOwnedClips(c, req.ClipIDs)
	if len(accs) == 0 || len(clips) == 0 {
		failBiz(c, 7311, "no valid accounts or clips")
		return
	}
	pairs := allocate(accIDs(accs), clipIDs(clips), req.Rule, req.LoopWhenShort, req.MaxRepeat)
	if len(pairs) == 0 {
		failBiz(c, 7312, "allocation produced nothing")
		return
	}
	if len(pairs) > tkMaxPostsPerTask {
		failBiz(c, 7313, "too many posts (max 500): lower account/clip count or repeat cap")
		return
	}

	clipByID := map[uint64]model.TikTokClip{}
	for _, cl := range clips {
		clipByID[cl.ID] = cl
	}

	status := "queued"
	taskStatus := "running"
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now().UTC()) {
		status = "scheduled"
		taskStatus = "scheduled"
	}
	uid := u.ID
	task := model.TKTask{
		TenantID: h.tID(c), UserID: u.ID, Name: strings.TrimSpace(req.Name),
		Rule: req.Rule, Total: len(pairs), Status: taskStatus,
		ScheduledAt: req.ScheduledAt, PrivacyLevel: req.PrivacyLevel, Title: req.Title,
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		batch := make([]model.TikTokPost, 0, len(pairs))
		for _, p := range pairs {
			cl := clipByID[p.ClipID]
			title := strings.ReplaceAll(req.Title, "{clip}", cl.Title)
			batch = append(batch, model.TikTokPost{
				TenantID: h.tID(c), TaskID: &task.ID, UserID: &uid,
				AccountID: p.AccountID, ClipID: p.ClipID,
				Title: title, PrivacyLevel: req.PrivacyLevel, Status: status,
				ScheduledAt: req.ScheduledAt,
				DisableComment: req.DisableComment, DisableDuet: req.DisableDuet, DisableStitch: req.DisableStitch,
			})
		}
		return tx.Create(&batch).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "create task failed")
		return
	}
	kickTTPublisher(h)
	ok(c, gin.H{"task_id": task.ID, "total": len(pairs)})
}

var tkTerminalPost = map[string]bool{
	"delivered": true, "published": true, "failed": true, "canceled": true,
}

// GET /api/user/tk/tasks — list with per-status counters
func (h *Handler) TKUserTasks(c *gin.Context) {
	// the studio polls this list; use it to wake the worker for due
	// scheduled posts (single-flight — a no-op when already running)
	kickTTPublisher(h)
	u := h.tkUser(c)
	var tasks []model.TKTask
	h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID).Order("id DESC").Limit(100).Find(&tasks)
	if len(tasks) == 0 {
		ok(c, []gin.H{})
		return
	}
	ids := make([]uint64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	type cntRow struct {
		TaskID uint64
		Status string
		N      int
	}
	var cnts []cntRow
	h.DB.Model(&model.TikTokPost{}).Select("task_id, status, COUNT(*) as n").
		Where("task_id IN ?", ids).Group("task_id, status").Scan(&cnts)
	byTask := map[uint64]map[string]int{}
	for _, r := range cnts {
		if byTask[r.TaskID] == nil {
			byTask[r.TaskID] = map[string]int{}
		}
		byTask[r.TaskID][r.Status] = r.N
	}

	type row struct {
		model.TKTask
		Counts map[string]int `json:"counts"`
	}
	out := make([]row, 0, len(tasks))
	for _, t := range tasks {
		counts := byTask[t.ID]
		if counts == nil {
			counts = map[string]int{}
		}
		// lazily finalize: all posts terminal → done
		if t.Status == "running" || t.Status == "scheduled" {
			allTerminal, n := true, 0
			for st, k := range counts {
				n += k
				if !tkTerminalPost[st] {
					allTerminal = false
				}
			}
			if allTerminal && n > 0 {
				t.Status = "done"
				h.DB.Model(&model.TKTask{ID: t.ID}).Update("status", "done")
			}
		}
		out = append(out, row{TKTask: t, Counts: counts})
	}
	ok(c, out)
}

// GET /api/user/tk/tasks/:id — task + its posts (account/clip joined)
func (h *Handler) TKUserTaskDetail(c *gin.Context) {
	u := h.tkUser(c)
	var task model.TKTask
	if err := h.DB.Where("id = ? AND tenant_id = ? AND user_id = ?",
		toInt(c.Param("id"), 0), h.tID(c), u.ID).First(&task).Error; err != nil {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	posts := h.tkPostsJoined(h.DB.Where("task_id = ?", task.ID).Order("id ASC"))
	ok(c, gin.H{"task": task, "posts": posts})
}

// POST /api/user/tk/tasks/:id/cancel — cancels everything not yet picked up
func (h *Handler) TKUserTaskCancel(c *gin.Context) {
	u := h.tkUser(c)
	var task model.TKTask
	if err := h.DB.Where("id = ? AND tenant_id = ? AND user_id = ?",
		toInt(c.Param("id"), 0), h.tID(c), u.ID).First(&task).Error; err != nil {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	h.DB.Model(&model.TikTokPost{}).
		Where("task_id = ? AND status IN ?", task.ID, []string{"scheduled", "queued"}).
		Update("status", "canceled")
	h.DB.Model(&task).Update("status", "canceled")
	ok(c, gin.H{"canceled": true})
}

// POST /api/user/tk/posts/:id/check — mark a delivered draft as human-reviewed
func (h *Handler) TKUserPostCheck(c *gin.Context) {
	u := h.tkUser(c)
	now := time.Now().UTC()
	res := h.DB.Model(&model.TikTokPost{}).
		Where("id = ? AND tenant_id = ? AND user_id = ? AND status = ? AND checked_at IS NULL",
			toInt(c.Param("id"), 0), h.tID(c), u.ID, "delivered").
		Update("checked_at", now)
	if res.RowsAffected == 0 {
		failBiz(c, 7321, "post not found or not in delivered state")
		return
	}
	ok(c, gin.H{"checked": true})
}

// GET /api/user/tk/posts?status=&page= — publish history
func (h *Handler) TKUserPosts(c *gin.Context) {
	u := h.tkUser(c)
	q := h.DB.Where("tenant_id = ? AND user_id = ?", h.tID(c), u.ID)
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	page, size := pageParams(c, 50)
	q.Order("id DESC").Limit(size).Offset(offset(page, size))
	ok(c, h.tkPostsJoined(q))
}

type ttPostRow struct {
	model.TikTokPost
	AccountName string `json:"account_name"`
	ClipTitle   string `json:"clip_title"`
	ClipURL     string `json:"clip_url"`
}

// tkPostsJoined runs the query then attaches account/clip display fields
// (same shape the admin posts endpoint returns).
func (h *Handler) tkPostsJoined(q *gorm.DB) []ttPostRow {
	var posts []model.TikTokPost
	q.Find(&posts)
	out := make([]ttPostRow, 0, len(posts))
	for _, p := range posts {
		var a model.TikTokAccount
		var cl model.TikTokClip
		h.DB.First(&a, p.AccountID)
		h.DB.First(&cl, p.ClipID)
		out = append(out, ttPostRow{p, a.DisplayName, cl.Title, cl.VideoURL})
	}
	return out
}
