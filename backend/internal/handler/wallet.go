package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"viewly/internal/model"
	"viewly/internal/service"
)

// GET /api/wallet — balance, VIP status, quick stats.
func (h *Handler) Wallet(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	var earned, spent int64
	h.DB.Model(&model.CoinTransaction{}).Where("user_id = ? AND amount > 0", u.ID).Select("COALESCE(SUM(amount),0)").Scan(&earned)
	h.DB.Model(&model.CoinTransaction{}).Where("user_id = ? AND amount < 0", u.ID).Select("COALESCE(SUM(amount),0)").Scan(&spent)
	ok(c, gin.H{
		"coins": u.Coins, "is_vip": u.IsVIP(time.Now().UTC()),
		"vip_expire_at": u.VipExpireAt,
		"total_earned": earned, "total_spent": -spent,
	})
}

// GET /api/wallet/transactions
func (h *Handler) Transactions(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	page, size := pageParams(c, 20)
	var total int64
	h.DB.Model(&model.CoinTransaction{}).Where("user_id = ?", u.ID).Count(&total)
	var list []model.CoinTransaction
	h.DB.Where("user_id = ?", u.ID).Order("id DESC").Offset(offset(page, size)).Limit(size).Find(&list)
	ok(c, gin.H{"list": list, "page": page, "size": size, "total": total})
}

// GET /api/store — coin packages + VIP plans (tenant-scoped).
func (h *Handler) Store(c *gin.Context) {
	var packs []model.CoinPackage
	h.DB.Where("status = 1 AND tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&packs)
	var plans []model.VipPlan
	h.DB.Where("status = 1 AND tenant_id = ?", h.tID(c)).Order("sort ASC, id ASC").Find(&plans)
	ok(c, gin.H{"coin_packages": packs, "vip_plans": plans})
}

type createOrderReq struct {
	Kind      string `json:"kind" binding:"required,oneof=coins vip"`
	PackageID uint64 `json:"package_id" binding:"required"`
}

func genOrderNo() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("V%s%s", time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b))
}

// POST /api/orders — create a pending order. Payment confirmation arrives via
// the (mock) pay endpoint, or a real PSP webhook once integrated.
func (h *Handler) CreateOrder(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "kind must be coins|vip and package_id required")
		return
	}

	order := model.Order{TenantID: h.tID(c), OrderNo: genOrderNo(), UserID: u.ID, Kind: req.Kind, PackageID: req.PackageID, Status: "pending", Currency: "USD"}
	if req.Kind == "coins" {
		var p model.CoinPackage
		if err := h.DB.Where("id = ? AND status = 1 AND tenant_id = ?", req.PackageID, h.tID(c)).First(&p).Error; err != nil {
			fail(c, http.StatusNotFound, "package not found")
			return
		}
		order.Coins, order.BonusCoins = p.Coins, p.BonusCoins
		order.AmountCents, order.Currency = p.PriceCents, p.Currency
	} else {
		var p model.VipPlan
		if err := h.DB.Where("id = ? AND status = 1 AND tenant_id = ?", req.PackageID, h.tID(c)).First(&p).Error; err != nil {
			fail(c, http.StatusNotFound, "plan not found")
			return
		}
		order.Days = p.Days
		order.AmountCents, order.Currency = p.PriceCents, p.Currency
	}
	if err := h.DB.Create(&order).Error; err != nil {
		fail(c, http.StatusInternalServerError, "create order failed")
		return
	}
	ok(c, order)
}

// settleOrder marks an order paid and delivers coins / VIP days exactly once.
func settleOrder(tx *gorm.DB, order *model.Order) error {
	res := tx.Model(&model.Order{}).Where("id = ? AND status = 'pending'", order.ID).
		Updates(map[string]any{"status": "paid", "paid_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("order already settled")
	}
	if order.Kind == "coins" {
		total := int64(order.Coins + order.BonusCoins)
		if total > 0 {
			if _, err := service.Credit(tx, order.UserID, total, "recharge", order.OrderNo, "coin package"); err != nil {
				return err
			}
		}
	} else {
		var u model.User
		if err := tx.Clauses(service.LockClause()).Where("id = ?", order.UserID).First(&u).Error; err != nil {
			return err
		}
		base := time.Now().UTC()
		if u.VipExpireAt != nil && u.VipExpireAt.After(base) {
			base = *u.VipExpireAt
		}
		newExp := base.AddDate(0, 0, order.Days)
		if err := tx.Model(&model.User{}).Where("id = ?", u.ID).Update("vip_expire_at", newExp).Error; err != nil {
			return err
		}
	}
	return nil
}

// POST /api/orders/:order_no/mock-pay — dev-only payment simulation that
// stands in for a Stripe/PayPal webhook. Gated by server.mock_pay config.
func (h *Handler) MockPay(c *gin.Context) {
	if !h.Cfg.Server.MockPay {
		fail(c, http.StatusForbidden, "mock pay disabled")
		return
	}
	u := c.MustGet("user").(*model.User)
	var order model.Order
	if err := h.DB.Where("order_no = ? AND user_id = ?", c.Param("order_no"), u.ID).First(&order).Error; err != nil {
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
	var fresh model.User
	h.DB.First(&fresh, u.ID)
	ok(c, gin.H{"order_no": order.OrderNo, "status": "paid", "coins": fresh.Coins, "vip_expire_at": fresh.VipExpireAt})
}

// GET /api/orders/:order_no
func (h *Handler) OrderStatus(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	var order model.Order
	if err := h.DB.Where("order_no = ? AND user_id = ?", c.Param("order_no"), u.ID).First(&order).Error; err != nil {
		fail(c, http.StatusNotFound, "order not found")
		return
	}
	ok(c, order)
}

// GET /api/orders — my orders
func (h *Handler) MyOrders(c *gin.Context) {
	u := c.MustGet("user").(*model.User)
	page, size := pageParams(c, 20)
	var total int64
	h.DB.Model(&model.Order{}).Where("user_id = ?", u.ID).Count(&total)
	var list []model.Order
	h.DB.Where("user_id = ?", u.ID).Order("id DESC").Offset(offset(page, size)).Limit(size).Find(&list)
	ok(c, gin.H{"list": list, "page": page, "size": size, "total": total})
}

// POST /api/webhooks/stripe — placeholder for the real payment provider.
// TODO: verify signature, map event -> order_no (client_reference_id), call settleOrder.
func (h *Handler) StripeWebhook(c *gin.Context) {
	fail(c, http.StatusNotImplemented, "stripe webhook not integrated yet; use mock-pay in dev")
}

func orderIDFromNo(no string) uint64 {
	id, _ := strconv.ParseUint(no, 10, 64)
	return id
}
