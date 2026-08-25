package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/model"
	"viewly/internal/service"
)

// accessible reports whether the user may play this episode and why.
// Order: free > active VIP > previously unlocked.
func (h *Handler) accessible(u *model.User, e *model.Episode) (bool, string) {
	if e.Status != 1 {
		return false, "episode unavailable"
	}
	if e.PriceCoins == 0 {
		return true, "free"
	}
	if u != nil && u.IsVIP(time.Now().UTC()) {
		return true, "vip"
	}
	if u != nil {
		var cnt int64
		h.DB.Model(&model.EpisodeUnlock{}).Where("user_id = ? AND episode_id = ?", u.ID, e.ID).Count(&cnt)
		if cnt > 0 {
			return true, "unlocked"
		}
	}
	return false, "locked"
}

// GET /api/episodes/:id/play — returns the stream URL if accessible.
// 402 with price info when locked, so the client can show the unlock dialog.
func (h *Handler) Play(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var e model.Episode
	if err := h.DB.Where("tenant_id = ?", h.tID(c)).First(&e, id).Error; err != nil {
		fail(c, http.StatusNotFound, "episode not found")
		return
	}
	u, _ := c.Get("user")
	var user *model.User
	if u != nil {
		user = u.(*model.User)
	}
	acc, reason := h.accessible(user, &e)
	if !acc {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"code": 402, "msg": "locked", "data": gin.H{
				"episode_id": e.ID, "price_coins": e.PriceCoins,
			},
		})
		return
	}
	var d model.Drama
	h.DB.First(&d, e.DramaID)
	h.DB.Model(&model.Drama{}).Where("id = ?", e.DramaID).
		UpdateColumn("views", gorm.Expr("views + 1"))

	// watch-count tasks advance once per episode play for signed-in users
	if user != nil {
		h.bumpTask(user.ID, "watch_5", 1)
		h.bumpTask(user.ID, "watch_10", 1)
		h.bumpTask(user.ID, "watch_20", 1)
	}

	ok(c, gin.H{
		"episode_id": e.ID, "video_url": e.VideoURL, "duration_sec": e.DurationSec,
		"drama_id": e.DramaID, "drama_title": d.Title, "reason": reason,
	})
}

type unlockResp struct {
	EpisodeID uint64 `json:"episode_id"`
	VideoURL  string `json:"video_url"`
	Coins     int64  `json:"coins"`
}

// POST /api/episodes/:id/unlock — spend coins to unlock one episode forever.
func (h *Handler) Unlock(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var e model.Episode
	if err := h.DB.Where("tenant_id = ?", h.tID(c)).First(&e, id).Error; err != nil {
		fail(c, http.StatusNotFound, "episode not found")
		return
	}
	if e.PriceCoins == 0 {
		failBiz(c, 2001, "episode is free")
		return
	}
	if acc, _ := h.accessible(u, &e); acc {
		failBiz(c, 2002, "already unlocked or VIP")
		return
	}

	var out *unlockResp
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var cnt int64
		tx.Model(&model.EpisodeUnlock{}).Where("user_id = ? AND episode_id = ?", u.ID, e.ID).Count(&cnt)
		if cnt > 0 {
			return errors.New("duplicate unlock")
		}
		if _, err := service.Debit(tx, u.ID, int64(e.PriceCoins), "unlock", strconv.FormatUint(e.ID, 10), "unlock episode "+strconv.Itoa(e.EpIndex)); err != nil {
			return err
		}
		rec := model.EpisodeUnlock{UserID: u.ID, DramaID: e.DramaID, EpisodeID: e.ID, CoinsCost: e.PriceCoins}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		var fresh model.User
		tx.First(&fresh, u.ID)
		out = &unlockResp{EpisodeID: e.ID, VideoURL: e.VideoURL, Coins: fresh.Coins}
		return nil
	})
	if err != nil {
		if errors.Is(err, service.ErrInsufficientCoins) {
			c.JSON(http.StatusOK, gin.H{"code": 2003, "msg": "insufficient coins", "data": gin.H{"price_coins": e.PriceCoins, "coins": u.Coins}})
			return
		}
		if err.Error() == "duplicate unlock" {
			failBiz(c, 2002, "already unlocked")
			return
		}
		fail(c, http.StatusInternalServerError, "unlock failed")
		return
	}
	ok(c, out)
}

type progressReq struct {
	PositionSec int `json:"position_sec"`
	DurationSec int `json:"duration_sec"`
}

// POST /api/episodes/:id/progress — continue-watching marker (one per drama).
func (h *Handler) Progress(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var e model.Episode
	if err := h.DB.Where("tenant_id = ?", h.tID(c)).First(&e, id).Error; err != nil {
		fail(c, http.StatusNotFound, "episode not found")
		return
	}
	var req progressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.PositionSec < 0 {
		req.PositionSec = 0
	}
	wp := model.WatchProgress{
		UserID: u.ID, DramaID: e.DramaID, EpisodeID: e.ID,
		PositionSec: req.PositionSec, DurationSec: req.DurationSec,
	}
	h.DB.Where("user_id = ? AND drama_id = ?", u.ID, e.DramaID).
		Assign(wp).FirstOrCreate(&wp)
	h.DB.Create(&model.WatchHistory{UserID: u.ID, DramaID: e.DramaID, EpisodeID: e.ID})
	ok(c, gin.H{"saved": true})
}

// GET /api/history — recently watched dramas with resume points.
func (h *Handler) History(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	type row struct {
		DramaID     uint64 `json:"drama_id"`
		Title       string `json:"title"`
		Cover       string `json:"cover"`
		EpisodeID   uint64 `json:"episode_id"`
		EpIndex     int    `json:"ep_index"`
		PositionSec int    `json:"position_sec"`
		DurationSec int    `json:"duration_sec"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	var rows []row
	h.DB.Raw(`
		SELECT wp.drama_id, d.title, d.cover, wp.episode_id, e.ep_index,
		       wp.position_sec, wp.duration_sec, wp.updated_at
		FROM watch_progress wp
		JOIN dramas d ON d.id = wp.drama_id
		JOIN episodes e ON e.id = wp.episode_id
		WHERE wp.user_id = ?
		ORDER BY wp.updated_at DESC LIMIT 50`, u.ID).Scan(&rows)
	ok(c, rows)
}
