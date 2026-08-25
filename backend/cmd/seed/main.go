package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"viewly/internal/config"
	"viewly/internal/database"
	"viewly/internal/model"
)

// Sample videos: Blender Foundation open movies + Google demo clips (CC/free use).
var sampleVideos = []string{
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ElephantsDream.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerFun.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerJoyrides.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerMeltdowns.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/Sintel.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/TearsOfSteel.mp4",
	"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/VolkswagenGTIReview.mp4",
}

type seedDrama struct {
	title, desc, tags string
	featured, hot, completed int8
}

var seedDramas = []seedDrama{
	{"The Mafia Boss's Fake Wife", "To save her brother, she signs a contract marriage with the city's most feared mafia boss. Fake turns real when bullets fly.", "CEO,Contract,Marriage", 1, 1, 0},
	{"Velocity Guardian", "A delivery rider by day, street-racing guardian by night. One last race decides the fate of the city.", "Action,Street", 1, 1, 0},
	{"Lethal Vision: Blind Girl's Survival", "Blind since the accident, she 'sees' more than anyone. Now the killer is hunting her darkest secret.", "Thriller,Suspense", 0, 1, 0},
	{"The Unrivaled Son-in-Law", "Mocked for three years, the hidden heir finally reveals his true power to protect his wife.", "CEO,Regret,Uplift", 1, 0, 0},
	{"Harbor Glow", "Two strangers, one lighthouse, and a summer that changes everything.", "Romance,Healing", 1, 0, 1},
	{"Tug of War in Life's Smoke", "A food-stall owner fights city hall to keep her father's legacy alive.", "Life,Family", 0, 0, 0},
	{"Leaving Blossoms for My Lord", "A modern nurse wakes up in ancient times as the emperor's discarded consort.", "Ancient,Costume", 0, 1, 0},
	{"Cretaceous Echo", "A ranger discovers dinosaurs never went extinct — in her own backyard.", "SciFi,Adventure", 0, 0, 0},
	{"Hundred Surrenders", "The general who never lost a war surrenders a hundred times — for the girl he loves.", "Ancient,Romance", 0, 0, 1},
	{"Midnight CEO", "By day she's an intern. By night she signs billion-dollar deals. Nobody knows her double life.", "CEO,Identity", 0, 1, 0},
}

