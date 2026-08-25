package model

import "time"

// Tenant is one operator site on the shared-table platform. Content, users,
// orders and admins are all scoped by TenantID. Ad fields hold each tenant's
// own monetization accounts (AdSense for web display, AdMob for app rewarded).
type Tenant struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64" json:"name"`
	Slug      string    `gorm:"size:32;uniqueIndex" json:"slug"`
	Status    int8      `json:"status"`

	AdsenseClient      string `gorm:"size:64" json:"adsense_client"`
	AdsenseEnabled     int8   `json:"adsense_enabled"`
	RewardedAdMode     string `gorm:"size:8" json:"rewarded_ad_mode"` // off | client | ssv
	RewardedAdCoins    int    `json:"rewarded_ad_coins"`
	RewardedAdDailyLimit int  `json:"rewarded_ad_daily_limit"`
	AdmobAppID         string `gorm:"size:64" json:"admob_app_id"`
	AdmobRewardedUnitID string `gorm:"size:64" json:"admob_rewarded_unit_id"`

	CreatedAt time.Time `json:"created_at"`
}

func (Tenant) TableName() string { return "tenants" }

// AdSSVTransaction dedupes AdMob server-side verification callbacks so a
// replayed callback URL can never credit twice.
type AdSSVTransaction struct {
	TransactionID string    `gorm:"primaryKey;size:64" json:"transaction_id"`
	UserID        uint64    `json:"user_id"`
	TenantID      uint64    `json:"tenant_id"`
	Coins         int       `json:"coins"`
	CreatedAt     time.Time `json:"created_at"`
}

func (AdSSVTransaction) TableName() string { return "ad_ssv_transactions" }

type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `json:"tenant_id"`
	GuestKey     *string   `gorm:"size:64" json:"-"`
	Email        *string   `gorm:"size:191" json:"email"`
	PasswordHash *string   `gorm:"size:191" json:"-"`
	Nickname     string    `gorm:"size:64" json:"nickname"`
	Avatar       string    `gorm:"size:500" json:"avatar"`
	Coins        int64     `json:"coins"`
	VipExpireAt  *time.Time `json:"vip_expire_at"`
	Status       int8      `json:"-"`
	Language     string    `gorm:"size:8" json:"language"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

func (u *User) IsVIP(now time.Time) bool {
	return u.VipExpireAt != nil && u.VipExpireAt.After(now)
}

type CoinTransaction struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	UserID       uint64    `json:"user_id"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	BizType      string    `gorm:"size:32" json:"biz_type"`
	BizID        string    `gorm:"size:64" json:"biz_id"`
	Remark       string    `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
}

func (CoinTransaction) TableName() string { return "coin_transactions" }

type Category struct {
	ID       uint64    `gorm:"primaryKey" json:"id"`
	TenantID uint64    `json:"tenant_id"`
	Name     string    `gorm:"size:64" json:"name"`
	Sort     int       `json:"sort"`
	Status   int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (Category) TableName() string { return "categories" }

type Drama struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `json:"tenant_id"`
	Title       string    `gorm:"size:191" json:"title"`
	Description string    `json:"description"`
	CategoryID  *uint64   `json:"category_id"`
	Cover       string    `gorm:"size:500" json:"cover"`
	Banner      string    `gorm:"size:500" json:"banner"`
	Tags        string    `gorm:"size:255" json:"tags"`
	IsFeatured  int8      `json:"is_featured"`
	IsCompleted int8      `json:"is_completed"`
	IsHot       int8      `json:"is_hot"`
	Status      int8      `json:"status"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Drama) TableName() string { return "dramas" }

type Episode struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `json:"tenant_id"`
	DramaID     uint64    `json:"drama_id"`
	EpIndex     int       `json:"ep_index"`
	Title       string    `gorm:"size:191" json:"title"`
	VideoURL    string    `gorm:"size:1000" json:"video_url"`
	DurationSec int       `json:"duration_sec"`
	PriceCoins  int       `json:"price_coins"`
	Status      int8      `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Episode) TableName() string { return "episodes" }

type EpisodeUnlock struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `json:"user_id"`
	DramaID    uint64    `json:"drama_id"`
	EpisodeID  uint64    `json:"episode_id"`
	CoinsCost  int       `json:"coins_cost"`
	CreatedAt  time.Time `json:"created_at"`
}

func (EpisodeUnlock) TableName() string { return "episode_unlocks" }

type Banner struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `json:"tenant_id"`
	Image     string    `gorm:"size:500" json:"image"`
	DramaID   uint64    `json:"drama_id"`
	Sort      int       `json:"sort"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (Banner) TableName() string { return "banners" }

