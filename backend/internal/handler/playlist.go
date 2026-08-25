package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/model"
)

// GET /api/favorites — the Playlist tab.
func (h *Handler) FavoriteList(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	list, err := h.cards(h.cardQuery(c).
		Joins("JOIN favorites f ON f.drama_id = d.id AND f.user_id = ?", u.ID).
		Where("d.status = 1").
		Order("f.id DESC"))
	if dbFailed(c, err) {
		return
	}
	ok(c, list)
}

// POST /api/favorites/:dramaID (toggle) — also feeds the favorite task.
func (h *Handler) FavoriteToggle(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	dramaID, _ := strconv.ParseUint(c.Param("dramaID"), 10, 64)
	if dramaID == 0 || !h.dramaExists(c, dramaID) {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var cnt int64
	h.DB.Model(&model.Favorite{}).Where("user_id = ? AND drama_id = ?", u.ID, dramaID).Count(&cnt)
	if cnt > 0 {
		h.DB.Where("user_id = ? AND drama_id = ?", u.ID, dramaID).Delete(&model.Favorite{})
		ok(c, gin.H{"favorited": false})
		return
	}
	h.DB.Create(&model.Favorite{UserID: u.ID, DramaID: dramaID})
	h.bumpTask(u.ID, "favorite", 1)
	ok(c, gin.H{"favorited": true})
}

// POST /api/likes/:dramaID (toggle)
func (h *Handler) LikeToggle(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	dramaID, _ := strconv.ParseUint(c.Param("dramaID"), 10, 64)
	if dramaID == 0 || !h.dramaExists(c, dramaID) {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var cnt int64
	h.DB.Model(&model.DramaLike{}).Where("user_id = ? AND drama_id = ?", u.ID, dramaID).Count(&cnt)
	if cnt > 0 {
		h.DB.Where("user_id = ? AND drama_id = ?", u.ID, dramaID).Delete(&model.DramaLike{})
		h.DB.Model(&model.Drama{}).Where("id = ?", dramaID).UpdateColumn("likes", gorm.Expr("GREATEST(likes - 1, 0)"))
		ok(c, gin.H{"liked": false})
		return
	}
	h.DB.Create(&model.DramaLike{UserID: u.ID, DramaID: dramaID})
	h.DB.Model(&model.Drama{}).Where("id = ?", dramaID).UpdateColumn("likes", gorm.Expr("likes + 1"))
	h.bumpTask(u.ID, "like", 1)
	ok(c, gin.H{"liked": true})
}

// POST /api/follows/:dramaID (toggle)
func (h *Handler) FollowToggle(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	dramaID, _ := strconv.ParseUint(c.Param("dramaID"), 10, 64)
	if dramaID == 0 || !h.dramaExists(c, dramaID) {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var cnt int64
	h.DB.Model(&model.DramaFollow{}).Where("user_id = ? AND drama_id = ?", u.ID, dramaID).Count(&cnt)
	if cnt > 0 {
		h.DB.Where("user_id = ? AND drama_id = ?", u.ID, dramaID).Delete(&model.DramaFollow{})
		ok(c, gin.H{"followed": false})
		return
	}
	h.DB.Create(&model.DramaFollow{UserID: u.ID, DramaID: dramaID})
	ok(c, gin.H{"followed": true})
}

func (h *Handler) dramaExists(c *gin.Context, id uint64) bool {
	var cnt int64
	h.DB.Model(&model.Drama{}).Where("id = ? AND status = 1 AND tenant_id = ?", id, h.tID(c)).Count(&cnt)
	return cnt > 0
}
