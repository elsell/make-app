package audit

import (
	"testing"
	"time"
)

func TestEventValidationRejectsUnknownTaxonomy(t *testing.T) {
	valid := Event{ID: "event", OwnerUserID: "owner", ActorUserID: "actor", Action: ResourceViewed, TargetType: "habit", TargetID: "habit", Outcome: Succeeded, CorrelationID: "request", OccurredAt: time.Now()}
	if !valid.Valid() {
		t.Fatal("known complete event was rejected")
	}
	unknownAction := valid
	unknownAction.Action = "resource.maybe"
	if unknownAction.Valid() {
		t.Fatal("unknown audit action was accepted")
	}
	unknownOutcome := valid
	unknownOutcome.Outcome = "maybe"
	if unknownOutcome.Valid() {
		t.Fatal("unknown audit outcome was accepted")
	}
}
