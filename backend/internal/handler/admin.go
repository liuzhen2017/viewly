package handler

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"viewly/internal/middleware"
	"viewly/internal/model"
	"viewly/internal/service"
)

type adminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// POST /api/admin/login — tenant admins resolve within the request tenant;
// platform super admins (tenant_id NULL) may sign in from any tenant site.
func (h *Handler) AdminLogin(c *gin.Context) {
	var req adminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "username and password required")
		return
	}
	var a model.AdminUser
	// super admin first: unique username among NULL-tenant rows
	err := h.DB.Where("username = ? AND tenant_id IS NULL AND role = 'super'", req.Username).First(&a).Error
	if err != nil {
		err = h.DB.Where("username = ? AND tenant_id = ?", req.Username, h.tID(c)).First(&a).Error
	}
	if err != nil {
		failBiz(c, 4001, "username or password incorrect")
		return
	}
	if a.Status != 1 || bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(req.Password)) != nil {
		failBiz(c, 4001, "username or password incorrect")
		return
	}
	token, _ := middleware.IssueToken(h.Cfg.Server.JWTSecret, a.ID, "admin")
	ok(c, gin.H{"token": token, "admin": gin.H{
		"id": a.ID, "username": a.Username, "nickname": a.Nickname, "role": a.Role,
	}})
}

// ---------- platform tenant management (super only) ----------

// GET /api/admin/tenants
func (h *Handler) AdminTenantsList(c *gin.Context) {
	var tenants []model.Tenant
	h.DB.Order("id ASC").Find(&tenants)
	type row struct {
		model.Tenant
		Dramas int64 `json:"dramas"`
		Users  int64 `json:"users"`
	}
	out := make([]row, 0, len(tenants))
	for _, t := range tenants {
		var d, u int64
		h.DB.Model(&model.Drama{}).Where("tenant_id = ?", t.ID).Count(&d)
		h.DB.Model(&model.User{}).Where("tenant_id = ?", t.ID).Count(&u)
		out = append(out, row{Tenant: t, Dramas: d, Users: u})
	}
	ok(c, out)
}

type tenantCreateReq struct {
	Name     string `json:"name" binding:"required"`
	Slug     string `json:"slug" binding:"required"`
	Username string `json:"admin_username" binding:"required,min=3,max=64"`
	Password string `json:"admin_password" binding:"required,min=6,max=64"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$`)

// POST /api/admin/tenants — provision a new tenant site with its admin.
func (h *Handler) AdminTenantCreate(c *gin.Context) {
	var req tenantCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "name, slug, admin_username(3+) and admin_password(6+) required")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(slug) {
		failBiz(c, 4002, "slug must be 3-32 chars: lowercase letters, digits, hyphens")
		return
	}
	var cnt int64
	h.DB.Model(&model.Tenant{}).Where("slug = ?", slug).Count(&cnt)
	if cnt > 0 {
		failBiz(c, 4003, "slug already exists")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "hash error")
		return
	}
	t := model.Tenant{Name: req.Name, Slug: slug, Status: 1}
	admin := model.AdminUser{
		Username: req.Username, PasswordHash: string(hash),
		Nickname: req.Name + " Admin", Role: "admin", Status: 1,
	}
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		admin.TenantID = &t.ID
		return tx.Create(&admin).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "create tenant failed (username taken?)")
		return
	}
	middleware.InvalidateTenantCache()
	ok(c, gin.H{"tenant": t, "admin": gin.H{"username": admin.Username}})
}

// ---------- dashboard ----------

