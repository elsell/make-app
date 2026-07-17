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
	for _, path := range []string{"compose.yaml", "apps/api/internal/domain/example/entity.go", "apps/mobile/package.json", ".github/workflows/ci.yml", "packages/i18n/src/index.ts", "scripts/check-i18n.mjs"} {
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

func TestGeneratedI18nIsMandatoryAndRejectsLiteralUICopy(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "localized")
	if err := run([]string{"new", "Localized", "--module", "example.com/localized", "--output", dir}); err != nil {
		t.Fatal(err)
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "All user-facing copy") || !strings.Contains(string(agents), "locale catalog") {
		t.Fatal("generated guidance must require internationalized user-facing copy")
	}

	check := exec.Command("node", "scripts/check-i18n.mjs")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("clean generated project failed i18n gate: %v\n%s", err, output)
	}

	page := filepath.Join(dir, "apps/web/src/routes/+page.svelte")
	contents, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	contents = append(contents, []byte("\n<p>Untranslated settings</p>\n")...)
	if err := os.WriteFile(page, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("node", "scripts/check-i18n.mjs")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err == nil {
		t.Fatalf("i18n gate accepted literal user-facing copy:\n%s", output)
	}
	if err := os.WriteFile(page, contents[:len(contents)-len("\n<p>Untranslated settings</p>\n")], 0o644); err != nil {
		t.Fatal(err)
	}

	mobile := filepath.Join(dir, "apps/mobile/app/index.tsx")
	mobileContents, err := os.ReadFile(mobile)
	if err != nil {
		t.Fatal(err)
	}
	mobileContents = append(mobileContents, []byte("\nconst untranslated = <Text>Untranslated settings</Text>;\n")...)
	if err := os.WriteFile(mobile, mobileContents, 0o644); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("node", "scripts/check-i18n.mjs")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err == nil {
		t.Fatalf("i18n gate accepted literal mobile copy:\n%s", output)
	}
	if err := os.WriteFile(mobile, mobileContents[:len(mobileContents)-len("\nconst untranslated = <Text>Untranslated settings</Text>;\n")], 0o644); err != nil {
		t.Fatal(err)
	}

	spanish := filepath.Join(dir, "packages/i18n/src/locales/es.json")
	spanishContents, err := os.ReadFile(spanish)
	if err != nil {
		t.Fatal(err)
	}
	brokenCatalog := strings.Replace(string(spanishContents), "  \"app.ready\": \"Tu aplicación generada está lista.\",\n", "", 1)
	if brokenCatalog == string(spanishContents) {
		t.Fatal("test fixture could not remove the Spanish catalog key")
	}
	if err := os.WriteFile(spanish, []byte(brokenCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("node", "scripts/check-i18n.mjs")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err == nil {
		t.Fatalf("i18n gate accepted an incomplete locale catalog:\n%s", output)
	}
}

func TestGeneratedI18nGateRejectsCommonCopyBypasses(t *testing.T) {
	fixtures := map[string]string{
		"apps/web/src/routes/bypass.svelte":     `<script>const buttonLabel = 'Delete account';</script><button>{buttonLabel}</button>`,
		"apps/web/src/lib/copy.ts":              `export const dialogTitle = 'Delete account';`,
		"apps/web/src/lib/warning.ts":           `export const warning = 'Delete account';`,
		"apps/mobile/components/expression.tsx": `export const Copy = () => <Text>{'Delete account'}</Text>;`,
		"apps/mobile/src/forms/attribute.jsx":   `export const Copy = () => <Input placeholder={'Email address'} />;`,
		"apps/mobile/src/forms/ternary.tsx":     `export const Copy = ({ danger }) => <Text>{danger ? 'Delete account' : 'Keep account'}</Text>;`,
	}
	for path, fixture := range fixtures {
		t.Run(path, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "copy-bypass")
			if err := run([]string{"new", "Copy Bypass", "--module", "example.com/copy-bypass", "--output", dir}); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, path)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte(fixture), 0o644); err != nil {
				t.Fatal(err)
			}
			check := exec.Command("node", "scripts/check-i18n.mjs")
			check.Dir = dir
			if output, err := check.CombinedOutput(); err == nil {
				t.Fatalf("i18n gate accepted copy bypass:\n%s", output)
			}
		})
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
	if !strings.Contains(workflow, "run: make acceptance") {
		t.Error("CI must run live authentication and authorization acceptance")
	}
	liveScript := filepath.Join(dir, "scripts/live-acceptance.sh")
	info, err := os.Stat(liveScript)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("live acceptance script must be executable")
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
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(store), "AutoMigrate") {
		t.Fatal("long-running persistence adapter must not mutate schema")
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "app-migrate:") || !strings.Contains(string(compose), "entrypoint: [/migrate]") {
		t.Fatal("generated stack must apply database migrations as a one-shot service")
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000001_baseline.up.sql")); err != nil {
		t.Fatal("missing reviewed baseline migration")
	}
}

func TestGeneratedWebComposeStartupIsNonInteractive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "web-compose")
	if err := run([]string{"new", "Web Compose", "--module", "example.com/web-compose", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "CI: \"true\"") {
		t.Fatal("generated web service must make pnpm installation non-interactive")
	}
	if !strings.Contains(string(compose), `user: "${MAKE_APP_UID:-1000}:${MAKE_APP_GID:-1000}"`) {
		t.Fatal("generated web service must not create root-owned host artifacts")
	}
	for _, setting := range []string{"HOME: /tmp/make-app-home", "XDG_CACHE_HOME: /tmp/make-app-cache", "COREPACK_HOME: /tmp/make-app-corepack"} {
		if !strings.Contains(string(compose), setting) {
			t.Fatalf("generated web service must support arbitrary numeric users with %s", setting)
		}
	}
	npmrc, err := os.ReadFile(filepath.Join(dir, ".npmrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(npmrc), "store-dir=.pnpm-store") {
		t.Fatal("host bootstrap and Compose must share a repository-local pnpm store")
	}
	liveAcceptance, err := os.ReadFile(filepath.Join(dir, "scripts/live-acceptance.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(liveAcceptance), "seq 1 600") {
		t.Fatal("live acceptance must allow a bounded slow-registry cold start")
	}
	if !strings.Contains(string(liveAcceptance), "node scripts/scalar-browser-acceptance.mjs") || strings.Contains(string(liveAcceptance), "pnpm exec node scripts/scalar-browser-acceptance.mjs") {
		t.Fatal("live browser acceptance must not trigger pnpm reconciliation while Compose is serving")
	}
	if !strings.Contains(string(liveAcceptance), `MAKE_APP_UID="${MAKE_APP_UID:-$(id -u)}"`) || !strings.Contains(string(liveAcceptance), `MAKE_APP_GID="${MAKE_APP_GID:-$(id -g)}"`) {
		t.Fatal("live acceptance must derive the Compose user from the host user")
	}
	if !strings.Contains(string(liveAcceptance), "--user 1001:1001") || !strings.Contains(string(liveAcceptance), "COREPACK_HOME=/tmp/make-app-corepack") {
		t.Fatal("live acceptance must exercise Corepack under a non-1000 numeric user")
	}
	makefile, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(makefile), `MAKE_APP_UID=$$(id -u) MAKE_APP_GID=$$(id -g) docker compose up --build`) {
		t.Fatal("the supported Compose entrypoint must work for non-1000 host users")
	}
}
