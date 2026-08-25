package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/model"
	"viewly/internal/service"
)

type taskDef struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Coins     int    `json:"coins"`
	Threshold int    `json:"threshold"` // progress needed; 0 = claim directly
	OneTime   bool   `json:"one_time"`
	Group     string `json:"group"` // daily | social
}

var taskDefs = []taskDef{
	{Key: "watch_5", Title: "Watch 5 Videos", Coins: 100, Threshold: 5, Group: "daily"},
	{Key: "watch_10", Title: "Watch 10 Videos", Coins: 200, Threshold: 10, Group: "daily"},
	{Key: "watch_20", Title: "Watch 20 Videos", Coins: 300, Threshold: 20, Group: "daily"},
	{Key: "share", Title: "Share Drama", Coins: 30, Threshold: 1, Group: "daily"},
	{Key: "like", Title: "Like Drama", Coins: 20, Threshold: 1, Group: "daily"},
	{Key: "favorite", Title: "Favorite Drama", Coins: 40, Threshold: 1, Group: "daily"},
	{Key: "follow_ins", Title: "Follow Instagram", Coins: 100, Group: "social", OneTime: true},
	{Key: "follow_tiktok", Title: "Follow TikTok", Coins: 100, Group: "social", OneTime: true},
	{Key: "follow_facebook", Title: "Follow Facebook", Coins: 100, Group: "social", OneTime: true},
	{Key: "follow_youtube", Title: "Follow YouTube", Coins: 100, Group: "social", OneTime: true},
}

func defByKey(key string) *taskDef {
	for i := range taskDefs {
		if taskDefs[i].Key == key {
			return &taskDefs[i]
		}
	}
	return nil
}

// bumpTask increments today's progress for a task key (server-side events).
func (h *Handler) bumpTask(userID uint64, key string, delta int) {
	def := defByKey(key)
	if def == nil || def.OneTime {
		return
	}
	day := h.appDay(time.Now().UTC())
	h.DB.Exec(`
		INSERT INTO task_records (user_id, task_key, day_date, progress, rewarded, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, NOW(), NOW())
		ON DUPLICATE KEY UPDATE progress = progress + VALUES(progress), updated_at = NOW()`,
		userID, key, day, delta)
}

// GET /api/rewards — check-in state plus all task states for today.
func (h *Handler) Rewards(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	now := time.Now().UTC()
	today := h.appDay(now)

	// check-in state from history: consecutive days within a rolling 7-day cycle
	var recs []model.CheckinRecord
	h.DB.Where("user_id = ?", u.ID).Order("day_date DESC").Limit(10).Find(&recs)
	rewards := h.Cfg.App.CheckinRewards
	cycle := len(rewards)
	cycleDay := 1
	todayDone := false
	if len(recs) > 0 {
		yesterdayStr := h.appDay(now.AddDate(0, 0, -1))
		if recs[0].DayDate == today {
			todayDone = true
			cycleDay = recs[0].CycleDay
		} else if recs[0].DayDate == yesterdayStr {
			cycleDay = recs[0].CycleDay%cycle + 1
		}
	}
	checkinDays := make([]gin.H, 0, cycle)
	for i := 1; i <= cycle; i++ {
		checkinDays = append(checkinDays, gin.H{
			"day": i, "coins": rewards[i-1],
			"done": len(recs) > 0 && recs[0].CycleDay >= i && (recs[0].DayDate == today || recs[0].CycleDay >= cycleDay),
			"current": i == cycleDay,
		})
	}

	// task states for today (+ lifetime state for one-time tasks)
	var daily []model.TaskRecord
	h.DB.Where("user_id = ? AND day_date = ?", u.ID, today).Find(&daily)
	dailyMap := map[string]model.TaskRecord{}
	for _, r := range daily {
		dailyMap[r.TaskKey] = r
	}
	var onetime []model.TaskRecord
	h.DB.Where("user_id = ? AND rewarded = 1", u.ID).Find(&onetime)
	doneMap := map[string]bool{}
	for _, r := range onetime {
		doneMap[r.TaskKey] = true
	}

	tasks := make([]gin.H, 0, len(taskDefs))
	for _, def := range taskDefs {
		var progress, rewarded int
		if def.OneTime {
			if doneMap[def.Key] {
				rewarded = 1
			}
		} else if r, ex := dailyMap[def.Key]; ex {
			progress = r.Progress
			rewarded = int(r.Rewarded)
		}
		tasks = append(tasks, gin.H{
			"key": def.Key, "title": def.Title, "coins": def.Coins,
			"threshold": def.Threshold, "one_time": def.OneTime, "group": def.Group,
			"progress": progress, "rewarded": rewarded,
			"claimable": rewarded == 0 && (def.Threshold == 0 || progress >= def.Threshold),
		})
	}

	ok(c, gin.H{
		"checkin": gin.H{"today_done": todayDone, "cycle_day": cycleDay, "days": checkinDays},
		"tasks":   tasks,
		"coins":   u.Coins,
	})
}

