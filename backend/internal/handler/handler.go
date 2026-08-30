package handler

import (
	"time"

	"gorm.io/gorm"

	"viewly/internal/config"
)

// Handler carries shared dependencies for all route handlers.
type Handler struct {
	DB     *gorm.DB
	Cfg    *config.Config
	TZ     *time.Location
}

func New(db *gorm.DB, cfg *config.Config, tz *time.Location) *Handler {
	remuxCDNBase = cfg.AWS.CDNBase
	remuxRegion = cfg.AWS.Region
	remuxBucket = cfg.AWS.Bucket
	return &Handler{DB: db, Cfg: cfg, TZ: tz}
}