// GET /api/admin/stats — dashboard overview for the resolved tenant.
func (h *Handler) AdminStats(c *gin.Context) {
	tid := h.tID(c)
	var users, dramas, episodes, orders, paidOrders int64
	var revenueCents int64
	h.DB.Model(&model.User{}).Where("tenant_id = ?", tid).Count(&users)
	h.DB.Model(&model.Drama{}).Where("tenant_id = ?", tid).Count(&dramas)
	h.DB.Model(&model.Episode{}).Where("tenant_id = ?", tid).Count(&episodes)
	h.DB.Model(&model.Order{}).Where("tenant_id = ?", tid).Count(&orders)
	h.DB.Model(&model.Order{}).Where("tenant_id = ? AND status = 'paid'", tid).Count(&paidOrders)
	h.DB.Model(&model.Order{}).Where("tenant_id = ? AND status = 'paid'", tid).Select("COALESCE(SUM(amount_cents),0)").Scan(&revenueCents)

	today := h.appDay(time.Now().UTC())
	var newUsersToday, checkinsToday, unlocksToday int64
	h.DB.Model(&model.User{}).Where("tenant_id = ? AND created_at >= ?", tid, today+" 00:00:00").Count(&newUsersToday)
	h.DB.Raw(`SELECT COUNT(*) FROM checkin_records r JOIN users u ON u.id = r.user_id
		WHERE u.tenant_id = ? AND r.day_date = ?`, tid, today).Scan(&checkinsToday)
	h.DB.Raw(`SELECT COUNT(*) FROM episode_unlocks x JOIN users u ON u.id = x.user_id
		WHERE u.tenant_id = ? AND x.created_at >= ?`, tid, today+" 00:00:00").Scan(&unlocksToday)

	type dayRow struct {
		Day    string `json:"day"`
		Cents  int64  `json:"cents"`
		Orders int64  `json:"orders"`
	}
	series := []dayRow{}
	h.DB.Raw(`
		SELECT DATE(CONVERT_TZ(o.paid_at, '+00:00', '+05:30')) AS day,
		       COALESCE(SUM(o.amount_cents),0) AS cents, COUNT(*) AS orders
		FROM orders o WHERE o.status='paid' AND o.tenant_id = ?
		  AND o.paid_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)
		GROUP BY day ORDER BY day`, tid).Scan(&series)

	ok(c, gin.H{
		"users": users, "dramas": dramas, "episodes": episodes,
		"orders": orders, "paid_orders": paidOrders, "revenue_cents": revenueCents,
		"new_users_today": newUsersToday, "checkins_today": checkinsToday, "unlocks_today": unlocksToday,
		"revenue_series": series,
	})
}

// ---------- dramas CRUD ----------

type dramaPayload struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	CategoryID  *uint64 `json:"category_id"`
	Cover       string  `json:"cover"`
	Banner      string  `json:"banner"`
	Tags        string  `json:"tags"`
	IsFeatured  *int8   `json:"is_featured"`
	IsCompleted *int8   `json:"is_completed"`
	IsHot       *int8   `json:"is_hot"`
	Status      *int8   `json:"status"`
	Sort        *int    `json:"sort"`
}

func (h *Handler) AdminDramaList(c *gin.Context) {
	page, size := pageParams(c, 20)
	q := h.DB.Table("dramas d").
		Joins("LEFT JOIN categories c ON c.id = d.category_id").
		Where("d.tenant_id = ?", h.tID(c)).
		Select(`d.*, COALESCE(c.name,'') AS category_name,
			(SELECT COUNT(*) FROM episodes e WHERE e.drama_id = d.id) AS episode_count`)
	if kw := c.Query("keyword"); kw != "" {
		q = q.Where("d.title LIKE ?", "%"+kw+"%")
	}
	var total int64
	if dbFailed(c, h.DB.Table("dramas").Where("tenant_id = ?", h.tID(c)).Count(&total).Error) {
		return
	}
	var list []map[string]any
	if dbFailed(c, q.Order("d.id DESC").Offset(offset(page, size)).Limit(size).Find(&list).Error) {
		return
	}
	ok(c, gin.H{"list": list, "page": page, "size": size, "total": total})
}

func (h *Handler) AdminDramaCreate(c *gin.Context) {
	var p dramaPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, http.StatusBadRequest, "title required")
		return
	}
	d := model.Drama{
		TenantID: h.tID(c),
		Title:    p.Title, Description: p.Description, CategoryID: p.CategoryID,
		Cover: p.Cover, Banner: p.Banner, Tags: p.Tags, Status: 1,
	}
	if p.IsFeatured != nil {
		d.IsFeatured = *p.IsFeatured
	}
	if p.IsCompleted != nil {
		d.IsCompleted = *p.IsCompleted
	}
	if p.IsHot != nil {
		d.IsHot = *p.IsHot
	}
	if p.Status != nil {
		d.Status = *p.Status
	}
	if p.Sort != nil {
		d.Sort = *p.Sort
	}
	if err := h.DB.Create(&d).Error; err != nil {
		fail(c, http.StatusInternalServerError, "create failed")
		return
	}
	ok(c, d)
}

