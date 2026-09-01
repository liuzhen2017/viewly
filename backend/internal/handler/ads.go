package handler

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/model"
	"viewly/internal/service"
)

// ---------- tenant ad config (public + admin) ----------

// GET /api/ad-config — public ad settings for the resolved tenant; the H5
// client uses this to inject the AdSense script and render the watch-ad card.
func (h *Handler) AdConfig(c *gin.Context) {
	t := h.tenant(c)
	if t == nil {
		fail(c, http.StatusNotFound, "unknown tenant")
		return
	}
	ok(c, gin.H{
		"adsense_client":       t.AdsenseClient,
		"adsense_enabled":      t.AdsenseEnabled == 1 && t.AdsenseClient != "",
		"rewarded_ad_mode":     t.RewardedAdMode,
		"rewarded_ad_coins":    t.RewardedAdCoins,
		"rewarded_ad_daily_limit": t.RewardedAdDailyLimit,
		"admob_app_id":         t.AdmobAppID,
		"admob_rewarded_unit_id": t.AdmobRewardedUnitID,
	})
}

type adSettingsReq struct {
	AdsenseClient         *string `json:"adsense_client"`
	AdsenseEnabled        *int8   `json:"adsense_enabled"`
	RewardedAdMode        *string `json:"rewarded_ad_mode"`
	RewardedAdCoins       *int    `json:"rewarded_ad_coins"`
	RewardedAdDailyLimit  *int    `json:"rewarded_ad_daily_limit"`
	AdmobAppID            *string `json:"admob_app_id"`
	AdmobRewardedUnitID   *string `json:"admob_rewarded_unit_id"`
}

// GET/PUT /api/admin/ad-settings — the caller's tenant ad monetization config.
func (h *Handler) AdSettingsGet(c *gin.Context) {
	ok(c, h.tenant(c))
}

func (h *Handler) AdSettingsUpdate(c *gin.Context) {
	var req adSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.RewardedAdMode != nil && *req.RewardedAdMode != "off" && *req.RewardedAdMode != "client" && *req.RewardedAdMode != "ssv" {
		fail(c, http.StatusBadRequest, "rewarded_ad_mode must be off|client|ssv")
		return
	}
	if req.RewardedAdCoins != nil && (*req.RewardedAdCoins < 0 || *req.RewardedAdCoins > 10000) {
		fail(c, http.StatusBadRequest, "rewarded_ad_coins out of range")
		return
	}
	if req.RewardedAdDailyLimit != nil && (*req.RewardedAdDailyLimit < 0 || *req.RewardedAdDailyLimit > 100) {
		fail(c, http.StatusBadRequest, "rewarded_ad_daily_limit out of range")
		return
	}
	updates := map[string]any{}
	if req.AdsenseClient != nil {
		updates["adsense_client"] = strings.TrimSpace(*req.AdsenseClient)
	}
	if req.AdsenseEnabled != nil {
		updates["adsense_enabled"] = *req.AdsenseEnabled
	}
	if req.RewardedAdMode != nil {
		updates["rewarded_ad_mode"] = *req.RewardedAdMode
	}
	if req.RewardedAdCoins != nil {
		updates["rewarded_ad_coins"] = *req.RewardedAdCoins
	}
	if req.RewardedAdDailyLimit != nil {
		updates["rewarded_ad_daily_limit"] = *req.RewardedAdDailyLimit
	}
	if req.AdmobAppID != nil {
		updates["admob_app_id"] = strings.TrimSpace(*req.AdmobAppID)
	}
	if req.AdmobRewardedUnitID != nil {
		updates["admob_rewarded_unit_id"] = strings.TrimSpace(*req.AdmobRewardedUnitID)
	}
	if len(updates) == 0 {
		ok(c, h.tenant(c))
		return
	}
	if err := h.DB.Model(&model.Tenant{}).Where("id = ?", h.tID(c)).Updates(updates).Error; err != nil {
		fail(c, http.StatusInternalServerError, "update failed")
		return
	}
	// refresh the cached row so the next request sees new ad settings
	var fresh model.Tenant
	if err := h.DB.First(&fresh, h.tID(c)).Error; err == nil {
		c.Set("tenant", &fresh)
	}
	ok(c, &fresh)
}

