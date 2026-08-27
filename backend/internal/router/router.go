package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/config"
	"viewly/internal/handler"
	"viewly/internal/middleware"
	"viewly/internal/model"
)

func loadUser(db *gorm.DB) func(uint64) (*model.User, error) {
	return func(uid uint64) (*model.User, error) {
		var u model.User
		if err := db.First(&u, uid).Error; err != nil {
			return nil, err
		}
		return &u, nil
	}
}

func loadAdmin(db *gorm.DB) func(uint64) (*model.AdminUser, error) {
	return func(id uint64) (*model.AdminUser, error) {
		var a model.AdminUser
		if err := db.First(&a, id).Error; err != nil {
			return nil, err
		}
		return &a, nil
	}
}

func Setup(db *gorm.DB, cfg *config.Config, h *handler.Handler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	r.MaxMultipartMemory = 8 << 20

	// local static assets (seed posters etc.)
	r.Static("/static", "./static")

	// per-tenant AdSense authorization file, resolved from the Host header
	r.GET("/ads.txt", h.AdsTxt)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	api.Use(func(c *gin.Context) { c.Set("db", db); c.Next() })
	api.Use(middleware.TenantResolve("main", db))

	// ---- public ----
	auth := api.Group("/auth")
	{
		auth.POST("/guest", h.GuestLogin)
		auth.POST("/register", h.EmailRegister)
		auth.POST("/login", h.EmailLogin)
	}

	optional := api.Group("", middleware.OptionalUserAuth(cfg.Server.JWTSecret, loadUser(db)))
	{
		optional.GET("/home", h.Home)
		optional.GET("/categories", h.Categories)
		optional.GET("/dramas", h.DramaList)
		optional.GET("/dramas/:id", h.DramaDetail)
		optional.GET("/search", h.Search)
		optional.GET("/feed", h.Feed)
		optional.GET("/store", h.Store)
		optional.GET("/ad-config", h.AdConfig)
		optional.GET("/episodes/:id/play", h.Play)
	}

	// ---- authenticated user ----
	user := api.Group("", middleware.UserAuth(cfg.Server.JWTSecret, loadUser(db)))
	{
		user.GET("/user/me", h.Me)
		user.PUT("/user/me", h.UpdateMe)
		user.POST("/auth/bind", h.BindEmail)

		user.POST("/episodes/:id/unlock", h.Unlock)
		user.POST("/episodes/:id/progress", h.Progress)
		user.GET("/history", h.History)

		user.GET("/rewards", h.Rewards)
		user.POST("/rewards/checkin", h.Checkin)
		user.POST("/rewards/tasks/:key/claim", h.ClaimTask)
		user.POST("/rewards/tasks/share/progress", h.ShareProgress)
		user.POST("/rewards/watch-ad/complete", h.WatchAdComplete)

		// payment/ad provider webhooks are public; auth is signature-based
		api.POST("/webhooks/stripe", h.StripeWebhook)
		api.GET("/webhooks/admob/ssv", h.AdMobSSV)

		user.GET("/wallet", h.Wallet)
		user.GET("/wallet/transactions", h.Transactions)
		user.POST("/orders", h.CreateOrder)
		user.GET("/orders", h.MyOrders)
		user.GET("/orders/:order_no", h.OrderStatus)
		user.POST("/orders/:order_no/mock-pay", h.MockPay)

		user.GET("/favorites", h.FavoriteList)
		user.POST("/favorites/:dramaID", h.FavoriteToggle)
		user.POST("/likes/:dramaID", h.LikeToggle)
		user.POST("/follows/:dramaID", h.FollowToggle)
	}

	// ---- admin ----
	admin := api.Group("/admin")
	{
		admin.POST("/login", h.AdminLogin)
	}
	adminAuth := api.Group("/admin", middleware.AdminAuth(cfg.Server.JWTSecret, loadAdmin(db)))
	{
		adminAuth.GET("/stats", h.AdminStats)

		// platform-level: provision tenant sites (super admin only)
		adminAuth.GET("/tenants", middleware.RequireSuper(), h.AdminTenantsList)
		adminAuth.POST("/tenants", middleware.RequireSuper(), h.AdminTenantCreate)

		// ad monetization settings for the managed tenant
		adminAuth.GET("/ad-settings", h.AdSettingsGet)
		adminAuth.PUT("/ad-settings", h.AdSettingsUpdate)

		// browser-direct S3 uploads (presigned)
		adminAuth.POST("/uploads/presign", h.PresignUpload)

		adminAuth.GET("/dramas", h.AdminDramaList)
		adminAuth.POST("/dramas", h.AdminDramaCreate)
		adminAuth.PUT("/dramas/:id", h.AdminDramaUpdate)
		adminAuth.DELETE("/dramas/:id", h.AdminDramaDelete)

		adminAuth.GET("/episodes", h.AdminEpisodeList)
		adminAuth.POST("/episodes", h.AdminEpisodeSave)
		adminAuth.PUT("/episodes/:id", h.AdminEpisodeSave)
		adminAuth.DELETE("/episodes/:id", h.AdminEpisodeDelete)

		adminAuth.GET("/categories", h.AdminCategoryList)
		adminAuth.POST("/categories", h.AdminCategorySave)
		adminAuth.DELETE("/categories/:id", h.AdminCategoryDelete)

		adminAuth.GET("/banners", h.AdminBannerList)
		adminAuth.POST("/banners", h.AdminBannerSave)
		adminAuth.DELETE("/banners/:id", h.AdminBannerDelete)

		adminAuth.GET("/packages", h.AdminPackageList)
		adminAuth.POST("/packages/:kind", h.AdminPackageSave)
		adminAuth.DELETE("/packages/:kind/:id", h.AdminPackageDelete)

		adminAuth.GET("/users", h.AdminUserList)
		adminAuth.POST("/users/adjust", h.AdminAdjustCoins)
		adminAuth.GET("/orders", h.AdminOrderList)
		adminAuth.POST("/orders/:orderNo/mark-paid", h.AdminMarkPaid)
	}

	return r
}
