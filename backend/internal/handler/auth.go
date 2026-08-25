package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"viewly/internal/middleware"
	"viewly/internal/model"
	"viewly/internal/service"
)

type guestReq struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// POST /api/auth/guest — anonymous login keyed by device id. Creates the user
// on first sight and grants the signup bonus.
func (h *Handler) GuestLogin(c *gin.Context) {
	var req guestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "device_id required")
		return
	}
	key := req.DeviceID
	if len(key) > 64 {
		key = key[:64]
	}
	tid := h.tID(c)

	var user model.User
	err := h.DB.Where("guest_key = ? AND tenant_id = ?", key, tid).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		nick := fmt.Sprintf("guest_%s", randSuffix())
		user = model.User{TenantID: tid, GuestKey: &key, Nickname: nick, Status: 1, Language: "en"}
		err = h.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		if h.Cfg.App.SignupBonus > 0 {
			fresh, err := service.Credit(tx, user.ID, h.Cfg.App.SignupBonus, "signup", fmt.Sprintf("%d", user.ID), "signup bonus")
			if err != nil {
				return err
			}
			user.Coins = fresh.Coins
		}
		return nil
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "create user failed")
		return
	}
} else if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if user.Status != 1 {
		fail(c, http.StatusForbidden, "account banned")
		return
	}
	token, err := middleware.IssueToken(h.Cfg.Server.JWTSecret, user.ID, "user")
	if err != nil {
		fail(c, http.StatusInternalServerError, "token error")
		return
	}
	ok(c, gin.H{"token": token, "user": h.userView(&user)})
}

type emailReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

// POST /api/auth/register
func (h *Handler) EmailRegister(c *gin.Context) {
	var req emailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "email and password (6-64 chars) required")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var count int64
	h.DB.Model(&model.User{}).Where("email = ? AND tenant_id = ?", email, h.tID(c)).Count(&count)
	if count > 0 {
		failBiz(c, 1001, "email already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "hash error")
		return
	}
	hashStr := string(hash)
	user := model.User{TenantID: h.tID(c), Email: &email, PasswordHash: &hashStr, Nickname: emailPrefix(email), Status: 1, Language: "en"}
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if h.Cfg.App.SignupBonus > 0 {
			_, err := service.Credit(tx, user.ID, h.Cfg.App.SignupBonus, "signup", fmt.Sprintf("%d", user.ID), "signup bonus")
			return err
		}
		return nil
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "create user failed")
		return
	}
	token, _ := middleware.IssueToken(h.Cfg.Server.JWTSecret, user.ID, "user")
	ok(c, gin.H{"token": token, "user": h.userView(&user)})
}

// POST /api/auth/login
func (h *Handler) EmailLogin(c *gin.Context) {
	var req emailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "email and password required")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user model.User
	if err := h.DB.Where("email = ? AND tenant_id = ?", email, h.tID(c)).First(&user).Error; err != nil {
		failBiz(c, 1002, "email or password incorrect")
		return
	}
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)) != nil {
		failBiz(c, 1002, "email or password incorrect")
		return
	}
	if user.Status != 1 {
		fail(c, http.StatusForbidden, "account banned")
		return
	}
	token, _ := middleware.IssueToken(h.Cfg.Server.JWTSecret, user.ID, "user")
	ok(c, gin.H{"token": token, "user": h.userView(&user)})
}

type bindReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

// POST /api/auth/bind — upgrade a guest account to a permanent email account.
func (h *Handler) BindEmail(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	if u.Email != nil {
		failBiz(c, 1003, "already bound to an email")
		return
	}
	var req bindReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "email and password (6-64 chars) required")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var count int64
	h.DB.Model(&model.User{}).Where("email = ? AND tenant_id = ?", email, h.tID(c)).Count(&count)
	if count > 0 {
		failBiz(c, 1001, "email already registered")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "hash error")
		return
	}
	if err := h.DB.Model(&model.User{}).Where("id = ?", u.ID).
		Updates(map[string]any{"email": email, "password_hash": string(hash)}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "bind failed")
		return
	}
	ok(c, h.userView(u))
}

// GET /api/user/me
func (h *Handler) Me(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	ok(c, h.userView(u))
}

type profileReq struct {
	Nickname *string `json:"nickname" binding:"omitempty,min=1,max=64"`
	Avatar   *string `json:"avatar" binding:"omitempty,max=500"`
	Language *string `json:"language" binding:"omitempty,max=8"`
}

// PUT /api/user/me
func (h *Handler) UpdateMe(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid profile payload")
		return
	}
	updates := map[string]any{}
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	if len(updates) > 0 {
		if err := h.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(updates).Error; err != nil {
			fail(c, http.StatusInternalServerError, "update failed")
			return
		}
	}
	var fresh model.User
	h.DB.First(&fresh, u.ID)
	ok(c, h.userView(&fresh))
}

func (h *Handler) userView(u *model.User) gin.H {
	isVip := u.IsVIP(time.Now().UTC())
	email := ""
	if u.Email != nil {
		email = *u.Email
	}
	return gin.H{
		"id": u.ID, "nickname": u.Nickname, "avatar": u.Avatar, "email": email,
		"coins": u.Coins, "is_vip": isVip,
		"vip_expire_at": u.VipExpireAt, "language": u.Language, "created_at": u.CreatedAt,
	}
}

func emailPrefix(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}

func randSuffix() string {
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)
}
