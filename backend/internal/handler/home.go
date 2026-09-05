package handler

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/model"
)

type dramaCard struct {
	ID          uint64    `json:"id"`
	Title       string    `json:"title"`
	Cover       string    `json:"cover"`
	Tags        string    `json:"tags"`
	IsCompleted int8      `json:"is_completed"`
	IsHot       int8      `json:"is_hot"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	CreatedAt   time.Time `json:"created_at"`
	Episodes    int64     `json:"episodes"`
	CategoryID  *uint64   `json:"category_id,omitempty"`
}

// cardQuery is the drama-card projection, always tenant-scoped.
func (h *Handler) cardQuery(c *gin.Context) *gorm.DB {
	return h.DB.Table("dramas d").
		Where("d.tenant_id = ?", h.tID(c)).
		Select(`d.id, d.title, d.cover, d.tags, d.is_completed, d.is_hot, d.views, d.likes, d.created_at,
			d.category_id,
			(SELECT COUNT(*) FROM episodes e WHERE e.drama_id = d.id AND e.status = 1) AS episodes`)
}

func (h *Handler) cards(q *gorm.DB) ([]dramaCard, error) {
	out := make([]dramaCard, 0) // never nil: JSON must be [], not null
	if err := q.Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// dbFailed centralizes the "database unavailable" response so a broken DB
// connection surfaces as 5xx instead of silent empty lists.
func dbFailed(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	fail(c, http.StatusInternalServerError, "database unavailable")
	return true
}

// GET /api/home — everything the home screen needs in one shot.
// Queries run concurrently: the remote DB has high per-query latency, so
// wall time is the slowest query rather than the sum of all of them.
func (h *Handler) Home(c *gin.Context) {
	var (
		bannerRows []struct {
			model.Banner
			DramaTitle *string `json:"drama_title"`
		}
		featured  []dramaCard
		newRel    []dramaCard
		hot       []dramaCard
		cats      []model.Category
		channelDs []dramaCard
		wg        sync.WaitGroup
		errs      [6]error
	)
	tid := h.tID(c)
	wg.Add(6)
	go func() {
		defer wg.Done()
		errs[0] = h.DB.Table("banners b").
			Select("b.*, d.title AS drama_title").
			Joins("LEFT JOIN dramas d ON d.id = b.drama_id AND d.status = 1").
			Where("b.status = 1 AND b.tenant_id = ?", tid).Order("b.sort ASC, b.id ASC").Limit(6).Scan(&bannerRows).Error
	}()
	go func() { defer wg.Done(); featured, errs[1] = h.cards(h.cardQuery(c).Where("d.status = 1 AND d.is_featured = 1").Order("d.sort ASC, d.id DESC").Limit(9)) }()
	go func() { defer wg.Done(); newRel, errs[2] = h.cards(h.cardQuery(c).Where("d.status = 1").Order("d.created_at DESC, d.id DESC").Limit(9)) }()
	go func() { defer wg.Done(); hot, errs[3] = h.cards(h.cardQuery(c).Where("d.status = 1").Order("d.views DESC").Limit(9)) }()
	go func() { defer wg.Done(); errs[4] = h.DB.Where("status = 1 AND tenant_id = ?", tid).Order("sort ASC, id ASC").Find(&cats).Error }()
	go func() { defer wg.Done(); channelDs, errs[5] = h.cards(h.cardQuery(c).Where("d.status = 1").Order("d.is_hot DESC, d.views DESC").Limit(120)) }()
	wg.Wait()
	for _, e := range errs {
		if dbFailed(c, e) {
			return
		}
	}

	// banner drama titles resolved by the JOIN above
	bannerOut := make([]gin.H, 0, len(bannerRows))
	for _, b := range bannerRows {
		item := gin.H{"image": b.Image, "drama_id": b.DramaID, "link": b.Link}
		if b.DramaID > 0 {
			if b.DramaTitle != nil {
				item["title"] = *b.DramaTitle
			} else {
				item["drama_id"] = 0
			}
		}
		bannerOut = append(bannerOut, item)
	}

	// group channel dramas by category (top 9 by the query order)
	catSet := map[uint64]bool{}
	for _, cat := range cats {
		catSet[cat.ID] = true
	}
	perCat := map[uint64][]dramaCard{}
	for _, d := range channelDs {
		if d.CategoryID != nil && catSet[*d.CategoryID] && len(perCat[*d.CategoryID]) < 9 {
			perCat[*d.CategoryID] = append(perCat[*d.CategoryID], d)
		}
	}
	channels := make([]gin.H, 0, len(cats))
	for _, cat := range cats {
		list := perCat[cat.ID]
		if list == nil {
			list = make([]dramaCard, 0)
		}
		channels = append(channels, gin.H{"id": cat.ID, "name": cat.Name, "dramas": list})
	}

	ok(c, gin.H{
		"banners": bannerOut, "featured": featured,
		"new_releases": newRel, "channels": channels, "hot": hot,
	})
}

// GET /api/categories
func (h *Handler) Categories(c *gin.Context) {
	var cats []model.Category
	if dbFailed(c, h.DB.Where("status = 1 AND tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&cats).Error) {
		return
	}
	ok(c, cats)
}

// GET /api/dramas — filterable, pageable list.
// filters: category_id, completed=1, hot=1, keyword, sort=views|newest
func (h *Handler) DramaList(c *gin.Context) {
	page, size := pageParams(c, 12)
	listFilters := func(q *gorm.DB) *gorm.DB {
		q = q.Where("d.status = 1 AND d.tenant_id = ?", h.tID(c))
		if v := c.Query("category_id"); v != "" {
			q = q.Where("d.category_id = ?", toInt(v, 0))
		}
		if c.Query("completed") == "1" {
			q = q.Where("d.is_completed = 1")
		}
		if c.Query("hot") == "1" {
			q = q.Where("d.is_hot = 1")
		}
		if kw := c.Query("keyword"); kw != "" {
			q = q.Where("d.title LIKE ?", "%"+kw+"%")
		}
		return q
	}
	q := h.cardQuery(c).Scopes(listFilters)
	switch c.Query("sort") {
	case "views":
		q = q.Order("d.views DESC")
	default:
		q = q.Order("d.created_at DESC, d.id DESC")
	}

	var total int64
	if dbFailed(c, h.DB.Table("dramas d").Scopes(listFilters).Count(&total).Error) {
		return
	}
	list, err := h.cards(q.Offset(offset(page, size)).Limit(size))
	if dbFailed(c, err) {
		return
	}
	ok(c, gin.H{"list": list, "page": page, "size": size, "total": total})
}

// GET /api/dramas/:id — detail with per-episode access flags for the caller.
func (h *Handler) DramaDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == 0 {
		fail(c, http.StatusBadRequest, "bad id")
		return
	}
	var d model.Drama
	if err := h.DB.Where("id = ? AND status = 1 AND tenant_id = ?", id, h.tID(c)).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "drama not found")
		} else {
			fail(c, http.StatusInternalServerError, "database unavailable")
		}
		return
	}

	var eps []model.Episode
	if dbFailed(c, h.DB.Where("drama_id = ? AND status = 1", id).Order("ep_index ASC").Find(&eps).Error) {
		return
	}

	u, _ := c.Get("user")
	isVip := false
	unlocked := map[uint64]bool{}
	fav, liked, followed := false, false, false
	var progress *model.WatchProgress
	if u != nil {
		user := u.(*model.User)
		isVip = user.IsVIP(time.Now().UTC())
		var unlocks []model.EpisodeUnlock
		h.DB.Where("user_id = ? AND drama_id = ?", user.ID, id).Find(&unlocks)
		for _, x := range unlocks {
			unlocked[x.EpisodeID] = true
		}
		var cnt int64
		h.DB.Model(&model.Favorite{}).Where("user_id = ? AND drama_id = ?", user.ID, id).Count(&cnt)
		fav = cnt > 0
		h.DB.Model(&model.DramaLike{}).Where("user_id = ? AND drama_id = ?", user.ID, id).Count(&cnt)
		liked = cnt > 0
		h.DB.Model(&model.DramaFollow{}).Where("user_id = ? AND drama_id = ?", user.ID, id).Count(&cnt)
		followed = cnt > 0
		var wp model.WatchProgress
		if err := h.DB.Where("user_id = ? AND drama_id = ?", user.ID, id).First(&wp).Error; err == nil {
			progress = &wp
		}
	}

	epOut := make([]gin.H, 0, len(eps))
	for _, e := range eps {
		epOut = append(epOut, gin.H{
			"id": e.ID, "ep_index": e.EpIndex, "title": e.Title,
			"duration_sec": e.DurationSec, "price_coins": e.PriceCoins,
			"free": e.PriceCoins == 0, "accessible": e.PriceCoins == 0 || isVip || unlocked[e.ID],
		})
	}

	var catName string
	if d.CategoryID != nil {
		var cat model.Category
		if err := h.DB.First(&cat, *d.CategoryID).Error; err == nil {
			catName = cat.Name
		}
	}

	ok(c, gin.H{
		"id": d.ID, "title": d.Title, "description": d.Description,
		"cover": d.Cover, "banner": d.Banner, "tags": d.Tags,
		"is_completed": d.IsCompleted, "views": d.Views, "likes": d.Likes,
		"category_id": d.CategoryID, "category_name": catName,
		"created_at": d.CreatedAt, "episodes": epOut,
		"is_favorite": fav, "is_liked": liked, "is_followed": followed,
		"progress": progress,
	})
}

// GET /api/search?keyword=
func (h *Handler) Search(c *gin.Context) {
	kw := c.Query("keyword")
	if kw == "" {
		ok(c, []any{})
		return
	}
	list, err := h.cards(h.cardQuery(c).Where("d.status = 1 AND d.title LIKE ?", "%"+kw+"%").Order("d.views DESC").Limit(30))
	if dbFailed(c, err) {
		return
	}
	ok(c, list)
}

// GET /api/feed — episodes for the vertical swipe Watch tab.
func (h *Handler) Feed(c *gin.Context) {
	page, _ := pageParams(c, 10)
	type feedItem struct {
		EpisodeID   uint64 `json:"episode_id"`
		EpIndex     int    `json:"ep_index"`
		EpTitle     string `json:"ep_title"`
		PriceCoins  int    `json:"price_coins"`
		VideoURL    string `json:"video_url"`
		DramaID     uint64 `json:"drama_id"`
		DramaTitle  string `json:"drama_title"`
		Cover       string `json:"cover"`
		Tags        string `json:"tags"`
		Category    string `json:"category"`
		Views       int64  `json:"views"`
	}
	var items []feedItem
	if dbFailed(c, h.DB.Raw(`
		SELECT e.id AS episode_id, e.ep_index, CONCAT(d.title, ' - Ep ', e.ep_index) AS ep_title,
		       e.price_coins, e.video_url, d.id AS drama_id, d.title AS drama_title, d.cover, d.tags,
		       COALESCE(c.name, '') AS category, d.views
		FROM episodes e
		JOIN dramas d ON d.id = e.drama_id AND d.status = 1
		LEFT JOIN categories c ON c.id = d.category_id
		WHERE e.status = 1 AND e.ep_index = 1 AND e.tenant_id = ?
		ORDER BY d.is_hot DESC, d.views DESC, d.id DESC
		LIMIT ? OFFSET ?`, h.tID(c), 10, offset(page, 10)).Scan(&items).Error) {
		return
	}

	u, _ := c.Get("user")
	isVip := false
	if u != nil {
		isVip = u.(*model.User).IsVIP(time.Now().UTC())
	}
	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		acc := it.PriceCoins == 0 || isVip
		if !acc && u != nil {
			var cnt int64
			h.DB.Model(&model.EpisodeUnlock{}).Where("user_id = ? AND episode_id = ?", u.(*model.User).ID, it.EpisodeID).Count(&cnt)
			acc = cnt > 0
		}
		// video URL only for episodes the viewer can actually watch:
		// the feed card previews it (muted autoplay); paid/unwatched ones
		// fall back to the cover + a tap into the unlock flow.
		vu := ""
		if acc {
			vu = it.VideoURL
		}
		out = append(out, gin.H{
			"episode_id": it.EpisodeID, "ep_index": it.EpIndex, "title": it.EpTitle,
			"drama_id": it.DramaID, "drama_title": it.DramaTitle, "cover": it.Cover,
			"tags": it.Tags, "category": it.Category, "views": it.Views,
			"price_coins": it.PriceCoins, "accessible": acc, "video_url": vu,
		})
	}
	ok(c, out)
}
