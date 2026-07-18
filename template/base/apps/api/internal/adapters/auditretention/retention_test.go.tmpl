package auditretention

import (
	"context"
	"testing"
	"time"
)

func TestRetentionRejectsUnsafeBoundsBeforeStorage(t *testing.T) {
	runner := Runner{}
	if _, err := runner.DeleteBatch(context.Background(), time.Now().Add(-time.Hour), time.Now(), 0); err == nil {
		t.Fatal("zero-sized retention batch was accepted")
	}
	if _, err := runner.DeleteBatch(context.Background(), time.Now(), time.Now().Add(-time.Hour), 10); err == nil {
		t.Fatal("future cutoff was accepted")
	}
}
