package service

import "gorm.io/gorm/clause"

// lockByPK issues SELECT ... FOR UPDATE so concurrent writers on the same
// user serialize and the coin balance can never go negative.
var lockByPK = clause.Locking{Strength: "UPDATE"}

// LockClause exposes the row lock for callers outside this package.
func LockClause() clause.Locking { return lockByPK }
