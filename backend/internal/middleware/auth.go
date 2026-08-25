package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"viewly/internal/model"
)

type Claims struct {
	UID uint64 `json:"uid"`
	jwt.RegisteredClaims
}

func IssueToken(secret string, uid uint64, audience string) (string, error) {
	claims := Claims{
		UID: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(90 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func parse(secret, audience string, raw string) (*Claims, bool) {
	tok, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, false
	}
	c, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, false
	}
	audOk := false
	for _, a := range c.Audience {
		if a == audience {
			audOk = true
			break
		}
	}
	if !audOk {
		return nil, false
	}
	return c, true
}

// ---------- tenant resolution ----------

type tenantCache struct {
	mu    sync.RWMutex
	bySlug map[string]*model.Tenant
}

var tc = &tenantCache{bySlug: map[string]*model.Tenant{}}

func cacheTenant(t *model.Tenant) {
	tc.mu.Lock()
	tc.bySlug[t.Slug] = t
	tc.mu.Unlock()
}

// InvalidateTenantCache drops cached tenants so the next resolve re-reads
// (called after a tenant is created/updated by the platform admin).
func InvalidateTenantCache() {
	tc.mu.Lock()
	tc.bySlug = map[string]*model.Tenant{}
	tc.mu.Unlock()
}

// TenantResolve maps X-Tenant-Slug (or the configured default) to a tenant row
// and stores it as "tenant". Unknown or disabled tenants are rejected.
func TenantResolve(defaultSlug string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.GetHeader("X-Tenant-Slug"))
		if slug == "" {
			slug = defaultSlug
		}

		tc.mu.RLock()
		t := tc.bySlug[slug]
		tc.mu.RUnlock()
		if t == nil {
			var row model.Tenant
			if err := db.Where("slug = ?", slug).First(&row).Error; err != nil {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "unknown tenant"})
				return
			}
			cacheTenant(&row)
			t = &row
		}
		if t.Status != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "tenant disabled"})
			return
		}
		c.Set("tenant", t)
		c.Next()
	}
}

// TenantFrom returns the resolved tenant for this request.
func TenantFrom(c *gin.Context) *model.Tenant {
	t, _ := c.Get("tenant")
	ten, _ := t.(*model.Tenant)
	return ten
}

// ---------- user auth ----------

// UserAuth validates a user JWT, loads the user, and refuses tokens issued
// for a different tenant than the one resolved for this request.
func UserAuth(secret string, loadUser func(uid uint64) (*model.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearer(c)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing token"})
			return
		}
		claims, ok := parse(secret, "user", raw)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			return
		}
		u, err := loadUser(claims.UID)
		if err != nil || u == nil || u.Status != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "user unavailable"})
			return
		}
		if ten := TenantFrom(c); ten != nil && u.TenantID != ten.ID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "cross-tenant token"})
			return
		}
		c.Set("user", u)
		c.Next()
	}
}

// OptionalUserAuth attaches the user identity when a valid, tenant-matching
// token is present but lets anonymous requests through.
func OptionalUserAuth(secret string, loadUser func(uid uint64) (*model.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearer(c)
		if raw == "" {
			c.Next()
			return
		}
		if claims, ok := parse(secret, "user", raw); ok {
			if u, err := loadUser(claims.UID); err == nil && u.Status == 1 {
				if ten := TenantFrom(c); ten == nil || u.TenantID == ten.ID {
					c.Set("user", u)
				}
			}
		}
		c.Next()
	}
}

// ---------- admin auth ----------

// AdminAuth validates an admin JWT. Platform super admins (tenant_id NULL)
// pass for any tenant; tenant admins must match the resolved tenant.
func AdminAuth(secret string, loadAdmin func(id uint64) (*model.AdminUser, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearer(c)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing token"})
			return
		}
		claims, ok := parse(secret, "admin", raw)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			return
		}
		a, err := loadAdmin(claims.UID)
		if err != nil || a == nil || a.Status != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "admin unavailable"})
			return
		}
		if a.Role != "super" {
			if ten := TenantFrom(c); ten == nil || a.TenantID == nil || *a.TenantID != ten.ID {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "cross-tenant admin"})
				return
			}
		}
		c.Set("admin", a)
		c.Next()
	}
}

// RequireSuper aborts unless the caller is a platform super admin.
func RequireSuper() gin.HandlerFunc {
	return func(c *gin.Context) {
		a := AdminFrom(c)
		if a == nil || a.Role != "super" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "super admin only"})
			return
		}
		c.Next()
	}
}

// AdminFrom returns the authenticated admin for this request.
func AdminFrom(c *gin.Context) *model.AdminUser {
	a, _ := c.Get("admin")
	admin, _ := a.(*model.AdminUser)
	return admin
}

func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Tenant-Slug")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
