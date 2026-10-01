// Package notifications contains optional delivery mechanics, not recipient policy.
package notifications

import (
	"context"
	"errors"
	"time"
)

type State string

const (
	Accepted      State = "accepted"
	Retryable     State = "retryable"
	InvalidDevice State = "invalid_device"
	Rejected      State = "rejected"
)

type Message struct{ ID, Token, Title, Body string }
type Result struct {
	State      State
	ReceiptID  string
	RetryAfter time.Duration
}
type Provider interface {
	Send(context.Context, Message) (Result, error)
}
type ReceiptReader interface {
	Receipts(context.Context, []string) (map[string]Result, error)
}

var ErrUnavailable = errors.New("notification provider unavailable")
var ErrConfiguration = errors.New("invalid notification delivery configuration")

type Delivery struct {
	Provider    Provider
	MaxAttempts int
	MaxDelay    time.Duration
	Wait        func(context.Context, time.Duration) error
}

func (d Delivery) Send(ctx context.Context, message Message) (Result, error) {
	if d.Provider == nil || d.MaxAttempts < 1 || d.MaxAttempts > 5 || d.MaxDelay <= 0 || d.Wait == nil || message.Token == "" || message.ID == "" {
		return Result{}, ErrConfiguration
	}
	var result Result
	var lastError error
	for attempt := 0; attempt < d.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		value, err := d.Provider.Send(ctx, message)
		lastError = nil
		if err != nil {
			if ctx.Err() != nil {
				return Result{}, ctx.Err()
			}
			value = Result{State: Retryable}
			lastError = ErrUnavailable
		}
		result = value
		switch result.State {
		case Accepted, InvalidDevice, Rejected:
			return result, nil
		case Retryable:
		default:
			return Result{}, ErrUnavailable
		}
		if attempt+1 < d.MaxAttempts {
			delay := result.RetryAfter
			if delay <= 0 {
				delay = time.Second * time.Duration(1<<attempt)
			}
			if delay > d.MaxDelay {
				delay = d.MaxDelay
			}
			if err := d.Wait(ctx, delay); err != nil {
				return Result{}, err
			}
		}
	}
	return result, lastError
}
