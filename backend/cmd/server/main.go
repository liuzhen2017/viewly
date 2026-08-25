package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"viewly/internal/config"
	"viewly/internal/database"
	"viewly/internal/handler"
	"viewly/internal/router"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	tz, err := time.LoadLocation(cfg.App.Timezone)
	if err != nil {
		log.Fatalf("load timezone %s: %v", cfg.App.Timezone, err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	h := handler.New(db, cfg, tz)
	r := router.Setup(db, cfg, h)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("viewly backend listening on %s (tz=%s, mock_pay=%v)", addr, cfg.App.Timezone, cfg.Server.MockPay)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
