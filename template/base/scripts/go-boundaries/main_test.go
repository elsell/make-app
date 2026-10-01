package main

import "testing"

func TestImportBoundaries(t *testing.T) {
	cases := []struct {
		path, source string
		bad          bool
	}{
		{"apps/api/internal/domain/note/entity.go", "package note; import \"net/http\"", true},
		{"apps/api/internal/domain/note/entity.go", "package note; import \"time\"", false},
		{"apps/api/internal/adapters/httpserver/note/dto/note.go", "package dto; import \"example.com/app/internal/app/note\"", true},
		{"apps/api/internal/app/note/service.go", "package note; import \"gorm.io/gorm\"", true},
	}
	for _, c := range cases {
		if (len(inspect(c.path, []byte(c.source))) > 0) != c.bad {
			t.Fatalf("wrong boundary result: %s", c.path)
		}
	}
}