func (h *Handler) AdminDramaUpdate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var p dramaPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, http.StatusBadRequest, "title required")
		return
	}
	updates := map[string]any{
		"title": p.Title, "description": p.Description, "category_id": p.CategoryID,
		"cover": p.Cover, "banner": p.Banner, "tags": p.Tags,
	}
	if p.IsFeatured != nil {
		updates["is_featured"] = *p.IsFeatured
	}
	if p.IsCompleted != nil {
		updates["is_completed"] = *p.IsCompleted
	}
	if p.IsHot != nil {
		updates["is_hot"] = *p.IsHot
	}
	if p.Status != nil {
		updates["status"] = *p.Status
	}
	if p.Sort != nil {
		updates["sort"] = *p.Sort
	}
	res := h.DB.Model(&model.Drama{}).Where("id = ? AND tenant_id = ?", id, h.tID(c)).Updates(updates)
	if res.Error != nil {
		fail(c, http.StatusInternalServerError, "update failed")
		return
	}
	if res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var d model.Drama
	h.DB.First(&d, id)
	ok(c, d)
}

func (h *Handler) AdminDramaDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.Drama{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("drama_id = ?", id).Delete(&model.Episode{}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "delete failed")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// ---------- episodes CRUD ----------

type episodePayload struct {
	DramaID     uint64 `json:"drama_id" binding:"required"`
	EpIndex     *int   `json:"ep_index"`
	Title       string `json:"title"`
	VideoURL    string `json:"video_url"`
	DurationSec *int   `json:"duration_sec"`
	PriceCoins  *int   `json:"price_coins"`
	Status      *int8  `json:"status"`
}

// dramaOwnedByTenant is the ownership check every episode admin call needs.
func (h *Handler) dramaOwnedByTenant(tx *gorm.DB, dramaID, tid uint64) bool {
	var cnt int64
	tx.Model(&model.Drama{}).Where("id = ? AND tenant_id = ?", dramaID, tid).Count(&cnt)
	return cnt > 0
}

func (h *Handler) AdminEpisodeList(c *gin.Context) {
	dramaID, _ := strconv.ParseUint(c.Query("drama_id"), 10, 64)
	if !h.dramaOwnedByTenant(h.DB, dramaID, h.tID(c)) {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	var eps []model.Episode
	h.DB.Where("drama_id = ?", dramaID).Order("ep_index ASC").Find(&eps)
	ok(c, eps)
}

func (h *Handler) AdminEpisodeSave(c *gin.Context) {
	var p episodePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, http.StatusBadRequest, "drama_id required")
		return
	}
	if !h.dramaOwnedByTenant(h.DB, p.DramaID, h.tID(c)) {
		fail(c, http.StatusNotFound, "drama not found")
		return
	}
	ep := model.Episode{TenantID: h.tID(c), DramaID: p.DramaID, Title: p.Title, VideoURL: p.VideoURL}
	if p.EpIndex != nil {
		ep.EpIndex = *p.EpIndex
	} else {
		var maxIdx int
		h.DB.Model(&model.Episode{}).Where("drama_id = ?", p.DramaID).Select("COALESCE(MAX(ep_index),0)").Scan(&maxIdx)
		ep.EpIndex = maxIdx + 1
	}
	if p.DurationSec != nil {
		ep.DurationSec = *p.DurationSec
	}
	if p.PriceCoins != nil {
		ep.PriceCoins = *p.PriceCoins
	} else {
		ep.PriceCoins = 0
	}
	if p.Status != nil {
		ep.Status = *p.Status
	} else {
		ep.Status = 1
	}
	var id uint64
	if v := c.Param("id"); v != "" {
		id, _ = strconv.ParseUint(v, 10, 64)
	}
	if id > 0 {
		ep.ID = id
		res := h.DB.Model(&model.Episode{}).Where("id = ? AND tenant_id = ?", id, h.tID(c)).
			Select("drama_id", "ep_index", "title", "video_url", "duration_sec", "price_coins", "status").Updates(&ep)
		if res.Error != nil || res.RowsAffected == 0 {
			fail(c, http.StatusInternalServerError, "update failed")
			return
		}
	} else {
		if err := h.DB.Create(&ep).Error; err != nil {
			fail(c, http.StatusInternalServerError, "create failed (duplicate ep_index?)")
			return
		}
	}
	ok(c, ep)
}

func (h *Handler) AdminEpisodeDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	res := h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.Episode{})
	if res.Error != nil || res.RowsAffected == 0 {
		fail(c, http.StatusInternalServerError, "delete failed")
		return
	}
	ok(c, gin.H{"deleted": true})
}

type episodeBatchPriceReq struct {
	IDs        []uint64 `json:"ids" binding:"required"`
	PriceCoins int      `json:"price_coins"`
}

