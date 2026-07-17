package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	registry, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/generated/domains.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(registry), `"habit"`) {
		t.Fatal("added domain was not registered")
	}
}

func TestGeneratedStructuralGateRejectsSecurityDrift(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "structural")
	if err := run([]string{"new", "Structural", "--module", "example.com/structural", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	check := exec.Command("bash", "scripts/check-structure.sh")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("clean generated project failed structural gate: %v\n%s", err, output)
	}
	dependencyMock := filepath.Join(dir, "apps/web/node_modules/dependency/mock.js")
	if err := os.MkdirAll(filepath.Dir(dependencyMock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dependencyMock, []byte("export default {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("bash", "scripts/check-structure.sh")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("structural gate inspected dependency output: %v\n%s", err, output)
	}
	bad := filepath.Join(dir, "apps/api/internal/domain/example/unsafe.go")
	if err := os.WriteFile(bad, []byte("package example\nimport \"fmt\"\nfunc unsafe(){fmt.Println(\"secret\")}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("bash", "scripts/check-structure.sh")
	check.Dir = dir
	if err := check.Run(); err == nil {
		t.Fatal("structural gate accepted ad hoc printing")
	}
}

func TestGeneratedDeliveryControlsArePinnedAndConsistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "secure-delivery")
	if err := run([]string{"new", "Secure Delivery", "--module", "example.com/secure-delivery", "--output", dir}); err != nil {
		t.Fatal(err)
	}

	ci, err := os.ReadFile(filepath.Join(dir, ".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(ci)
	for _, floating := range []string{"actions/checkout@v", "actions/setup-go@v", "actions/setup-node@v", "pnpm/action-setup@v"} {
		if strings.Contains(workflow, floating) {
			t.Errorf("workflow contains floating action reference %q", floating)
		}
	}
	if !strings.Contains(workflow, "pnpm install --frozen-lockfile") {
		t.Error("CI must consume the frozen dependency lockfile")
	}
	if !strings.Contains(workflow, "run: make verify") {
		t.Error("CI must use the authoritative verification target")
	}

	toolModule, err := os.ReadFile(filepath.Join(dir, "tools/go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(toolModule), "golang.org/x/vuln v1.5.0") {
		t.Error("vulnerability scanner and its transitive graph must be pinned in the tools module")
	}
	makefile, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(makefile), "govulncheck@") {
		t.Error("security gate must use the reviewed tools module rather than an ad hoc tool download")
	}

	hook, err := os.ReadFile(filepath.Join(dir, ".git/hooks/pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hook), "make verify") {
		t.Error("pre-commit hook must use the same authoritative verification target as CI")
	}
}

func TestGeneratedAPIDoesNotOwnAuthorizationSchemaAdministration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "schema-boundary")
	if err := run([]string{"new", "Schema Boundary", "--module", "example.com/schema-boundary", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile(filepath.Join(dir, "apps/api/cmd/server/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(server), "WriteSchema") {
		t.Fatal("long-running API must not apply authorization schema")
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/cmd/schema/main.go")); err != nil {
		t.Fatal("missing one-shot schema command")
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "entrypoint: [/schema]") {
		t.Fatal("schema service must override the API image entrypoint")
	}
}

func TestGeneratedDatabaseUsesOneShotReviewedMigrations(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "database-migrations")
	if err := run([]string{"new", "Database Migrations", "--module", "example.com/database-migrations", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	store, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/gormstore/store.go"))
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(store), "AutoMigrate") {
		t.Fatal("long-running persistence adapter must not mutate schema")
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(compose), "app-migrate:") || !strings.Contains(string(compose), "entrypoint: [/migrate]") {
		t.Fatal("generated stack must apply database migrations as a one-shot service")
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000001_baseline.up.sql")); err != nil {
		t.Fatal("missing reviewed baseline migration")
	}
}
