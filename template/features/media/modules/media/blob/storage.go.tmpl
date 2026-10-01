// Package blob defines owner-neutral media ports. Application services authorize access.
package blob

import (
	"context"
	"crypto/sha256"
	"errors"
	"path"
	"strings"
	"time"
)

type Key string

var ErrInvalid = errors.New("invalid media request")
var ErrUnavailable = errors.New("media storage unavailable")

func ParseKey(value string) (Key, error) {
	if value == "" || len(value) > 512 || strings.ContainsAny(value, "\\\x00\r\n") || strings.HasPrefix(value, "/") || path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return "", ErrInvalid
	}
	return Key(value), nil
}

type Storage interface {
	Put(context.Context, Key, string, []byte) error
	Get(context.Context, Key) ([]byte, error)
	Delete(context.Context, Key) error
}
type Upload struct {
	Owner        string
	Key          Key
	PublishedKey Key
	Size         int64
	ContentType  string
	ExpiresAt    time.Time
}
type Verified struct {
	Key         Key
	Size        int64
	SHA256      [32]byte
	ContentType string
}
type ContentInspector interface {
	ContentType([]byte) string
}

type Completion struct {
	Inspector ContentInspector
	Storage   Storage
	Now       func() time.Time
	MaxBytes  int64
}

// Complete accepts a trusted persisted upload intent, never client-supplied owner/expiry.
func (c Completion) Complete(ctx context.Context, u Upload, owner string) (Verified, error) {
	if c.Storage == nil || c.Inspector == nil || c.Now == nil || c.MaxBytes <= 0 || owner == "" || u.Owner != owner || !c.Now().Before(u.ExpiresAt) || u.Size < 1 || u.Size > c.MaxBytes {
		return Verified{}, ErrInvalid
	}
	if _, err := ParseKey(string(u.Key)); err != nil {
		return Verified{}, err
	}
	if _, err := ParseKey(string(u.PublishedKey)); err != nil || u.Key == u.PublishedKey {
		return Verified{}, ErrInvalid
	}
	data, err := c.Storage.Get(ctx, u.Key)
	if err != nil {
		return Verified{}, ErrUnavailable
	}
	if int64(len(data)) != u.Size {
		return Verified{}, ErrInvalid
	}
	actual := strings.Split(c.Inspector.ContentType(data), ";")[0]
	if actual != u.ContentType || !c.Now().Before(u.ExpiresAt) {
		return Verified{}, ErrInvalid
	}
	if err := c.Storage.Put(ctx, u.PublishedKey, actual, data); err != nil {
		return Verified{}, ErrUnavailable
	}
	return Verified{Key: u.PublishedKey, Size: u.Size, SHA256: sha256.Sum256(data), ContentType: actual}, nil
}

// DeleteBatch consumes an already claimed application outbox batch; failed entries remain retryable.
func DeleteBatch(ctx context.Context, storage Storage, keys []Key) ([]Key, error) {
	if storage == nil || len(keys) > 100 {
		return nil, ErrInvalid
	}
	deleted := []Key{}
	for _, key := range keys {
		if _, err := ParseKey(string(key)); err != nil {
			return deleted, err
		}
		if err := storage.Delete(ctx, key); err != nil {
			return deleted, ErrUnavailable
		}
		deleted = append(deleted, key)
	}
	return deleted, nil
}
