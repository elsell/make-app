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
	for _, path := range []string{
		"apps/api/internal/domain/habit/entity.go",
		"apps/api/internal/app/habit/ports.go",
		"apps/api/internal/adapters/gormstore/habit/repository.go",
		"apps/api/internal/adapters/httpserver/habit/dto/habit.go",
		"apps/api/internal/adapters/httpserver/habit/mapper/habit.go",
		"apps/api/internal/adapters/httpserver/habit/routes/routes.go",
		"apps/api/internal/adapters/dbmigrations/000010_create_habits.up.sql",
		"apps/api/internal/adapters/dbmigrations/000010_create_habits.down.sql",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("domain vertical slice missing %s: %v", path, err)
		}
	}
	registry, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/generated/domains.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(registry), `"habit"`) {
		t.Fatal("added domain was incorrectly registered against generic resource routes")
	}
	repository, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/gormstore/habit/repository.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, invariant := range []string{"idempotency_models", "authorization_outbox_models", "ports.AuthorizationChange", "ports.Idempotency", "CreatedAt", "AuthorizationDelete"} {
		if !strings.Contains(string(repository), invariant) {
			t.Fatalf("generated domain create cannot preserve transactional platform invariant %q", invariant)
		}
	}
	if err := run([]string{"domain", "add", "journal", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000011_create_journals.up.sql")); err != nil {
		t.Fatalf("second domain did not receive the next migration version: %v", err)
	}
}

func TestDomainAddRollsBackPartialGeneration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "atomic")
	if err := run([]string{"new", "Atomic", "--module", "example.com/atomic", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	migrationsPath := filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/migrations.go")
	original, err := os.ReadFile(migrationsPath)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(original), "const LatestVersion uint = 9", "const MissingVersion uint = 9", 1)
	if err := os.WriteFile(migrationsPath, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err == nil {
		t.Fatal("domain add unexpectedly accepted missing migration metadata")
	}
	for _, path := range []string{
		"apps/api/internal/domain/habit",
		"apps/api/internal/adapters/gormstore/habit",
		"apps/api/internal/adapters/dbmigrations/000010_create_habits.up.sql",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("failed domain add left partial path %s: %v", path, err)
		}
	}
	if err := os.WriteFile(migrationsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err != nil {
		t.Fatalf("domain add could not be retried after rollback: %v", err)
	}
}

func TestGeneratedAuditLogIsMandatoryAndAppendOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "audited")
	if err := run([]string{"new", "Audited", "--module", "example.com/audited", "--output", dir}); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"specs/audit/audit.spec.md",
		"apps/api/internal/domain/audit/event.go",
		"apps/api/internal/adapters/dbmigrations/000004_create_audit_events.up.sql",
		"apps/api/internal/adapters/dbmigrations/000004_create_audit_events.down.sql",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("generated audit primitive is missing %s: %v", path, err)
		}
	}

	server, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/httpserver/audit_routes.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(server), `Path: "/v1/audit-events"`) {
		t.Fatal("generated API must expose authorized audit history")
	}

	migration, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000004_create_audit_events.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, invariant := range []string{"audit_event_models", "BEFORE UPDATE OR DELETE", "audit events are append-only"} {
		if !strings.Contains(string(migration), invariant) {
			t.Errorf("audit migration does not enforce %q", invariant)
		}
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "Every authenticated domain read, list, state change, and denied authorization") {
		t.Fatal("generated guidance must make audit coverage mandatory")
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
	for _, path := range []string{"pnpm-lock.yaml", "apps/web/Dockerfile", ".dockerignore", ".github/workflows/release.yml", "scripts/plan-release.sh"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("generated delivery primitive missing %s: %v", path, err)
		}
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

func TestGeneratedWebComposeUsesProductionImage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "web-compose")
	if err := run([]string{"new", "Web Compose", "--module", "example.com/web-compose", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, setting := range []string{"dockerfile: apps/web/Dockerfile", "PORT: \"5173\""} {
		if !strings.Contains(string(compose), setting) {
			t.Fatalf("generated web service must use production image setting %s", setting)
		}
	}
	if strings.Contains(string(compose), "volumes: [\".:/workspace\"]") || strings.Contains(string(compose), "pnpm --dir apps/web dev") {
		t.Fatal("generated web service must not substitute a bind-mounted development server for the production artifact")
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
	scalarAcceptance, err := os.ReadFile(filepath.Join(dir, "scripts/scalar-browser-acceptance.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(scalarAcceptance), `get \/v1\/me\)`) {
		t.Fatal("Scalar browser acceptance must select GET /v1/me without matching account deletion")
	}
	if !strings.Contains(string(scalarAcceptance), "Web browser OIDC and application-session acceptance passed") || !strings.Contains(string(scalarAcceptance), "url.pathname.includes('/dex/auth/')") || !strings.Contains(string(scalarAcceptance), "waitForURL(`${webBaseURL}/`)") {
		t.Fatal("browser acceptance must complete generated web OIDC callback and authenticated rendering")
	}
	dexConfig, err := os.ReadFile(filepath.Join(dir, "deploy/dex/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dexConfig), "allowedOrigins: [http://localhost:5173]") {
		t.Fatal("bundled Dex must permit the generated public web client's token exchange origin")
	}
	svelteConfig, err := os.ReadFile(filepath.Join(dir, "apps/web/svelte.config.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svelteConfig), "mode: 'nonce'") || strings.Contains(string(svelteConfig), "'unsafe-inline'") {
		t.Fatal("generated web CSP must use framework-managed nonces without unsafe-inline scripts")
	}
	if strings.Count(string(liveAcceptance), `owner_token="$(session `) < 2 || strings.Contains(string(liveAcceptance), `owner_token="$(token `) {
		t.Fatal("live acceptance must exchange OIDC credentials for application sessions, including after restart")
	}
	if !strings.Contains(string(liveAcceptance), `MAKE_APP_UID="${MAKE_APP_UID:-$(id -u)}"`) || !strings.Contains(string(liveAcceptance), `MAKE_APP_GID="${MAKE_APP_GID:-$(id -g)}"`) {
		t.Fatal("live acceptance must derive the Compose user from the host user")
	}
	if !strings.Contains(string(liveAcceptance), `COMPOSE_PROJECT_NAME="make-app-acceptance-`) || strings.Count(string(liveAcceptance), "docker compose down --volumes --remove-orphans") < 2 {
		t.Fatal("live acceptance must isolate each Compose project and clean it before and after execution")
	}
	stopAPI := strings.Index(string(liveAcceptance), "docker compose stop api")
	postgresTests := strings.Index(string(liveAcceptance), "go test ./apps/api/internal/adapters/gormstore")
	restartAPI := strings.Index(string(liveAcceptance), "docker compose up -d api")
	if stopAPI < 0 || postgresTests < stopAPI || restartAPI < postgresTests {
		t.Fatal("live acceptance must isolate PostgreSQL adapter tests from the authorization worker and restart the API afterward")
	}
	if !strings.Contains(string(liveAcceptance), "<<'AUDIT_SQL' | grep -qx 1") {
		t.Fatal("live acceptance must pass owner identifiers to psql through variable-aware standard input")
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
