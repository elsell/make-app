package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func inspect(path string, source []byte) []string {
	file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ImportsOnly)
	if err != nil {
		return []string{"invalid Go source"}
	}
	domain := strings.Contains(path, "/internal/domain/")
	application := strings.Contains(path, "/internal/app/")
	dto := strings.Contains(path, "/httpserver/") && strings.Contains(path, "/dto/")
	findings := []string{}
	for _, imp := range file.Imports {
		name, _ := strconv.Unquote(imp.Path.Value)
		infrastructure := strings.HasPrefix(name, "gorm.io/") || strings.Contains(name, "/huma/") || strings.Contains(name, "/authzed/") || name == "net/http" || name == "database/sql" || strings.Contains(name, "/internal/adapters/")
		if ((domain || application) && infrastructure) || (domain && strings.Contains(name, "/internal/app/")) || (dto && (strings.Contains(name, "/internal/domain/") || strings.Contains(name, "/internal/app/") || strings.Contains(name, "/internal/ports"))) {
			findings = append(findings, "forbidden import "+name)
		}
	}
	return findings
}
func main() {
	failed := false
	err := filepath.WalkDir("apps/api/internal", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, finding := range inspect(filepath.ToSlash(path), body) {
			fmt.Fprintln(os.Stderr, path+": "+finding)
			failed = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}
