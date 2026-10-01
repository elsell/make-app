package notifications

import (
	"context"
	"errors"
	"testing"
	"time"
)

type controlledProvider struct {
	calls  int
	result Result
	err    error
}

func (p *controlledProvider) Send(context.Context, Message) (Result, error) {
	p.calls++
	return p.result, p.err
}
func TestBoundedDelivery(t *testing.T) {
	p := &controlledProvider{result: Result{State: Retryable, RetryAfter: time.Second}}
	sleeps := 0
	r := Delivery{Provider: p, MaxAttempts: 3, MaxDelay: 2 * time.Second, Wait: func(context.Context, time.Duration) error { sleeps++; return nil }}
	result, err := r.Send(context.Background(), Message{Token: "secret", ID: "n1"})
	if err != nil || result.State != Retryable || p.calls != 3 || sleeps != 2 {
		t.Fatalf("retry bound failed: %+v %v %d", result, err, p.calls)
	}
	p.calls = 0
	p.result = Result{State: InvalidDevice}
	r.Send(context.Background(), Message{Token: "secret", ID: "n1"})
	if p.calls != 1 {
		t.Fatal("retried invalid token")
	}
	p.calls = 0
	p.err = errors.New("private provider response")
	_, err = r.Send(context.Background(), Message{Token: "secret", ID: "n1"})
	if err == nil || err.Error() == p.err.Error() {
		t.Fatal("unsafe provider error")
	}
}