// POST /api/rewards/checkin
func (h *Handler) Checkin(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	now := time.Now().UTC()
	today := h.appDay(now)
	rewards := h.Cfg.App.CheckinRewards
	cycle := len(rewards)

	var last model.CheckinRecord
	err := h.DB.Where("user_id = ?", u.ID).Order("day_date DESC").First(&last).Error
	day := 1
	if err == nil {
		if last.DayDate == today {
			failBiz(c, 3001, "already checked in today")
			return
		}
		yesterdayStr := h.appDay(now.AddDate(0, 0, -1))
		if last.DayDate == yesterdayStr {
			day = last.CycleDay%cycle + 1
		}
	}

	coins := int64(rewards[day-1])
	var newBal int64
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		rec := model.CheckinRecord{UserID: u.ID, DayDate: today, CycleDay: day, Coins: int(coins)}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		fresh, err := service.Credit(tx, u.ID, coins, "checkin", today, fmt.Sprintf("day %d check-in", day))
		if err != nil {
			return err
		}
		newBal = fresh.Coins
		return nil
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "check-in failed")
		return
	}
	ok(c, gin.H{"day": day, "coins": coins, "balance": newBal})
}

// POST /api/rewards/tasks/:key/claim
func (h *Handler) ClaimTask(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	key := c.Param("key")
	def := defByKey(key)
	if def == nil {
		fail(c, http.StatusNotFound, "unknown task")
		return
	}
	now := time.Now().UTC()
	today := h.appDay(now)

	var newBal int64
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if def.OneTime {
			var cnt int64
			tx.Model(&model.TaskRecord{}).Where("user_id = ? AND task_key = ? AND rewarded = 1", u.ID, key).Count(&cnt)
			if cnt > 0 {
				return errTaskClaimed
			}
			rec := model.TaskRecord{UserID: u.ID, TaskKey: key, DayDate: today, Progress: 1, Rewarded: 1}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		} else {
			var rec model.TaskRecord
			if err := tx.Where("user_id = ? AND task_key = ? AND day_date = ?", u.ID, key, today).First(&rec).Error; err != nil {
				return errTaskNotReady
			}
			if rec.Rewarded == 1 {
				return errTaskClaimed
			}
			if rec.Progress < def.Threshold {
				return errTaskNotReady
			}
			if err := tx.Model(&rec).Update("rewarded", 1).Error; err != nil {
				return err
			}
		}
		fresh, err := service.Credit(tx, u.ID, int64(def.Coins), "task", key, def.Title)
		if err != nil {
			return err
		}
		newBal = fresh.Coins
		return nil
	})
	if err != nil {
		switch err {
		case errTaskClaimed:
			failBiz(c, 3002, "reward already claimed")
		case errTaskNotReady:
			failBiz(c, 3003, "task not completed yet")
		default:
			fail(c, http.StatusInternalServerError, "claim failed")
		}
		return
	}
	ok(c, gin.H{"coins": def.Coins, "balance": newBal})
}

// POST /api/rewards/tasks/share/progress — client-reported share event.
// (Anti-abuse: server accepts at most 1 progress per day for share.)
func (h *Handler) ShareProgress(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	h.bumpTask(u.ID, "share", 1)
	ok(c, gin.H{"progressed": true})
}

var (
	errTaskClaimed   = fmt.Errorf("task claimed")
	errTaskNotReady  = fmt.Errorf("task not ready")
)
