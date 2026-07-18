package ratelimit

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const shardCount = 64

type shardModel struct {
	Scope string `gorm:"primaryKey"`
	Shard int    `gorm:"primaryKey"`
}

func (shardModel) TableName() string { return "rate_limit_shard_models" }

type windowModel struct {
	Scope         string `gorm:"primaryKey"`
	PrincipalHash string `gorm:"primaryKey"`
	Shard         int
	WindowStart   time.Time
	RequestCount  int
	UpdatedAt     time.Time
}

func (windowModel) TableName() string { return "rate_limit_window_models" }

type Limiter struct {
	db            *gorm.DB
	scope         string
	limit         int
	window        time.Duration
	maxPrincipals int
}

func New(db *gorm.DB, scope string, limit int, window time.Duration, maxPrincipals int) (*Limiter, error) {
	if db == nil || (scope != "audit" && scope != "request") || limit <= 0 || window <= 0 || maxPrincipals < shardCount {
		return nil, errors.New("invalid shared rate limiter configuration")
	}
	return &Limiter{db: db, scope: scope, limit: limit, window: window, maxPrincipals: maxPrincipals}, nil
}

func (l *Limiter) Allow(principal string, now time.Time) bool {
	if principal == "" || now.IsZero() {
		return false
	}
	digest := sha256.Sum256([]byte(principal))
	principalHash := hex.EncodeToString(digest[:])
	shard := int(binary.BigEndian.Uint64(digest[:8]) % shardCount)
	allowed := false
	err := l.db.Transaction(func(tx *gorm.DB) error {
		var lock shardModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("scope = ? AND shard = ?", l.scope, shard).First(&lock).Error; err != nil {
			return err
		}
		cutoff := now.Add(-2 * l.window)
		if err := tx.Where("scope = ? AND shard = ? AND updated_at < ?", l.scope, shard, cutoff).Delete(&windowModel{}).Error; err != nil {
			return err
		}
		var row windowModel
		err := tx.Where("scope = ? AND principal_hash = ?", l.scope, principalHash).First(&row).Error
		current := now.UTC().Truncate(l.window)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var count int64
			if err := tx.Model(&windowModel{}).Where("scope = ? AND shard = ?", l.scope, shard).Count(&count).Error; err != nil {
				return err
			}
			perShard := (l.maxPrincipals + shardCount - 1) / shardCount
			if count >= int64(perShard) {
				return nil
			}
			row = windowModel{Scope: l.scope, PrincipalHash: principalHash, Shard: shard, WindowStart: current, RequestCount: 1, UpdatedAt: now.UTC()}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			allowed = true
			return nil
		}
		if err != nil {
			return err
		}
		if !row.WindowStart.Equal(current) {
			allowed = true
			return updateWindow(tx, &row, map[string]any{"window_start": current, "request_count": 1, "updated_at": now.UTC()})
		}
		if row.RequestCount >= l.limit {
			return updateWindow(tx, &row, map[string]any{"updated_at": now.UTC()})
		}
		allowed = true
		return updateWindow(tx, &row, map[string]any{"request_count": row.RequestCount + 1, "updated_at": now.UTC()})
	})
	return err == nil && allowed
}

func updateWindow(tx *gorm.DB, row *windowModel, values map[string]any) error {
	result := tx.Model(row).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("shared rate limit window update affected an unexpected row count")
	}
	return nil
}