var seedCategories = []string{"CEO", "Regret", "Funny", "Revenge", "Romance", "Ancient", "Travel"}

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config.yaml")
	force := flag.Bool("force", false, "re-seed content tables even if data exists")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	// ---- tenants ----
	mainTenant := model.Tenant{Name: "Viewly Main", Slug: "main", Status: 1}
	if err := db.Where("slug = ?", "main").FirstOrCreate(&mainTenant).Error; err != nil {
		log.Fatalf("ensure main tenant: %v", err)
	}
	demoTenant := model.Tenant{Name: "Demo Site", Slug: "demo", Status: 1}
	if err := db.Where("slug = ?", "demo").FirstOrCreate(&demoTenant).Error; err != nil {
		log.Fatalf("ensure demo tenant: %v", err)
	}

	// ---- platform super admin (NULL tenant) ----
	var superAdmin model.AdminUser
	if err := db.Where("username = ? AND tenant_id IS NULL", "admin").First(&superAdmin).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Create(&model.AdminUser{Username: "admin", PasswordHash: string(hash), Nickname: "Platform Admin", Role: "super", Status: 1})
		log.Println("super admin created: admin / admin123")
	} else if superAdmin.Role != "super" {
		db.Model(&superAdmin).Update("role", "super")
	}

	// ---- demo tenant admin ----
	var demoAdminCount int64
	db.Model(&model.AdminUser{}).Where("tenant_id = ?", demoTenant.ID).Count(&demoAdminCount)
	if demoAdminCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
		tid := demoTenant.ID
		db.Create(&model.AdminUser{Username: "demo", PasswordHash: string(hash), Nickname: "Demo Admin", Role: "admin", TenantID: &tid, Status: 1})
		log.Println("demo tenant admin created: demo / demo123")
	}

	var dramaCount int64
	db.Model(&model.Drama{}).Where("tenant_id = ?", mainTenant.ID).Count(&dramaCount)
	if dramaCount > 0 && !*force {
		seedDemo(db, demoTenant)
		log.Println("content already seeded, use -force to re-seed main content")
		return
	}
	if *force {
		db.Where("tenant_id = ?", mainTenant.ID).Delete(&model.Drama{})
		db.Where("tenant_id = ?", mainTenant.ID).Delete(&model.Episode{})
		db.Where("tenant_id = ?", mainTenant.ID).Delete(&model.Category{})
		db.Where("tenant_id = ?", mainTenant.ID).Delete(&model.Banner{})
		db.Where("tenant_id = ?", mainTenant.ID).Delete(&model.CoinPackage{})
		db.Where("tenant_id = ?", mainTenant.ID).Delete(&model.VipPlan{})
	}

	// ---- main tenant categories ----
	cats := map[string]uint64{}
	for i, name := range seedCategories {
		cat := model.Category{TenantID: mainTenant.ID, Name: name, Sort: i * 10, Status: 1}
		if err := db.Create(&cat).Error; err == nil {
			cats[name] = cat.ID
		}
	}
	catNames := []string{"CEO", "Romance", "Revenge", "Ancient"}

	// ---- main tenant dramas + episodes ----
	for i, sd := range seedDramas {
		catID := cats[catNames[i%len(catNames)]]
		now := time.Now().UTC().AddDate(0, 0, -i)
		d := model.Drama{
			TenantID: mainTenant.ID,
			Title: sd.title, Description: sd.desc, CategoryID: &catID,
			Cover: fmt.Sprintf("/static/posters/p%d.svg", i+1),
			Banner: fmt.Sprintf("/static/posters/p%d.svg", i+1),
			Tags: sd.tags, IsFeatured: sd.featured, IsCompleted: sd.completed,
			IsHot: sd.hot, Status: 1, Views: int64(12000 - i*900), Likes: int64(3200 - i*210),
			Sort: i * 10, CreatedAt: now, UpdatedAt: now,
		}
		if err := db.Create(&d).Error; err != nil {
			log.Printf("drama %q: %v", sd.title, err)
			continue
		}
		epCount := 10 + (i % 3) * 2
		for ep := 1; ep <= epCount; ep++ {
			price := 0
			if ep > 3 {
				price = 40 // first 3 episodes free, then 40 coins each
			}
			e := model.Episode{
				TenantID: mainTenant.ID,
				DramaID: d.ID, EpIndex: ep,
				Title: fmt.Sprintf("%s Ep %d", sd.title, ep),
				VideoURL: sampleVideos[(i+ep)%len(sampleVideos)],
				DurationSec: 120 + (ep%5)*30,
				PriceCoins: price, Status: 1,
			}
			if err := db.Create(&e).Error; err != nil {
				log.Printf("episode %d: %v", ep, err)
			}
		}
	}

	// ---- banners (top 3 main-tenant dramas) ----
	var tops []model.Drama
	db.Where("tenant_id = ?", mainTenant.ID).Order("is_featured DESC, views DESC").Limit(3).Find(&tops)
	for i, d := range tops {
		db.Create(&model.Banner{TenantID: mainTenant.ID, Image: d.Cover, DramaID: d.ID, Sort: i * 10, Status: 1})
	}

	// ---- coin packages ----
	packs := []model.CoinPackage{
		{TenantID: mainTenant.ID, Coins: 1000, BonusCoins: 10, PriceCents: 199, Currency: "USD", Label: "1,000 Coins", Tag: "", Sort: 10, Status: 1},
		{TenantID: mainTenant.ID, Coins: 3000, BonusCoins: 300, PriceCents: 499, Currency: "USD", Label: "3,300 Coins", Tag: "Popular", Sort: 20, Status: 1},
		{TenantID: mainTenant.ID, Coins: 8000, BonusCoins: 1200, PriceCents: 999, Currency: "USD", Label: "9,200 Coins", Tag: "Best Value", Sort: 30, Status: 1},
		{TenantID: mainTenant.ID, Coins: 20000, BonusCoins: 5000, PriceCents: 1999, Currency: "USD", Label: "25,000 Coins", Tag: "", Sort: 40, Status: 1},
	}
	for i := range packs {
		db.Create(&packs[i])
	}

	// ---- vip plans ----
	plans := []model.VipPlan{
		{TenantID: mainTenant.ID, Days: 7, PriceCents: 299, Currency: "USD", Label: "Weekly VIP", Tag: "", Sort: 10, Status: 1},
		{TenantID: mainTenant.ID, Days: 30, PriceCents: 999, Currency: "USD", Label: "Monthly VIP", Tag: "Most Popular", Sort: 20, Status: 1},
		{TenantID: mainTenant.ID, Days: 90, PriceCents: 2499, Currency: "USD", Label: "Quarterly VIP", Tag: "Save 17%", Sort: 30, Status: 1},
	}
	for i := range plans {
		db.Create(&plans[i])
	}

	seedDemo(db, demoTenant)
	log.Println("seed complete: main + demo tenants")
}