// GET /ads.txt — Google's crawler never sends X-Tenant-Slug, so resolve the
// tenant from the Host header (first label), same convention as the frontend.
func (h *Handler) AdsTxt(c *gin.Context) {
	slug := strings.TrimSpace(c.GetHeader("X-Tenant-Slug"))
	if slug == "" {
		host := c.Request.Host
		if i := strings.LastIndex(host, ":"); i > 0 {
			host = host[:i]
		}
		host = strings.TrimSuffix(host, ".")
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			slug = parts[0]
		}
		// mirror the frontend: apex and reserved labels serve the main tenant
		if len(parts) <= 2 || slug == "www" || slug == "api" || slug == "admin" || slug == "cdn" {
			slug = "main"
		}
	}
	var t model.Tenant
	if err := h.DB.Where("slug = ? AND status = 1", slug).First(&t).Error; err != nil {
		c.String(http.StatusNotFound, "")
		return
	}
	if t.AdsenseEnabled != 1 || t.AdsenseClient == "" {
		c.String(http.StatusNotFound, "")
		return
	}
	c.String(http.StatusOK, "google.com, %s, DIRECT, f08c47fec0942fa0\n", t.AdsenseClient)
}

// ---------- rewarded video: client-reported completion (H5) ----------

// POST /api/rewards/watch-ad/complete — H5 flow. The ad SDK runs client-side;
// the server enforces the per-tenant daily cap so abuse is bounded by
// limit × coins per day. Production apps should use the SSV flow below.
func (h *Handler) WatchAdComplete(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	t := h.tenant(c)
	if t == nil || (t.RewardedAdMode != "client" && t.RewardedAdMode != "ssv") {
		failBiz(c, 5001, "rewarded ads not enabled on this site")
		return
	}
	if t.RewardedAdMode == "ssv" {
		failBiz(c, 5002, "this site only credits ads via server verification")
		return
	}
	day := h.appDay(time.Now().UTC())
	limit := t.RewardedAdDailyLimit
	if limit <= 0 {
		failBiz(c, 5001, "rewarded ads not enabled on this site")
		return
	}

	var balance int64
	watched := 0
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var rec model.TaskRecord
		err := tx.Where("user_id = ? AND task_key = ? AND day_date = ?", u.ID, "watch_ad", day).First(&rec).Error
		if err == gorm.ErrRecordNotFound {
			rec = model.TaskRecord{UserID: u.ID, TaskKey: "watch_ad", DayDate: day, Progress: 0}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if rec.Progress >= limit {
			return errAdCapReached
		}
		if err := tx.Model(&rec).Update("progress", gorm.Expr("progress + 1")).Error; err != nil {
			return err
		}
		watched = rec.Progress + 1
		fresh, err := service.Credit(tx, u.ID, int64(t.RewardedAdCoins), "ad_reward", fmt.Sprintf("%s#%d", day, watched), "rewarded video ad")
		if err != nil {
			return err
		}
		balance = fresh.Coins
		return nil
	})
	if err != nil {
		if err == errAdCapReached {
			c.JSON(http.StatusOK, gin.H{"code": 5003, "msg": "daily ad limit reached", "data": gin.H{"watched": limit, "limit": limit}})
			return
		}
		fail(c, http.StatusInternalServerError, "credit failed")
		return
	}
	ok(c, gin.H{"coins": t.RewardedAdCoins, "balance": balance, "watched": watched, "limit": limit})
}

var errAdCapReached = errors.New("ad daily cap reached")

// ---------- rewarded video: AdMob SSV callback (app) ----------

const googlePublicKeyURL = "https://www.googleapis.com/admob/v1/publicKeys"

type publicKeyEntry struct {
	Key   string `json:"key"`
	KeyID string `json:"keyId"`
}

type ssvKeyCache struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

var ssvKeys = &ssvKeyCache{}
var httpClient = &http.Client{Timeout: 10 * time.Second}

