package auditretention

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var databaseDSN = flag.String("database-dsn", "", "migration-role PostgreSQL DSN")
var retentionDSN = flag.String("retention-dsn", "", "retention-role PostgreSQL DSN")

func TestRetentionRoleDeletesOnlyEligibleDetailsAndAppendsSummary(t *testing.T) {
	if *databaseDSN == "" || *retentionDSN == "" {
		t.Skip("database and retention DSNs are required")
	}
	admin, err := gorm.Open(postgres.Open(*databaseDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	retention, err := gorm.Open(postgres.Open(*retentionDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.NewString()
	user := map[string]any{"id": userID, "email": "", "display_name": "", "status": "active", "invitation_admin": false, "created_at": now, "updated_at": now}
	if err := admin.Table("user_models").Create(user).Error; err != nil {
		t.Fatal(err)
	}
	oldID, newID := uuid.NewString(), uuid.NewString()
	for id, occurred := range map[string]time.Time{oldID: now.Add(-400 * 24 * time.Hour), newID: now} {
		event := map[string]any{"id": id, "owner_user_id": userID, "actor_user_id": userID, "action": "resource.viewed", "target_type": "example", "target_id": id, "outcome": "succeeded", "correlation_id": uuid.NewString(), "occurred_at": occurred}
		if err := admin.Table("audit_event_models").Create(event).Error; err != nil {
			t.Fatal(err)
		}
	}
	runner := Runner{DB: retention}
	count, err := runner.Count(context.Background(), now.Add(-365*24*time.Hour))
	if err != nil || count != 1 {
		t.Fatalf("retention dry run failed: count=%d err=%v", count, err)
	}
	deleted, err := runner.DeleteBatch(context.Background(), now.Add(-365*24*time.Hour), now.Add(time.Second), 100)
	if err != nil || deleted != 1 {
		t.Fatalf("retention batch failed: deleted=%d err=%v", deleted, err)
	}
	var oldCount, newCount, summaries int64
	admin.Table("audit_event_models").Where("id = ?", oldID).Count(&oldCount)
	admin.Table("audit_event_models").Where("id = ?", newID).Count(&newCount)
	admin.Table("audit_retention_run_models").Where("deleted_count = ?", 1).Count(&summaries)
	if oldCount != 0 || newCount != 1 || summaries < 2 {
		t.Fatalf("retention boundary drifted: old=%d new=%d summaries=%d", oldCount, newCount, summaries)
	}

	var forbiddenEvent map[string]any
	if err := retention.Table("audit_event_models").Where("id = ?", newID).Take(&forbiddenEvent).Error; err == nil {
		t.Fatal("retention role unexpectedly read audit detail")
	}
	if err := retention.Table("audit_event_models").Where("id = ?", newID).Delete(&struct{ ID string }{}).Error; err == nil {
		t.Fatal("retention role unexpectedly deleted audit detail directly")
	}
	if err := retention.Model(&runModel{}).Where("deleted_count = ?", 1).Update("deleted_count", 99).Error; err == nil {
		t.Fatal("retention role unexpectedly mutated immutable retention history")
	}
	admin.Table("audit_event_models").Where("id = ?", newID).Count(&newCount)
	if newCount != 1 {
		t.Fatal("direct least-privilege denial changed the fresh audit event")
	}
}
