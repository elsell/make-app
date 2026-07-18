package auditlimit

import (
	"sync"
	"time"
)

type entry struct {
	windowStart time.Time
	lastSeen    time.Time
	count       int
}

type Limiter struct {
	mu            sync.Mutex
	entries       map[string]entry
	limit         int
	window        time.Duration
	maxPrincipals int
}

func New(limit int, window time.Duration, maxPrincipals int) *Limiter {
	return &Limiter{entries: make(map[string]entry), limit: limit, window: window, maxPrincipals: maxPrincipals}
}

func (l *Limiter) Allow(principal string, now time.Time) bool {
	if principal == "" || l.limit <= 0 || l.window <= 0 || l.maxPrincipals <= 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	current := now.Truncate(l.window)
	value, exists := l.entries[principal]
	if !exists && len(l.entries) >= l.maxPrincipals {
		var oldestID string
		var oldest time.Time
		for id, candidate := range l.entries {
			if oldestID == "" || candidate.lastSeen.Before(oldest) {
				oldestID, oldest = id, candidate.lastSeen
			}
		}
		delete(l.entries, oldestID)
	}
	if !exists || !value.windowStart.Equal(current) {
		l.entries[principal] = entry{windowStart: current, lastSeen: now, count: 1}
		return true
	}
	if value.count >= l.limit {
		value.lastSeen = now
		l.entries[principal] = value
		return false
	}
	value.count++
	value.lastSeen = now
	l.entries[principal] = value
	return true
}