func fetchAdMobKeys() (map[string]*rsa.PublicKey, error) {
	ssvKeys.mu.RLock()
	if ssvKeys.keys != nil && time.Since(ssvKeys.fetched) < 24*time.Hour {
		keys := ssvKeys.keys
		ssvKeys.mu.RUnlock()
		return keys, nil
	}
	ssvKeys.mu.RUnlock()

	resp, err := httpClient.Get(googlePublicKeyURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var entries []publicKeyEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, e := range entries {
		block, _ := pem.Decode([]byte(e.Key))
		if block == nil {
			continue
		}
		pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}
		if rsaPub, ok := pubAny.(*rsa.PublicKey); ok {
			keys[e.KeyID] = rsaPub
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("no usable admob public keys")
	}
	ssvKeys.mu.Lock()
	ssvKeys.keys, ssvKeys.fetched = keys, time.Now()
	ssvKeys.mu.Unlock()
	return keys, nil
}

// verifyAdMobSignature checks the callback signature: message = raw query
// string up to "&signature=", RSA PKCS#1 v1.5 SHA-256, key by key_id.
func verifyAdMobSignature(rawQuery string) error {
	idx := strings.Index(rawQuery, "&signature=")
	if idx < 0 {
		return errors.New("missing signature parameter")
	}
	message := rawQuery[:idx]
	sigPart := rawQuery[idx+len("&signature="):]

	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		return err
	}
	keyID := q.Get("key_id")
	if keyID == "" {
		return errors.New("missing key_id")
	}
	// Google URL-encodes the base64 signature (+ / =); undo it after Parse
	sigPart = strings.ReplaceAll(sigPart, "%2B", "+")
	sigPart = strings.ReplaceAll(sigPart, "%2F", "/")
	sigPart = strings.ReplaceAll(sigPart, "%3D", "=")
	sig, err := base64.StdEncoding.DecodeString(sigPart)
	if err != nil {
		return fmt.Errorf("bad signature encoding: %w", err)
	}
	keys, err := fetchAdMobKeys()
	if err != nil {
		return fmt.Errorf("fetch public keys: %w", err)
	}
	pub, ok := keys[keyID]
	if !ok {
		return errors.New("unknown key_id")
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig)
}

// GET /api/webhooks/admob/ssv — AdMob server-side verification for rewarded
// video. Configure the callback URL in the AdMob console with custom_data
// set to "uid=<user_id>&tenant=<slug>". Coins are credited only after the
// RSA signature verifies, and transaction_id is deduped.
func (h *Handler) AdMobSSV(c *gin.Context) {
	raw := c.Request.URL.RawQuery
	if err := verifyAdMobSignature(raw); err != nil {
		c.String(http.StatusForbidden, "signature verification failed: %v", err)
		return
	}
	q := c.Request.URL.Query()
	transactionID := q.Get("transaction_id")
	custom := q.Get("custom_data")
	if transactionID == "" {
		c.String(http.StatusBadRequest, "missing transaction_id")
		return
	}
	customQ, _ := url.ParseQuery(custom)
	uidStr := customQ.Get("uid")
	slug := customQ.Get("tenant")
	var uid uint64
	fmt.Sscanf(uidStr, "%d", &uid)
	if uid == 0 || slug == "" {
		c.String(http.StatusBadRequest, "custom_data must carry uid and tenant")
		return
	}
	var t model.Tenant
	if err := h.DB.Where("slug = ? AND status = 1", slug).First(&t).Error; err != nil {
		c.String(http.StatusNotFound, "unknown tenant")
		return
	}
	if t.RewardedAdMode != "ssv" {
		c.String(http.StatusForbidden, "tenant not in ssv mode")
		return
	}

	// dedupe: INSERT .. ON DUPLICATE fails softly; row exists → already paid
	res := h.DB.Exec(`INSERT IGNORE INTO ad_ssv_transactions (transaction_id, user_id, tenant_id, coins, created_at)
		VALUES (?, ?, ?, ?, NOW())`, transactionID, uid, t.ID, t.RewardedAdCoins)
	if res.Error != nil || res.RowsAffected == 0 {
		c.String(http.StatusOK, "duplicate; already credited")
		return
	}
	if _, err := service.Credit(h.DB, uid, int64(t.RewardedAdCoins), "ad_reward", transactionID, "rewarded video (ssv)"); err != nil {
		c.String(http.StatusInternalServerError, "credit failed")
		return
	}
	c.String(http.StatusOK, "credited")
}