// seedDemo provisions a small independent catalog for the demo tenant so
// cross-tenant isolation can be verified end to end.
func seedDemo(db *gorm.DB, t model.Tenant) {
	var count int64
	db.Model(&model.Drama{}).Where("tenant_id = ?", t.ID).Count(&count)
	if count > 0 {
		return
	}
	catCEO := model.Category{TenantID: t.ID, Name: "CEO", Sort: 10, Status: 1}
	catRomance := model.Category{TenantID: t.ID, Name: "Romance", Sort: 20, Status: 1}
	db.Create(&catCEO)
	db.Create(&catRomance)

	demos := []struct{ title, desc string; cat *model.Category }{
		{"Demo: CEO's Contract Bride", "A demo-tenant exclusive: contract marriage with a twist.", &catCEO},
		{"Demo: Sunset Vows", "A demo-tenant romance set on a tropical island.", &catRomance},
	}
	for i, dd := range demos {
		d := model.Drama{
			TenantID: t.ID, Title: dd.title, Description: dd.desc, CategoryID: &dd.cat.ID,
			Cover: fmt.Sprintf("/static/posters/p%d.svg", i+3), Banner: fmt.Sprintf("/static/posters/p%d.svg", i+3),
			Tags: "Demo", IsFeatured: 1, IsHot: 1, Status: 1, Views: int64(500 - i*100), Likes: 80,
			Sort: i * 10,
		}
		if err := db.Create(&d).Error; err != nil {
			log.Printf("demo drama %q: %v", dd.title, err)
			continue
		}
		for ep := 1; ep <= 6; ep++ {
			price := 0
			if ep > 2 {
				price = 30
			}
			db.Create(&model.Episode{
				TenantID: t.ID, DramaID: d.ID, EpIndex: ep,
				Title: fmt.Sprintf("%s Ep %d", dd.title, ep),
				VideoURL: sampleVideos[(i+ep)%len(sampleVideos)],
				DurationSec: 100, PriceCoins: price, Status: 1,
			})
		}
		db.Create(&model.Banner{TenantID: t.ID, Image: d.Cover, DramaID: d.ID, Sort: i * 10, Status: 1})
	}
	db.Create(&model.CoinPackage{TenantID: t.ID, Coins: 500, BonusCoins: 20, PriceCents: 99, Currency: "USD", Label: "520 Coins", Tag: "Popular", Sort: 10, Status: 1})
	db.Create(&model.VipPlan{TenantID: t.ID, Days: 30, PriceCents: 499, Currency: "USD", Label: "Demo Monthly VIP", Tag: "", Sort: 10, Status: 1})
	log.Println("demo tenant content created (2 dramas, own packages)")
}
