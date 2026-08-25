package service

import (
	"errors"

	"gorm.io/gorm"

	"viewly/internal/model"
)

var (
	ErrInsufficientCoins = errors.New("insufficient coins")
)

// Credit adds coins to a user inside the given transaction and appends a
// ledger row. caller owns tx commit/rollback.
func Credit(tx *gorm.DB, userID uint64, amount int64, bizType, bizID, remark string) (*model.User, error) {
	if amount <= 0 {
		return nil, errors.New("credit amount must be positive")
	}
	var u model.User
	if err := tx.Clauses(lockByPK).Where("id = ?", userID).First(&u).Error; err != nil {
		return nil, err
	}
	newBal := u.Coins + amount
	if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("coins", newBal).Error; err != nil {
		return nil, err
	}
	led := model.CoinTransaction{
		UserID: userID, Amount: amount, BalanceAfter: newBal,
		BizType: bizType, BizID: bizID, Remark: remark,
	}
	if err := tx.Create(&led).Error; err != nil {
		return nil, err
	}
	u.Coins = newBal
	return &u, nil
}

// Debit removes coins from a user inside the given transaction, refusing to go
// below zero, and appends a ledger row with a negative amount.
func Debit(tx *gorm.DB, userID uint64, amount int64, bizType, bizID, remark string) (*model.User, error) {
	if amount <= 0 {
		return nil, errors.New("debit amount must be positive")
	}
	var u model.User
	if err := tx.Clauses(lockByPK).Where("id = ?", userID).First(&u).Error; err != nil {
		return nil, err
	}
	if u.Coins < amount {
		return nil, ErrInsufficientCoins
	}
	newBal := u.Coins - amount
	if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("coins", newBal).Error; err != nil {
		return nil, err
	}
	led := model.CoinTransaction{
		UserID: userID, Amount: -amount, BalanceAfter: newBal,
		BizType: bizType, BizID: bizID, Remark: remark,
	}
	if err := tx.Create(&led).Error; err != nil {
		return nil, err
	}
	u.Coins = newBal
	return &u, nil
}