// PUT /api/admin/episodes/batch-price — set the same coin price on many
// episodes at once (setting 30+ episodes one by one was painful).
func (h *Handler) AdminEpisodeBatchPrice(c *gin.Context) {
	var req episodeBatchPriceReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		fail(c, http.StatusBadRequest, "ids required")
		return
	}
	if req.PriceCoins < 0 {
		fail(c, http.StatusBadRequest, "price_coins must be >= 0")
		return
	}
	res := h.DB.Model(&model.Episode{}).
		Where("id IN ? AND tenant_id = ?", req.IDs, h.tID(c)).
		Update("price_coins", req.PriceCoins)
	if res.Error != nil {
		fail(c, http.StatusInternalServerError, "update failed")
		return
	}
	ok(c, gin.H{"updated": res.RowsAffected})
}

// ---------- categories / banners ----------

func (h *Handler) AdminCategoryList(c *gin.Context) {
	var cats []model.Category
	h.DB.Where("tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&cats)
	ok(c, cats)
}

func (h *Handler) AdminCategorySave(c *gin.Context) {
	var p model.Category
	if err := c.ShouldBindJSON(&p); err != nil || p.Name == "" {
		fail(c, http.StatusBadRequest, "name required")
		return
	}
	if p.Status == 0 {
		p.Status = 1
	}
	if p.ID > 0 {
		res := h.DB.Model(&model.Category{}).Where("id = ? AND tenant_id = ?", p.ID, h.tID(c)).
			Select("name", "sort", "status").Updates(&p)
		if res.Error != nil || res.RowsAffected == 0 {
			fail(c, http.StatusInternalServerError, "update failed")
			return
		}
	} else {
		p.TenantID = h.tID(c)
		if err := h.DB.Where("name = ? AND tenant_id = ?", p.Name, p.TenantID).FirstOrCreate(&p).Error; err != nil {
			fail(c, http.StatusInternalServerError, "save failed")
			return
		}
	}
	ok(c, p)
}

func (h *Handler) AdminCategoryDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.Category{})
	ok(c, gin.H{"deleted": true})
}

func (h *Handler) AdminBannerList(c *gin.Context) {
	var bs []model.Banner
	h.DB.Where("tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&bs)
	ok(c, bs)
}

func (h *Handler) AdminBannerSave(c *gin.Context) {
	var p model.Banner
	if err := c.ShouldBindJSON(&p); err != nil || p.Image == "" {
		fail(c, http.StatusBadRequest, "image required")
		return
	}
	if p.Status == 0 {
		p.Status = 1
	}
	if p.ID > 0 {
		res := h.DB.Model(&model.Banner{}).Where("id = ? AND tenant_id = ?", p.ID, h.tID(c)).
			Select("image", "link", "drama_id", "sort", "status").Updates(&p)
		if res.Error != nil || res.RowsAffected == 0 {
			fail(c, http.StatusInternalServerError, "update failed")
			return
		}
	} else {
		p.TenantID = h.tID(c)
		if err := h.DB.Create(&p).Error; err != nil {
			fail(c, http.StatusInternalServerError, "save failed")
			return
		}
	}
	ok(c, p)
}

func (h *Handler) AdminBannerDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.Banner{})
	ok(c, gin.H{"deleted": true})
}

// ---------- packages ----------

