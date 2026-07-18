package auditlimit

import (
	"testing"
	"time"
)

func TestLimiterBoundsAuditProducingOperationsPerPrincipal(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	limiter := New(2, time.Minute, 2)
	if !limiter.Allow("one", now) || !limiter.Allow("one", now) || limiter.Allow("one", now) {
		t.Fatal("principal exceeded configured audit write rate")
	}
	if !limiter.Allow("two", now) || !limiter.Allow("three", now) {
		t.Fatal("independent principals were not admitted")
	}
	if !limiter.Allow("one", now.Add(time.Minute)) {
		t.Fatal("new window did not restore capacity")
	}
}
