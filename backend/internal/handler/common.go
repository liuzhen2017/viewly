package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/middleware"
	"viewly/internal/model"
)

// tenant returns the tenant resolved for this request (never nil once the
// TenantResolve middleware ran).
func (h *Handler) tenant(c *gin.Context) *model.Tenant {
	return middleware.TenantFrom(c)
}

// tID is the tenant id shorthand used by every scoped query.
func (h *Handler) tID(c *gin.Context) uint64 {
	if t := h.tenant(c); t != nil {
		return t.ID
	}
	return 0
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": data})
}

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "msg": msg})
}

func failBiz(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "msg": msg})
}

func pageParams(c *gin.Context, defaultSize int) (int, int) {
	page := toInt(c.Query("page"), 1)
	size := toInt(c.Query("size"), defaultSize)
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = defaultSize
	}
	return page, size
}

func offset(page, size int) int {
	return (page - 1) * size
}

func toInt(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
		if n > 1_000_000_000 {
			return def
		}
	}
	return n
}

// appDay returns the current date (YYYY-MM-DD) in the app timezone,
// which is the boundary for daily tasks such as check-in.
func (h *Handler) appDay(nowUTC time.Time) string {
	return nowUTC.In(h.TZ).Format("2006-01-02")
}

func dbFrom(c *gin.Context) *gorm.DB {
	return c.MustGet("db").(*gorm.DB)
}
