package blob

import (
	"context"
	"errors"
	"testing"
	"time"
)

type textInspector struct{}

func (textInspector) ContentType([]byte) string { return "text/plain" }

type memory struct{ objects map[Key][]byte }

func (m memory) Put(_ context.Context, k Key, _ string, b []byte) error {
	if _, exists := m.objects[k]; exists {
		return errors.New("exists")
	}
	m.objects[k] = append([]byte{}, b...)
	return nil
}
func (m memory) Get(_ context.Context, k Key) ([]byte, error) { return m.objects[k], nil }
func (m memory) Delete(_ context.Context, k Key) error        { delete(m.objects, k); return nil }
func TestKeysAndUploadCompletion(t *testing.T) {
	for _, value := range []string{"../secret", "/root", "a/../../b", "a\\b", "", "a//b"} {
		if _, err := ParseKey(value); err == nil {
			t.Fatalf("unsafe key %q", value)
		}
	}
	key, _ := ParseKey("owners/a/file")
	now := time.Unix(100, 0)
	store := memory{objects: map[Key][]byte{key: []byte("hello")}}
	upload := Upload{Owner: "alice", Key: key, PublishedKey: Key("published/file"), Size: 5, ExpiresAt: now.Add(time.Minute), ContentType: "text/plain"}
	service := Completion{Inspector: textInspector{}, Storage: store, Now: func() time.Time { return now }, MaxBytes: 10}
	if _, err := service.Complete(context.Background(), upload, "bob"); err == nil {
		t.Fatal("wrong owner accepted")
	}
	if _, err := service.Complete(context.Background(), upload, "alice"); err != nil {
		t.Fatal(err)
	}
	store.objects[key] = []byte("evil!")
	if string(store.objects[upload.PublishedKey]) != "hello" {
		t.Fatal("upload replay changed published content")
	}
	if _, err := service.Complete(context.Background(), upload, "alice"); err == nil {
		t.Fatal("published key overwritten")
	}
	upload.Size = 4
	if _, err := service.Complete(context.Background(), upload, "alice"); err == nil {
		t.Fatal("size mismatch accepted")
	}
	upload.Size = 5
	now = now.Add(time.Minute)
	if _, err := service.Complete(context.Background(), upload, "alice"); err == nil {
		t.Fatal("expired upload accepted")
	}
}