func (h *Handler) AdminPackageList(c *gin.Context) {
	var packs []model.CoinPackage
	h.DB.Where("tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&packs)
	var plans []model.VipPlan
	h.DB.Where("tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&plans)
	ok(c, gin.H{"coin_packages": packs, "vip_plans": plans})
}

func (h *Handler) AdminPackageSave(c *gin.Context) {
	kind := c.Param("kind") // coins | vip
	if kind == "coins" {
		var p model.CoinPackage
		if err := c.ShouldBindJSON(&p); err != nil || p.Coins <= 0 || p.PriceCents <= 0 {
			fail(c, http.StatusBadRequest, "coins and price_cents required")
			return
		}
		if p.Status == 0 {
			p.Status = 1
		}
		if p.Currency == "" {
			p.Currency = "USD"
		}
		if p.ID > 0 {
			res := h.DB.Model(&model.CoinPackage{}).Where("id = ? AND tenant_id = ?", p.ID, h.tID(c)).
				Select("coins", "bonus_coins", "price_cents", "currency", "label", "tag", "sort", "status").Updates(&p)
			if res.Error != nil || res.RowsAffected == 0 {
				fail(c, http.StatusInternalServerError, "update failed")
				return
			}
		} else {
			p.TenantID = h.tID(c)
			if err := h.DB.Create(&p).Error; err != nil {
				fail(c, http.StatusInternalServerError, "save failed")
				return
			}
		}
		ok(c, p)
		return
	}
	var p model.VipPlan
	if err := c.ShouldBindJSON(&p); err != nil || p.Days <= 0 || p.PriceCents <= 0 {
		fail(c, http.StatusBadRequest, "days and price_cents required")
		return
	}
	if p.Status == 0 {
		p.Status = 1
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.ID > 0 {
		res := h.DB.Model(&model.VipPlan{}).Where("id = ? AND tenant_id = ?", p.ID, h.tID(c)).
			Select("days", "price_cents", "currency", "label", "tag", "sort", "status").Updates(&p)
		if res.Error != nil || res.RowsAffected == 0 {
			fail(c, http.StatusInternalServerError, "update failed")
			return
		}
	} else {
		p.TenantID = h.tID(c)
		if err := h.DB.Create(&p).Error; err != nil {
			fail(c, http.StatusInternalServerError, "save failed")
			return
		}
	}
	ok(c, p)
}

func (h *Handler) AdminPackageDelete(c *gin.Context) {
	kind, id := c.Param("kind"), c.Param("id")
	if kind == "coins" {
		h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.CoinPackage{})
	} else {
		h.DB.Where("id = ? AND tenant_id = ?", id, h.tID(c)).Delete(&model.VipPlan{})
	}
	ok(c, gin.H{"deleted": true})
}

// ---------- users & orders ----------

func (h *Handler) AdminUserList(c *gin.Context) {
	page, size := pageParams(c, 20)
	q := h.DB.Model(&model.User{}).Where("tenant_id = ?", h.tID(c))
	if kw := c.Query("keyword"); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("nickname LIKE ? OR email LIKE ?", like, like)
	}
	var total int64
	if dbFailed(c, q.Count(&total).Error) {
		return
	}
	var users []model.User
	if dbFailed(c, q.Order("id DESC").Offset(offset(page, size)).Limit(size).Find(&users).Error) {
		return
	}
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		v := h.userView(&u)
		v["is_guest"] = u.Email == nil
		out = append(out, v)
	}
	ok(c, gin.H{"list": out, "page": page, "size": size, "total": total})
}

type adjustReq struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Coins  int64  `json:"coins" binding:"required"`
	Remark string `json:"remark"`
}

// POST /api/admin/users/adjust — manual coin adjustment with ledger entry.
func (h *Handler) AdminAdjustCoins(c *gin.Context) {
	var req adjustReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "user_id and coins required")
		return
	}
	// the target user must belong to the caller's tenant
	var cnt int64
	h.DB.Model(&model.User{}).Where("id = ? AND tenant_id = ?", req.UserID, h.tID(c)).Count(&cnt)
	if cnt == 0 {
		fail(c, http.StatusNotFound, "user not found")
		return
	}
	var balance int64
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var fresh *model.User
		var err error
		if req.Coins >= 0 {
			fresh, err = service.Credit(tx, req.UserID, req.Coins, "admin", "", req.Remark)
		} else {
			fresh, err = service.Debit(tx, req.UserID, -req.Coins, "admin", "", req.Remark)
		}
		if err != nil {
			return err
		}
		balance = fresh.Coins
		return nil
	})
	if err != nil {
		if err == service.ErrInsufficientCoins {
			failBiz(c, 2003, "insufficient coins")
			return
		}
		fail(c, http.StatusInternalServerError, "adjust failed")
		return
	}
	ok(c, gin.H{"balance": balance})
}

func (h *Handler) AdminOrderList(c *gin.Context) {
	page, size := pageParams(c, 20)
	q := h.DB.Model(&model.Order{}).Where("tenant_id = ?", h.tID(c))
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var total int64
	if dbFailed(c, q.Count(&total).Error) {
		return
	}
	var list []model.Order
	if dbFailed(c, q.Order("id DESC").Offset(offset(page, size)).Limit(size).Find(&list).Error) {
		return
	}
	ok(c, gin.H{"list": list, "page": page, "size": size, "total": total})
}

// POST /api/admin/orders/:orderNo/mark-paid — manual settlement (e.g. bank transfer).
func (h *Handler) AdminMarkPaid(c *gin.Context) {
	var order model.Order
	if err := h.DB.Where("order_no = ? AND tenant_id = ?", c.Param("orderNo"), h.tID(c)).First(&order).Error; err != nil {
		fail(c, http.StatusNotFound, "order not found")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		return settleOrder(tx, &order)
	})
	if err != nil && err.Error() != "order already settled" {
		fail(c, http.StatusInternalServerError, "settle failed")
		return
	}
	ok(c, gin.H{"order_no": order.OrderNo, "status": "paid"})
}
