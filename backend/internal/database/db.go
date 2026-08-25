package database

import (
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	gormmysql "gorm.io/driver/mysql"

	"viewly/internal/config"
)

func Open(cfg *config.Config) (*gorm.DB, error) {
	mc := mysql.NewConfig()
	mc.User = cfg.MySQL.User
	mc.Passwd = cfg.MySQL.Password
	mc.Net = "tcp"
	mc.Addr = fmt.Sprintf("%s:%d", cfg.MySQL.Host, cfg.MySQL.Port)
	mc.DBName = cfg.MySQL.Database
	mc.ParseTime = true
	mc.Loc = time.UTC
	mc.Params = map[string]string{"charset": "utf8mb4", "time_zone": "'+00:00'"}

	db, err := gorm.Open(gormmysql.New(gormmysql.Config{DSNConfig: mc, SkipInitializeWithVersion: false}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}
