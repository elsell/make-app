package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppAndDomain(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "habit-kit")
	err := run([]string{"new", "Habit Kit", "--module", "example.com/habit-kit", "--output", dir})
	if err != nil {
		t.Fatalf("unexpected result: %v", err)
	}
	for _, path := range []string{"compose.yaml", "apps/api/internal/domain/example/entity.go", "apps/mobile/package.json", ".github/workflows/ci.yml"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("missing %s: %v", path, err)
		}
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/domain/habit/entity.go")); err != nil {
		t.Fatal(err)
	}
}