type CoinPackage struct {
	ID         uint64 `gorm:"primaryKey" json:"id"`
	TenantID   uint64 `json:"tenant_id"`
	Coins      int    `json:"coins"`
	BonusCoins int    `json:"bonus_coins"`
	PriceCents int    `json:"price_cents"`
	Currency   string `gorm:"size:8" json:"currency"`
	Label      string `gorm:"size:64" json:"label"`
	Tag        string `gorm:"size:32" json:"tag"`
	Sort       int    `json:"sort"`
	Status     int8   `json:"status"`
}

func (CoinPackage) TableName() string { return "coin_packages" }

type VipPlan struct {
	ID         uint64 `gorm:"primaryKey" json:"id"`
	TenantID   uint64 `json:"tenant_id"`
	Days       int    `json:"days"`
	PriceCents int    `json:"price_cents"`
	Currency   string `gorm:"size:8" json:"currency"`
	Label      string `gorm:"size:64" json:"label"`
	Tag        string `gorm:"size:32" json:"tag"`
	Sort       int    `json:"sort"`
	Status     int8   `json:"status"`
}

func (VipPlan) TableName() string { return "vip_plans" }

type Order struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	TenantID    uint64     `json:"tenant_id"`
	OrderNo     string     `gorm:"size:64;uniqueIndex" json:"order_no"`
	UserID      uint64     `json:"user_id"`
	Kind        string     `gorm:"size:8" json:"kind"` // coins | vip
	PackageID   uint64     `json:"package_id"`
	Coins       int        `json:"coins"`
	BonusCoins  int        `json:"bonus_coins"`
	Days        int        `json:"days"`
	AmountCents int        `json:"amount_cents"`
	Currency    string     `gorm:"size:8" json:"currency"`
	Status      string     `gorm:"size:16" json:"status"` // pending | paid | failed | closed
	PaidAt      *time.Time `json:"paid_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Order) TableName() string { return "orders" }

type CheckinRecord struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	DayDate   string    `gorm:"size:10" json:"day_date"`
	CycleDay  int       `json:"cycle_day"`
	Coins     int       `json:"coins"`
	CreatedAt time.Time `json:"created_at"`
}

func (CheckinRecord) TableName() string { return "checkin_records" }

type TaskRecord struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	TaskKey   string    `gorm:"size:32" json:"task_key"`
	DayDate   string    `gorm:"size:10" json:"day_date"`
	Progress  int       `json:"progress"`
	Rewarded  int8      `json:"rewarded"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TaskRecord) TableName() string { return "task_records" }

type Favorite struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	DramaID   uint64    `json:"drama_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (Favorite) TableName() string { return "favorites" }

type DramaLike struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	DramaID   uint64    `json:"drama_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (DramaLike) TableName() string { return "drama_likes" }

type DramaFollow struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	DramaID   uint64    `json:"drama_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (DramaFollow) TableName() string { return "drama_follows" }

type WatchProgress struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UserID      uint64    `json:"user_id"`
	DramaID     uint64    `json:"drama_id"`
	EpisodeID   uint64    `json:"episode_id"`
	PositionSec int       `json:"position_sec"`
	DurationSec int       `json:"duration_sec"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (WatchProgress) TableName() string { return "watch_progress" }

type WatchHistory struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	DramaID   uint64    `json:"drama_id"`
	EpisodeID uint64    `json:"episode_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (WatchHistory) TableName() string { return "watch_history" }

type AdminUser struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64" json:"username"`
	PasswordHash string    `gorm:"size:191" json:"-"`
	Nickname     string    `gorm:"size:64" json:"nickname"`
	Role         string    `gorm:"size:16" json:"role"`
	TenantID     *uint64   `json:"tenant_id"`
	Status       int8      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

func (AdminUser) TableName() string { return "admin_users" }

type AppConfig struct {
	CfgKey   string    `gorm:"primaryKey;size:64" json:"cfg_key"`
	CfgValue string    `json:"cfg_value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AppConfig) TableName() string { return "app_configs" }
