package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			result[relative+"/"] = info.Mode().String()
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[relative] = info.Mode().String() + "\x00" + string(body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNewIsAtomicVersionedAndUsesMain(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "atomic-new")
	originalCommandName := commandName
	commandName = func(name string) string {
		if name == "git" {
			return filepath.Join(root, "missing-git")
		}
		return name
	}
	err := run([]string{"new", "Atomic New", "--module", "example.com/atomic-new", "--output", dir})
	commandName = originalCommandName
	if err == nil {
		t.Fatal("new unexpectedly succeeded without its late Git prerequisite")
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("failed new left a destination that blocks retry: %v", statErr)
	}
	if err := run([]string{"new", "Atomic New", "--module", "example.com/atomic-new", "--output", dir}); err != nil {
		t.Fatalf("new could not be retried after late failure: %v", err)
	}
	branch := exec.Command("git", "branch", "--show-current")
	branch.Dir = dir
	output, err := branch.Output()
	if err != nil || strings.TrimSpace(string(output)) != "main" {
		t.Fatalf("generated repository did not initialize main: output=%q err=%v", output, err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dir, ".make-app.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest projectManifest
	if json.Unmarshal(manifestBytes, &manifest) != nil || manifest.SchemaVersion != templateSchemaVersion || manifest.GeneratorVersion == "" || manifest.Module != "example.com/atomic-new" || len(manifest.Domains) != 1 {
		t.Fatalf("generated compatibility manifest is incomplete: %s", manifestBytes)
	}
}

func TestNewRejectsUnsafeApplicationNamesBeforeWriting(t *testing.T) {
	for _, name := range []string{"", "<script>alert(1)</script>", "bad\nname", strings.Repeat("a", 81)} {
		dir := filepath.Join(t.TempDir(), "unsafe")
		if err := run([]string{"new", name, "--module", "example.com/unsafe", "--output", dir}); err == nil {
			t.Fatalf("unsafe application name was accepted: %q", name)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("unsafe name wrote destination: %q: %v", name, err)
		}
	}
}

func TestNewRejectsInvalidGoModulePathsBeforeWriting(t *testing.T) {
	for _, modulePath := range []string{"example.com/foo@bar", "example.com/foo:bar", "example.com/foo\nbar", "/example.com/foo", "example.com/foo/"} {
		dir := filepath.Join(t.TempDir(), "unsafe-module")
		if err := run([]string{"new", "Bad Module", "--module", modulePath, "--output", dir}); err == nil {
			t.Fatalf("invalid Go module path was accepted: %q", modulePath)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("invalid module path wrote destination: %q: %v", modulePath, err)
		}
	}
}

func TestNewCanOmitExampleAndMutationsRejectIncompatibleProjects(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "blank")
	if err := run([]string{"new", "Blank", "--module", "example.com/blank", "--output", dir, "--without-example"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/domain/example")); !os.IsNotExist(err) {
		t.Fatalf("--without-example generated example source: %v", err)
	}
	baseline, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000001_baseline.up.sql"))
	if err != nil || strings.Contains(string(baseline), "resource_models") {
		t.Fatalf("--without-example retained generic shared storage: %v\n%s", err, baseline)
	}
	for _, relative := range exampleClientPaths {
		body, readErr := os.ReadFile(filepath.Join(dir, relative))
		if readErr != nil || strings.Contains(string(body), "/v1/examples") || strings.Contains(string(body), "resource_models") {
			t.Fatalf("--without-example retained example client behavior in %s: %v", relative, readErr)
		}
	}
	blankAcceptance, err := os.ReadFile(filepath.Join(dir, "scripts/live-acceptance.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blankAcceptance), "expect_status 503 http://localhost:8080/readyz") || !strings.Contains(string(blankAcceptance), "expect_status 204 http://localhost:8080/livez") {
		t.Fatal("blank live acceptance must prove readiness fails closed while liveness stays independent")
	}
	blankScalar, err := os.ReadFile(filepath.Join(dir, "scripts/scalar-browser-acceptance.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blankScalar), "waitForAuthorizedTryRequest") {
		t.Fatal("blank Scalar acceptance must tolerate only a bounded credential-application delay")
	}
	manifestPath := filepath.Join(dir, ".make-app.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(body), fmt.Sprintf(`"schemaVersion": %d`, templateSchemaVersion), `"schemaVersion": 3`, 1)
	if err := os.WriteFile(manifestPath, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err == nil || !strings.Contains(err.Error(), "docs/upgrading-v3-to-v4.md") {
		t.Fatalf("domain mutation did not provide the version-3 upgrade procedure: %v", err)
	}
}

func TestExampleRemoveEliminatesPublicSliceWithForwardMigration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "removable")
	if err := run([]string{"new", "Removable", "--module", "example.com/removable", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"example", "remove", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/domain/example")); !os.IsNotExist(err) {
		t.Fatalf("example source remains after removal: %v", err)
	}
	registry, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/generated/domains.go"))
	if err != nil || strings.Contains(string(registry), `"example"`) {
		t.Fatalf("example route registry remains: %v\n%s", err, registry)
	}
	up, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000016_remove_example_resources.up.sql"))
	if err != nil || !strings.Contains(string(up), "DROP TABLE resource_models") {
		t.Fatalf("example removal lacks forward migration: %v\n%s", err, up)
	}
	if err := run([]string{"example", "remove", "--dir", dir}); err == nil {
		t.Fatal("second example removal unexpectedly succeeded")
	}
}

func TestExampleRemoveRefusesModifiedDependentClient(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "modified")
	if err := run([]string{"new", "Modified", "--module", "example.com/modified", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "apps/web/src/routes/+page.svelte")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, []byte("\n<!-- user behavior -->\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"example", "remove", "--dir", dir}); err == nil || !strings.Contains(err.Error(), "was modified") {
		t.Fatalf("modified dependent client was not protected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/domain/example")); err != nil {
		t.Fatalf("refusal partially removed example: %v", err)
	}
}

func TestExampleRemoveRefusesUserSourceDependency(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dependent")
	if err := run([]string{"new", "Dependent", "--module", "example.com/dependent", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "apps/api/internal/app/example_consumer.go")
	body := []byte("package app\n\nimport _ \"example.com/dependent/apps/api/internal/domain/example\"\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"example", "remove", "--dir", dir}); err == nil || !strings.Contains(err.Error(), "depends on the example") {
		t.Fatalf("user dependency did not block example removal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/domain/example")); err != nil {
		t.Fatalf("dependency refusal partially removed example: %v", err)
	}
}

func TestDomainAddRestoresContractsAndCanRetryAfterGenerationFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "contract-rollback")
	if err := run([]string{"new", "Contract Rollback", "--module", "example.com/contract-rollback", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	openAPIPath := filepath.Join(dir, "packages/api-client/openapi.json")
	schemaPath := filepath.Join(dir, "packages/api-client/src/schema.d.ts")
	originalOpenAPI, err := os.ReadFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	originalSchema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(dir, "apps/api/internal/generated/domains.go")
	originalRegistry, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	failingMake := filepath.Join(t.TempDir(), "make")
	script := "#!/bin/sh\nprintf partial > packages/api-client/openapi.json\nprintf partial > packages/api-client/src/schema.d.ts\nexit 1\n"
	if err := os.WriteFile(failingMake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	originalCommandName := commandName
	commandName = func(name string) string {
		if name == "make" {
			return failingMake
		}
		return name
	}
	err = run([]string{"domain", "add", "habit", "--dir", dir})
	commandName = originalCommandName
	if err == nil {
		t.Fatal("domain add unexpectedly survived failed contract generation")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "apps/api/internal/domain/habit")); !os.IsNotExist(statErr) {
		t.Fatalf("failed domain add left installed source: %v", statErr)
	}
	if body, readErr := os.ReadFile(openAPIPath); readErr != nil || string(body) != string(originalOpenAPI) {
		t.Fatalf("failed domain add did not restore OpenAPI: %v", readErr)
	}
	if body, readErr := os.ReadFile(schemaPath); readErr != nil || string(body) != string(originalSchema) {
		t.Fatalf("failed domain add did not restore schema: %v", readErr)
	}
	if body, readErr := os.ReadFile(registryPath); readErr != nil || string(body) != string(originalRegistry) {
		t.Fatalf("failed domain add did not restore the DI registry: %v", readErr)
	}
	if err := os.RemoveAll(filepath.Join(dir, "node_modules")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err != nil {
		t.Fatalf("identical retry failed after rollback: %v", err)
	}
}

func TestDomainPluralAndFieldValidation(t *testing.T) {
	if pluralize("category") != "categories" || pluralize("status") != "statuses" || pluralize("journal") != "journals" {
		t.Fatal("common English pluralization is not stable")
	}
	for _, valid := range []string{"name:string", "active:bool,count:int,score:float,due_at:time", "select:string,order:int,references:bool"} {
		if err := validateFieldSpec(valid); err != nil {
			t.Fatalf("valid fields rejected: %s: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "Name:string", "name:uuid", "name:string,name:bool", "id:string", "attributes:string", "table_name:string", "due_at:time,due__at:time"} {
		if err := validateFieldSpec(invalid); err == nil {
			t.Fatalf("invalid fields accepted: %q", invalid)
		}
	}
	if err := validateFieldSpec(strings.Repeat("a", 41) + ":string"); err == nil {
		t.Fatal("overlong field accepted")
	}
	fields := make([]string, 26)
	for index := range fields {
		fields[index] = fmt.Sprintf("field_%d:string", index)
	}
	if err := validateFieldSpec(strings.Join(fields, ",")); err == nil {
		t.Fatal("structurally unsafe field count accepted")
	}
}

func TestDomainAddRejectsReservedAndDuplicateRESTCollections(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "route-collisions")
	if err := run([]string{"new", "Route Collisions", "--module", "example.com/route-collisions", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "profile", "--plural", "me", "--dir", dir}); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("reserved /v1/me collection was accepted: %v", err)
	}
	if err := run([]string{"domain", "add", "token", "--plural", "session", "--dir", dir}); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("reserved /v1/session collection was accepted: %v", err)
	}
	if err := run([]string{"domain", "add", "type", "--dir", dir}); err == nil {
		t.Fatal("Go keyword domain was accepted")
	}
	if err := run([]string{"domain", "add", strings.Repeat("a", 41), "--dir", dir}); err == nil {
		t.Fatal("PostgreSQL-unsafe long domain was accepted")
	}
	if err := run([]string{"domain", "add", "journal", "--plural", "entries", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "log", "--plural", "entries", "--dir", dir}); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("duplicate REST collection was accepted: %v", err)
	}
}

func TestDoctorVersionPolicyRejectsUnsupportedToolchains(t *testing.T) {
	if generatorVersion() == "" {
		t.Fatal("generator version must always be reportable")
	}
	valid := map[string]string{"go": "go version go1.25.12 linux/amd64", "node": "v22.17.0", "pnpm": "11.0.7", "python3": "Python 3.12.1", "git": "git version 2.50.0", "docker": "Docker Compose version v2.39.1"}
	for tool, output := range valid {
		if !validToolVersion(tool, output) {
			t.Fatalf("supported %s version rejected: %s", tool, output)
		}
	}
	invalid := map[string]string{"go": "go version go1.24.9", "node": "v20.19.0", "pnpm": "10.9.0", "python3": "Python 3.9.9", "git": "git version 2.39.0", "docker": "Docker Compose version v2.19.0"}
	for tool, output := range invalid {
		if validToolVersion(tool, output) {
			t.Fatalf("unsupported %s version accepted: %s", tool, output)
		}
	}
}

func TestDoctorChecksNativeBuildPrerequisites(t *testing.T) {
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, evidence := range []string{"Ruby 3.2.9", "Bundler 2.6.9", "androidSDKToolAvailable", "adb or sdkmanager"} {
		if !strings.Contains(string(body), evidence) {
			t.Errorf("doctor omits native prerequisite %q", evidence)
		}
	}
}

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
	mobilePackage, err := os.ReadFile(filepath.Join(dir, "apps/mobile/package.json"))
	if err != nil || !strings.Contains(string(mobilePackage), `"expo-crypto":"55.0.17"`) {
		t.Fatalf("mobile idempotency dependency is not explicitly SDK-pinned: %v\n%s", err, mobilePackage)
	}
	if !strings.Contains(string(mobilePackage), `expo export --platform ios`) || !strings.Contains(string(mobilePackage), `expo export --platform android`) || strings.Contains(string(mobilePackage), `--platform all`) {
		t.Fatalf("mobile production build does not explicitly export both native targets: %s", mobilePackage)
	}
	mobileConfig, err := os.ReadFile(filepath.Join(dir, "apps/mobile/app.json"))
	if err != nil || !strings.Contains(string(mobileConfig), `"scheme": "habitkit"`) || !strings.Contains(string(mobileConfig), `"package": "com.example.habitkit"`) {
		t.Fatalf("mobile native identifiers are not platform-safe: %v\n%s", err, mobileConfig)
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"apps/api/internal/domain/habit/entity.go",
		"apps/api/internal/app/habit/ports.go",
		"apps/api/internal/app/habit/service.go",
		"apps/api/internal/app/habit/service_test.go",
		"apps/api/internal/adapters/gormstore/habit/repository.go",
		"apps/api/internal/adapters/httpserver/habit/dto/habit.go",
		"apps/api/internal/adapters/httpserver/habit/mapper/habit.go",
		"apps/api/internal/adapters/httpserver/habit/routes/routes.go",
		"apps/api/internal/generated/habit_wiring_test.go",
		"apps/api/internal/adapters/dbmigrations/000016_create_habits.up.sql",
		"apps/api/internal/adapters/dbmigrations/000016_create_habits.down.sql",
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
	for _, wiring := range []string{
		`habitstore.New(dependencies.DB)`,
		`habitapp.New(habitapp.Dependencies{`,
		`habitroutes.Register(api, service)`,
	} {
		if !strings.Contains(string(registry), wiring) {
			t.Fatalf("added domain DI registry is missing %q:\n%s", wiring, registry)
		}
	}
	service, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/app/habit/service.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dependency := range []string{"ports.Authenticator", "ports.Authorizer", "ports.AuthorizationOutbox", "ports.AuthorizationSerializer", "Repository", "ports.Audits", "ports.Clock", "ports.Probe", "NewID", "AuthorizationWorker", "AuthorizationLease", "CursorSigningKey"} {
		if !strings.Contains(string(service), dependency) {
			t.Fatalf("generated domain service dependency bundle is missing %q", dependency)
		}
	}
	if !strings.Contains(string(service), "ports.ErrAuthorizationPolicyNotConfigured") || !strings.Contains(string(service), "Authenticate(ctx, authorization)") {
		t.Fatal("generated domain service does not authenticate and fail closed before policy implementation")
	}
	if strings.Contains(string(service), "Authenticate(ctx, authorization); err != nil {\n\t\treturn ports.ErrInvalidCredential") {
		t.Fatal("generated domain service reclassifies authentication dependency failures as invalid credentials")
	}
	server, err := os.ReadFile(filepath.Join(dir, "apps/api/cmd/server/main.go"))
	if err != nil || !strings.Contains(string(server), "DomainRegistrations: generated.Registrations(generated.Dependencies{") {
		t.Fatalf("runtime composition does not inject generated domains: %v\n%s", err, server)
	}
	for _, wiring := range []string{"AuthorizationOutbox: store", "AuthorizationSerializer: store"} {
		if !strings.Contains(string(server), wiring) {
			t.Fatalf("runtime composition is missing %s", wiring)
		}
	}
	openAPI, err := os.ReadFile(filepath.Join(dir, "apps/api/cmd/openapi/main.go"))
	if err != nil || !strings.Contains(string(openAPI), "DomainRegistrations: generated.Registrations(generated.Dependencies{})") {
		t.Fatalf("OpenAPI composition does not register generated domains: %v\n%s", err, openAPI)
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
	routes, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/httpserver/habit/routes/routes.go"))
	if err != nil || !strings.Contains(string(routes), "Method: http.MethodPut") || strings.Contains(string(routes), "Method: http.MethodPatch") {
		t.Fatalf("generated update route does not expose explicit full replacement: %v\n%s", err, routes)
	}
	dto, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/httpserver/habit/dto/habit.go"))
	if err != nil || !strings.Contains(string(dto), `required:"true"`) {
		t.Fatalf("generated replacement DTO permits omitted fields: %v\n%s", err, dto)
	}
	if err := run([]string{"domain", "add", "journal", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000017_create_journals.up.sql")); err != nil {
		t.Fatalf("second domain did not receive the next migration version: %v", err)
	}
	journalDTO, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/httpserver/journal/dto/journal.go"))
	if err != nil || !strings.Contains(string(journalDTO), "type JournalCreate struct") {
		t.Fatalf("domain DTO schemas are not uniquely named for Huma: %v\n%s", err, journalDTO)
	}
	if err := run([]string{"domain", "add", "foo_1", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "foo1", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	separatedDTO, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/httpserver/foo_1/dto/foo_1.go"))
	if err != nil || !strings.Contains(string(separatedDTO), "type Foo_1Create struct") {
		t.Fatalf("separator-preserving domain schema name is missing: %v\n%s", err, separatedDTO)
	}
	compactDTO, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/httpserver/foo1/dto/foo1.go"))
	if err != nil || !strings.Contains(string(compactDTO), "type Foo1Create struct") {
		t.Fatalf("compact domain schema name is missing: %v\n%s", err, compactDTO)
	}
	openAPICommand := exec.Command("go", "run", "./cmd/openapi")
	openAPICommand.Dir = filepath.Join(dir, "apps/api")
	openAPI, err = openAPICommand.CombinedOutput()
	if err != nil {
		t.Fatalf("collision-prone domains could not register with Huma: %v\n%s", err, openAPI)
	}
	for _, path := range []string{"/v1/foo_1s", "/v1/foo1s"} {
		if !strings.Contains(string(openAPI), path) {
			t.Fatalf("OpenAPI is missing collision-prone route %s", path)
		}
	}
}

func TestNewNormalizesNumericLeadingNativeIdentifiers(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "numeric-native")
	if err := run([]string{"new", "123.App", "--module", "example.com/numeric-native", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "apps/mobile/app.json"))
	if err != nil || !strings.Contains(string(body), `"scheme": "app123app"`) || !strings.Contains(string(body), `"bundleIdentifier": "com.example.app123app"`) {
		t.Fatalf("numeric-leading name produced invalid native identifiers: %v\n%s", err, body)
	}
	mobileSource, err := os.ReadFile(filepath.Join(dir, "apps/mobile/app/index.tsx"))
	if err != nil || !strings.Contains(string(mobileSource), "scheme: 'app123app'") {
		t.Fatalf("mobile runtime redirect does not match registered scheme: %v\n%s", err, mobileSource)
	}
	environment, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	if err != nil || !strings.Contains(string(environment), "APP_123_APP_HTTP_ADDR=") {
		t.Fatalf("numeric-leading name produced an invalid shell environment prefix: %v\n%s", err, environment)
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
	broken := strings.Replace(string(original), "const LatestVersion uint = 15", "const MissingVersion uint = 15", 1)
	if err := os.WriteFile(migrationsPath, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"domain", "add", "habit", "--dir", dir}); err == nil {
		t.Fatal("domain add unexpectedly accepted missing migration metadata")
	}
	for _, path := range []string{
		"apps/api/internal/domain/habit",
		"apps/api/internal/adapters/gormstore/habit",
		"apps/api/internal/adapters/dbmigrations/000016_create_habits.up.sql",
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
	if strings.Contains(string(migration), "invitation.created") {
		t.Error("historical audit migration was rewritten with later invitation actions")
	}
	invitationMigration, err := os.ReadFile(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/000011_invitations.up.sql"))
	if err != nil || !strings.Contains(string(invitationMigration), "invitation.created") || !strings.Contains(string(invitationMigration), "DROP CONSTRAINT audit_event_models_action_check") {
		t.Fatalf("invitation migration does not evolve the audit constraint forward: %v\n%s", err, invitationMigration)
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
	blankDir := filepath.Join(t.TempDir(), "localized-blank")
	if err := run([]string{"new", "Localized Blank", "--module", "example.com/localized-blank", "--output", blankDir, "--without-example"}); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("node", "scripts/check-i18n.mjs")
	check.Dir = blankDir
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("clean blank generated project failed i18n gate on non-visible protocol syntax: %v\n%s", err, output)
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
		"apps/web/src/routes/bypass.svelte":      `<script>const buttonLabel = 'Delete account';</script><button>{buttonLabel}</button>`,
		"apps/web/src/lib/copy.ts":               `export const dialogTitle = 'Delete account';`,
		"apps/web/src/lib/warning.ts":            `export const warning = 'Delete account';`,
		"apps/mobile/components/expression.tsx":  `export const Copy = () => <Text>{'Delete account'}</Text>;`,
		"apps/mobile/src/forms/attribute.jsx":    `export const Copy = () => <Input placeholder={'Email address'} />;`,
		"apps/mobile/src/forms/ternary.tsx":      `export const Copy = ({ danger }) => <Text>{danger ? 'Delete account' : 'Keep account'}</Text>;`,
		"apps/mobile/src/forms/ternary-last.tsx": `export const Copy = ({ danger }) => <Text>{danger ? 'ok' : 'Delete account'}</Text>;`,
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
	for _, maintained := range []string{
		"actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0",   // v7.0.0, Node 24
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",   // v7.0.0, Node 24
		"actions/setup-node@820762786026740c76f36085b0efc47a31fe5020", // v7.0.0, Node 24
	} {
		if !strings.Contains(workflow, maintained) {
			t.Errorf("generated CI must pin maintained Node 24 action %q", maintained)
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
	webDockerfile, err := os.ReadFile(filepath.Join(dir, "apps/web/Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"COPY --chown=65532:0 packages/client-core/package.json packages/client-core/package.json", "COPY --chown=65532:0 packages/client-core packages/client-core"} {
		if !strings.Contains(string(webDockerfile), required) {
			t.Errorf("web production image omits imported workspace package: %s", required)
		}
	}
	releaseWorkflow, err := os.ReadFile(filepath.Join(dir, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(releaseWorkflow), `test "$(git rev-parse origin/main)" = "$SOURCE_SHA"`) < 3 {
		t.Fatal("release workflow must reverify current main before candidate publication, digest promotion, and Git tagging")
	}
	generatedRelease := string(releaseWorkflow)
	if !strings.Contains(generatedRelease, "actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0") {
		t.Fatal("generated release workflow must pin maintained Node 24 checkout")
	}
	if strings.Count(generatedRelease, "actions/attest@f7c74d28b9d84cb8768d0b8ca14a4bac6ef463e6") != 4 {
		t.Fatal("generated release workflow must use the pinned unified attestation action for SBOM and provenance")
	}
	if strings.Count(generatedRelease, "create-storage-record: false") != 4 {
		t.Fatal("generated release attestations must remain compatible with personal repositories")
	}
	if strings.Contains(generatedRelease, "actions/attest-sbom@") || strings.Contains(generatedRelease, "actions/attest-build-provenance@") {
		t.Fatal("generated release workflow retained a deprecated attestation wrapper")
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
	bootstrapPrerequisites := strings.Index(string(makefile), "command -v node")
	bootstrapInstall := strings.Index(string(makefile), "pnpm install --frozen-lockfile")
	bootstrapGenerate := strings.Index(string(makefile), "$(MAKE) generate")
	if bootstrapPrerequisites < 0 || bootstrapInstall < bootstrapPrerequisites || bootstrapGenerate < bootstrapPrerequisites {
		t.Fatal("generated bootstrap must verify Node before Node-dependent installation and contract generation")
	}
	if strings.Contains(string(makefile), "govulncheck@") {
		t.Error("security gate must use the reviewed tools module rather than an ad hoc tool download")
	}
	apiDockerfile, err := os.ReadFile(filepath.Join(dir, "apps/api/Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(apiDockerfile), "RUN CGO_ENABLED=0 go build") != 1 || !strings.Contains(string(apiDockerfile), "-o /out/ ./cmd/server ./cmd/schema ./cmd/migrate") {
		t.Error("API image must compile command binaries in one bounded build layer")
	}
	for _, hardened := range []string{
		"registry.access.redhat.com/hi/go:1.25.12-builder-1784449036@sha256:2c7403868c5bae2b988136003a228b0e1be01d1696dca579a8fef3586cbd10e4",
		"registry.access.redhat.com/hi/static:latest-1782367629@sha256:4c1752f4eabb162d15e26f84909725a3ef516f5753211fcb6ad7d76e6fd519e5",
	} {
		if !strings.Contains(string(apiDockerfile), hardened) {
			t.Errorf("API Dockerfile must use reviewed Red Hat Hardened Image %q", hardened)
		}
	}
	for _, hardened := range []string{
		"registry.access.redhat.com/hi/nodejs:24.18.0-builder-1784114937@sha256:0493d282a3e210a3f95d98326f3e2e2c6b151b13830a9fbdb03ff52d1f60354e",
		"registry.access.redhat.com/hi/nodejs:24.18.0-1784114937@sha256:c0fd66cf088af1c6f122bcdbbf5d701ffea303fe84ebb23575ac75edde625113",
	} {
		if !strings.Contains(string(webDockerfile), hardened) {
			t.Errorf("web Dockerfile must use reviewed Red Hat Hardened Image %q", hardened)
		}
	}
	if strings.Contains(string(webDockerfile), "corepack") {
		t.Error("hardened Node build must not depend on unavailable Corepack")
	}
	if !strings.Contains(string(webDockerfile), "deploy --legacy --prod /app/prod") || !strings.Contains(string(webDockerfile), "COPY --from=build --chown=65532:0 /app/prod ./") {
		t.Error("non-root hardened Node build must deploy through its writable application directory")
	}
	compose, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "registry.access.redhat.com/hi/postgresql:18.4-1784429813@sha256:9e685e4bc9838b5ace7a673b5b85ffcada7f851b52c07514d7dbeb8d3f9434f1") {
		t.Error("Compose must use the reviewed Red Hat Hardened PostgreSQL image")
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "Use Red Hat Hardened Images wherever") || !strings.Contains(string(agents), "does not establish FIPS compliance") {
		t.Error("generated engineering guidance must preserve the Hardened Images and FIPS boundary")
	}

	hook, err := os.ReadFile(filepath.Join(dir, ".git/hooks/pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hook), "make pre-commit") {
		t.Error("pre-commit hook must use the generated change-aware gate")
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
	if strings.Contains(string(compose), "network_mode: host") || !strings.Contains(string(compose), "required: false") || !strings.Contains(string(compose), "OIDC_BACKCHANNEL_URL: http://dex:5556/dex") {
		t.Fatal("generated Compose must use portable bridge networking, optional .env loading, and an OIDC backchannel")
	}
	if !strings.Contains(string(compose), `test "$(cat /proc/1/comm)" = postgres && pg_isready`) {
		t.Fatal("generated PostgreSQL health check must reject the temporary initialization server")
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
	if !strings.Contains(string(scalarAcceptance), "waitForAuthorizedTryRequest") {
		t.Fatal("Scalar browser acceptance must tolerate only a bounded credential-application delay")
	}
	for _, retryEvidence := range []string{"errors.TimeoutError", "responseTimeoutMilliseconds", "if (!response)"} {
		if !strings.Contains(string(scalarAcceptance), retryEvidence) {
			t.Errorf("Scalar browser acceptance does not retry missing Try-It responses: %s", retryEvidence)
		}
	}
	scalarSource := string(scalarAcceptance)
	responseWait := strings.Index(scalarSource, "const responsePromise = page.waitForResponse")
	timeoutCatch, sendRequest := -1, -1
	if responseWait >= 0 {
		timeoutCatch = strings.Index(scalarSource[responseWait:], ").catch((error) => {")
		sendRequest = strings.Index(scalarSource[responseWait:], "await page.getByRole('button', { name: /Send Request/ }).click()")
	}
	if responseWait < 0 || timeoutCatch < 0 || sendRequest < 0 || timeoutCatch > sendRequest {
		t.Error("Scalar timeout handler must attach before clicking Send Request so rejection cannot become unhandled")
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
	if strings.Count(string(liveAcceptance), `owner_token="$(session `) < 1 || !strings.Contains(string(liveAcceptance), "owner_exchange_body") || !strings.Contains(string(liveAcceptance), "expect_status 401 -X POST") {
		t.Fatal("live acceptance must exchange OIDC credentials once, reject replay, and exchange again only with a new token after restart")
	}
	if !strings.Contains(string(liveAcceptance), `MAKE_APP_UID="${MAKE_APP_UID:-$(id -u)}"`) || !strings.Contains(string(liveAcceptance), `MAKE_APP_GID="${MAKE_APP_GID:-$(id -g)}"`) {
		t.Fatal("live acceptance must derive the Compose user from the host user")
	}
	if !strings.Contains(string(liveAcceptance), `COMPOSE_PROJECT_NAME="make-app-acceptance-`) || strings.Count(string(liveAcceptance), "docker compose down --volumes --remove-orphans") < 2 {
		t.Fatal("live acceptance must isolate each Compose project and clean it before and after execution")
	}
	if !strings.Contains(string(liveAcceptance), "docker compose down --volumes --remove-orphans --rmi local") {
		t.Fatal("live acceptance cleanup must remove locally built project images")
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
	if !strings.Contains(string(liveAcceptance), "expect_status 503 http://localhost:8080/readyz") || !strings.Contains(string(liveAcceptance), "expect_status 204 http://localhost:8080/livez") {
		t.Fatal("example live acceptance must prove readiness fails closed while liveness stays independent")
	}
	if !strings.Contains(string(liveAcceptance), "restart_audit_cursor") || !strings.Contains(string(liveAcceptance), "restart audit event was not found after paginated traversal") {
		t.Fatal("restart acceptance must traverse cursor-paginated audit history instead of assuming the event is on the default page")
	}
	if !strings.Contains(string(liveAcceptance), "--user 1001:1001") || !strings.Contains(string(liveAcceptance), "NPM_CONFIG_PREFIX=/tmp/make-app-npm-global") {
		t.Fatal("live acceptance must install pinned pnpm with the hardened Node builder under a non-1000 numeric user")
	}
	makefile, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(makefile), `MAKE_APP_UID=$$(id -u) MAKE_APP_GID=$$(id -g) docker compose up --build`) {
		t.Fatal("the supported Compose entrypoint must work for non-1000 host users")
	}
	for _, workflow := range []string{"dev:", "mobile:", "db-shell:", "reset:", "pre-commit:", "pre-push:"} {
		if !strings.Contains(string(makefile), workflow) {
			t.Fatalf("generated developer workflow is missing %s", workflow)
		}
	}
	for _, path := range []string{"docs/development.md", "docs/domains.md", "docs/mobile.md", "docs/oidc.md", "docs/production.md", "scripts/dev.sh", "scripts/watch-go.sh", "scripts/seed.sh"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("generated onboarding asset is missing %s: %v", path, err)
		}
	}
}

func TestGeneratedWebProductionBuildOutputIsIgnored(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "clean-web-build")
	if err := run([]string{"new", "Clean Web Build", "--module", "example.com/clean-web-build", "--output", dir, "--without-example"}); err != nil {
		t.Fatal(err)
	}

	check := exec.Command("git", "check-ignore", "apps/web/build/index.js")
	check.Dir = dir
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("generated web production build output is not ignored: %v\n%s", err, output)
	}
}

func TestGeneratedProjectScopedCodeCriticExists(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "project-agent")
	if err := run([]string{"new", "Project Agent", "--module", "example.com/project-agent", "--output", dir, "--without-example"}); err != nil {
		t.Fatal(err)
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "code-critic") {
		t.Fatal("generated engineering guidance does not reference the code critic")
	}
	critic, err := os.ReadFile(filepath.Join(dir, ".codex/agents/code-critic.toml"))
	if err != nil {
		t.Fatalf("generated project-scoped code critic is missing: %v", err)
	}
	for _, required := range []string{`name = "code-critic"`, `sandbox_mode = "read-only"`} {
		if !strings.Contains(string(critic), required) {
			t.Errorf("generated code critic is missing %q", required)
		}
	}
}

func TestGeneratorReleaseWorkflowDoesNotAssumeGeneratedWorkspace(t *testing.T) {
	body, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(body)
	for _, maintained := range []string{
		"actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0", // v7.0.0, Node 24
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", // v7.0.0, Node 24
	} {
		if !strings.Contains(workflow, maintained) {
			t.Errorf("generator release workflow must pin maintained Node 24 action %q", maintained)
		}
	}
	if strings.Contains(workflow, "pnpm/action-setup") || strings.Contains(workflow, "cache: pnpm") {
		t.Fatal("Go-only generator release workflow assumes a root JavaScript lockfile")
	}
	if !strings.Contains(workflow, "workflow_run:") || !strings.Contains(workflow, "Require successful CI evidence") {
		t.Fatal("generator release workflow must consume exact successful CI evidence without rerunning the suite")
	}
	for _, provenance := range []string{"github.event.workflow_run.event == 'push'", "github.event.workflow_run.head_repository.full_name == github.repository", "sha: ${{ steps.source.outputs.sha }}", "needs.plan.outputs.sha"} {
		if !strings.Contains(workflow, provenance) {
			t.Fatalf("generator release workflow is missing trusted exact-SHA provenance check %q", provenance)
		}
	}
	if strings.Contains(workflow, `git tag "${{ needs.plan.outputs.tag }}" "$GITHUB_SHA"`) {
		t.Fatal("generator release workflow publishes the workflow event SHA instead of the tested source SHA")
	}
}

func TestCIOnlyPushesMainWhilePullRequestsRemainEnabled(t *testing.T) {
	for _, path := range []string{".github/workflows/ci.yml", "template/base/DOTgithub/workflows/ci.yml"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		workflow := string(body)
		if !strings.Contains(workflow, "push:\n    branches: [main]\n") || !strings.Contains(workflow, "pull_request:\n") {
			t.Errorf("%s must avoid duplicate feature-push CI while retaining PR CI", path)
		}
	}
}

func TestGeneratorReleaseDocumentationCannotDrift(t *testing.T) {
	for _, path := range []string{".release-version", "scripts/check-release-docs.sh", "scripts/update-release-docs.sh", "scripts/release-docs_test.sh"} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("release documentation control is missing %s: %v", path, err)
		}
	}
	command := exec.Command("scripts/check-release-docs.sh")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("release documentation is stale: %v\n%s", err, output)
	}
	makefile, _ := os.ReadFile("Makefile")
	for _, gate := range []string{"scripts/check-release-docs.sh", "scripts/release-docs_test.sh"} {
		if !strings.Contains(string(makefile), gate) {
			t.Errorf("make verify does not enforce %s", gate)
		}
	}
	tests, _ := os.ReadFile("scripts/release-docs_test.sh")
	for _, evidence := range []string{"stale README", "stale compatibility", "v9.8.7", "@latest"} {
		if !strings.Contains(string(tests), evidence) {
			t.Errorf("release documentation regression test lacks %q", evidence)
		}
	}
	workflow, _ := os.ReadFile(".github/workflows/release.yml")
	for _, evidence := range []string{"scripts/update-release-docs.sh", "docs: synchronize release documentation", "git push origin HEAD:main"} {
		if !strings.Contains(string(workflow), evidence) {
			t.Errorf("release workflow does not automatically synchronize documentation: missing %q", evidence)
		}
	}
}

func TestInitAdoptsExistingSpecFirstRepositoryWithoutReplacingHistory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(filepath.Join(dir, "specs", "habits"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Existing guidance\n\nKeep this rule.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hour Paths\n\nExisting introduction.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("product-local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "specs", "habits", "habits.spec.md"), []byte("# Existing habit specification\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git := exec.Command("git", "init", "-b", "main")
	git.Dir = dir
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	git = exec.Command("git", "add", ".")
	git.Dir = dir
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	git = exec.Command("git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "docs: specify application")
	git.Dir = dir
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}
	before := exec.Command("git", "rev-parse", "HEAD")
	before.Dir = dir
	beforeSHA, err := before.Output()
	if err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"init", "Hour Paths", "--module", "example.com/hour-paths", "--dir", dir, "--without-example"}); err != nil {
		t.Fatal(err)
	}
	after := exec.Command("git", "rev-parse", "HEAD")
	after.Dir = dir
	afterSHA, err := after.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeSHA) != string(afterSHA) {
		t.Fatal("init replaced existing Git history")
	}
	for path, want := range map[string]string{
		"AGENTS.md": "Keep this rule.", "README.md": "Existing introduction.",
		".gitignore": "product-local", "specs/habits/habits.spec.md": "Existing habit specification",
	} {
		body, readErr := os.ReadFile(filepath.Join(dir, path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("%s did not preserve existing content %q", path, want)
		}
	}
	agents, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(agents), "BEGIN MAKE-APP BASELINE GUIDANCE") || !strings.Contains(string(agents), "Internationalization is mandatory") {
		t.Fatal("init did not merge generated engineering guidance")
	}
	if _, err := os.Stat(filepath.Join(dir, ".make-app.json")); err != nil {
		t.Fatal("init did not install generated application manifest")
	}
	if _, err := os.Stat(filepath.Join(dir, "apps", "api", "go.mod")); err != nil {
		t.Fatal("init did not install application")
	}
}

func TestInitRefusesConflictsAndUnsupportedExistingFilesWithoutMutation(t *testing.T) {
	for name, setup := range map[string]func(string) error{
		"unsupported source": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "main.go"), []byte("package existing\n"), 0o644)
		},
		"spec collision": func(dir string) error {
			path := filepath.Join(dir, "specs", "platform", "architecture.spec.md")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(path, []byte("# Product-specific architecture\n"), 0o644)
		},
		"hook collision": func(dir string) error {
			path := filepath.Join(dir, ".git", "hooks", "pre-commit")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(path, []byte("#!/bin/sh\nexisting-hook\n"), 0o755)
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "existing")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			git := exec.Command("git", "init", "-b", "main")
			git.Dir = dir
			if output, err := git.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v\n%s", err, output)
			}
			if err := setup(dir); err != nil {
				t.Fatal(err)
			}
			before := snapshotTree(t, dir)
			if err := run([]string{"init", "Conflict", "--module", "example.com/conflict", "--dir", dir}); err == nil {
				t.Fatal("init accepted unsafe existing content")
			}
			after := snapshotTree(t, dir)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("failed init mutated existing repository")
			}
		})
	}
}

func TestInitRejectsSymlinksWithoutWritingOutsideRepository(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git := exec.Command("git", "init", "-b", "main")
	git.Dir = dir
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	external := filepath.Join(t.TempDir(), "outside.md")
	want := []byte("must remain unchanged\n")
	if err := os.WriteFile(external, want, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"init", "Safe", "--module", "example.com/safe", "--dir", dir}); err == nil {
		t.Fatal("init accepted a symlink in adoptable content")
	}
	got, err := os.ReadFile(external)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("failed init wrote through repository symlink")
	}
}

func TestInitSupportsLinkedWorktreeAndConfiguredHooksPath(t *testing.T) {
	base := filepath.Join(t.TempDir(), "base")
	worktree := filepath.Join(t.TempDir(), "worktree")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.name", "Test"}, {"config", "user.email", "test@example.com"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = base
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "AGENTS.md"), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "docs: specify"}, {"worktree", "add", "-b", "adopt", worktree}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = base
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	config := exec.Command("git", "config", "core.hooksPath", ".hooks")
	config.Dir = worktree
	if output, err := config.CombinedOutput(); err != nil {
		t.Fatalf("git config: %v\n%s", err, output)
	}
	if err := run([]string{"init", "Worktree", "--module", "example.com/worktree", "--dir", worktree, "--without-example"}); err != nil {
		t.Fatal(err)
	}
	for _, hook := range []string{"pre-commit", "pre-push"} {
		info, err := os.Stat(filepath.Join(worktree, ".hooks", hook))
		if err != nil || info.Mode()&0o111 == 0 {
			t.Fatalf("configured hook %s not installed: %v", hook, err)
		}
	}
}

func TestInitRejectsExternalConfiguredHooksPath(t *testing.T) {
	dir := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Existing guidance\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git := exec.Command("git", "init", "-b", "main")
	git.Dir = dir
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	config := exec.Command("git", "config", "core.hooksPath", external)
	config.Dir = dir
	if output, err := config.CombinedOutput(); err != nil {
		t.Fatalf("git config: %v: %s", err, output)
	}
	if err := run([]string{"init", "External Hooks", "--dir", dir, "--module", "example.com/external-hooks"}); err == nil || !strings.Contains(err.Error(), "outside the repository") {
		t.Fatalf("expected external hooks path refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(external, "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("external hook was written: %v", err)
	}
}

func TestGeneratedNativeMobileAndPublicProjectBaseline(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "native")
	if err := run([]string{"new", "Native Ready", "--module", "example.com/native-ready", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"LICENSE", "SECURITY.md", "docs/compatibility.md", ".github/ISSUE_TEMPLATE/bug.yml",
		".github/dependabot.yml", "apps/mobile/eas.json", "apps/mobile/assets/icon.png",
		"apps/mobile/assets/adaptive-icon.png", "apps/mobile/assets/splash-icon.png",
		"packages/client-core/package.json", "packages/client-core/src/index.ts", "packages/client-core/src/index.test.ts",
		"tools/eas-cli/package.json",
		"apps/mobile/Gemfile", "apps/mobile/Gemfile.lock", "scripts/run-eas.mjs", "scripts/run-eas.test.mjs",
		"scripts/validate-mobile-config.mjs", "scripts/validate-expo-native-set.mjs", "scripts/check-ruby-vulnerabilities.py",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("generated baseline missing %s: %v", path, err)
		}
	}
	license, _ := os.ReadFile(filepath.Join(dir, "LICENSE"))
	if !strings.Contains(string(license), "Apache License") || !strings.Contains(string(license), "Version 2.0") {
		t.Fatal("generated project is not Apache-2.0 licensed")
	}
	mobilePackage, _ := os.ReadFile(filepath.Join(dir, "apps/mobile/package.json"))
	for _, required := range []string{"expo-dev-client", "expo-doctor", "mobile:export", "mobile:prebuild", "mobile:build:android", "mobile:build:ios", "validate-mobile-release-env.mjs"} {
		if !strings.Contains(string(mobilePackage), required) {
			t.Errorf("mobile package missing %q", required)
		}
	}
	if strings.Contains(string(mobilePackage), "pnpm dlx") || strings.Contains(string(mobilePackage), "pnpm --dir ../../tools/eas-cli") || !strings.Contains(string(mobilePackage), "run-eas.mjs") {
		t.Fatal("mobile release command dynamically resolves EAS CLI")
	}
	config, _ := os.ReadFile(filepath.Join(dir, "apps/mobile/app.json"))
	for _, required := range []string{`"version"`, `"buildNumber"`, `"versionCode"`, `"runtimeVersion"`, `"updates"`, `"icon"`, `"adaptiveIcon"`, `"splash"`} {
		if !strings.Contains(string(config), required) {
			t.Errorf("mobile config missing %s", required)
		}
	}
	eas, _ := os.ReadFile(filepath.Join(dir, "apps/mobile/eas.json"))
	for _, required := range []string{`"environment": "production"`, `"NATIVE_READY_APP_ENV": "production"`, "ubuntu-24.04-jdk-17-ndk-r27b-sdk-55", "macos-sequoia-15.6-xcode-26.2", `"cocoapods": "1.16.2"`} {
		if !strings.Contains(string(eas), required) {
			t.Errorf("production EAS profile missing %s", required)
		}
	}
	mobileSource, _ := os.ReadFile(filepath.Join(dir, "apps/mobile/app/index.tsx"))
	for _, required := range []string{"loadMobileConfig", "Constants.expoConfig?.extra", "refreshSessionCredential", "isValidSessionCredential", "sessionRetryDelay", "decision.retryable", "handleSessionFailure(cause, next"} {
		if !strings.Contains(string(mobileSource), required) {
			t.Errorf("mobile session/release adapter missing %q", required)
		}
	}
	workflow, _ := os.ReadFile(filepath.Join(dir, ".github/workflows/ci.yml"))
	for _, required := range []string{"ubuntu-24.04", "macos-26", "17.0.15+6", "ruby/setup-ruby@003a5c4d8d6321bd302e38f6f0ec593f77f06600", "expo-doctor", "assembleDebug", "xcodebuild"} {
		if !strings.Contains(string(workflow), required) {
			t.Errorf("native workflow missing %q", required)
		}
	}
}

func TestClientSessionStateMachinePreservesValidCredentialForTransientFailures(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "session")
	if err := run([]string{"new", "Session State", "--module", "example.com/session-state", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "packages/client-core/src/index.test.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, evidence := range []string{"network", "429", "503", "401", "expired", "local_session_unreadable", "authenticated_offline", "atomically persist", "sessionRetryDelay"} {
		if !strings.Contains(string(body), evidence) {
			t.Errorf("session tests lack %q evidence", evidence)
		}
	}
	mobile, _ := os.ReadFile(filepath.Join(dir, "apps/mobile/app/index.tsx"))
	if !strings.Contains(string(mobile), "classifySessionFailure") || !strings.Contains(string(mobile), "authenticated_offline") {
		t.Fatal("mobile does not consume shared session classifier")
	}
	restoreSource, err := os.ReadFile(filepath.Join(dir, "apps/mobile/src/session-restoration.ts"))
	if err != nil {
		t.Fatal("mobile session restoration use case is missing")
	}
	restoreTest, err := os.ReadFile(filepath.Join(dir, "apps/mobile/src/session-restoration.test.ts"))
	if err != nil {
		t.Fatal("mobile cold-launch restoration test is missing")
	}
	for _, evidence := range []string{"OIDC discovery unavailable", "API unavailable", "authenticated_offline", "stored-token"} {
		if !strings.Contains(string(restoreTest), evidence) {
			t.Errorf("mobile cold-launch restoration test lacks %q evidence", evidence)
		}
	}
	restoreCall := strings.Index(string(mobile), "void restoreStoredSession")
	if restoreCall < 0 {
		t.Fatal("mobile secure-storage restoration is not invoked")
	}
	restoreEffectEnd := strings.Index(string(mobile)[restoreCall:], "}, []);")
	if restoreEffectEnd < 0 || strings.Contains(string(mobile)[restoreCall:restoreCall+restoreEffectEnd], "discovery") || !strings.Contains(string(restoreSource), "restoreStoredSession") {
		t.Fatal("mobile secure-storage restoration must run once without waiting for OIDC discovery")
	}
	webAuth, _ := os.ReadFile(filepath.Join(dir, "apps/web/src/lib/auth.ts"))
	refreshStart := strings.Index(string(webAuth), "export async function refreshApplicationSession")
	refreshEnd := strings.Index(string(webAuth), "export async function revokeApplicationSession")
	if refreshStart < 0 || refreshEnd <= refreshStart {
		t.Fatal("web refresh adapter is missing")
	}
	refreshBody := string(webAuth)[refreshStart:refreshEnd]
	if !strings.Contains(refreshBody, "refreshSessionCredential") || strings.Contains(refreshBody, "clearApplicationSession()") {
		t.Fatal("web refresh adapter bypasses typed shared retention behavior")
	}
	blank := filepath.Join(t.TempDir(), "blank-session")
	if err := run([]string{"new", "Blank Session", "--module", "example.com/blank-session", "--output", blank, "--without-example"}); err != nil {
		t.Fatal(err)
	}
	blankMobile, _ := os.ReadFile(filepath.Join(blank, "apps/mobile/app/index.tsx"))
	blankWeb, _ := os.ReadFile(filepath.Join(blank, "apps/web/src/routes/+page.svelte"))
	for name, source := range map[string][]byte{"blank mobile": blankMobile, "blank web": blankWeb} {
		for _, evidence := range []string{"sessionRetryDelay", "isSessionFailure"} {
			if !strings.Contains(string(source), evidence) {
				t.Errorf("%s omits %s", name, evidence)
			}
		}
	}
}

func TestGeneratedWebProductionConfigurationFailsClosed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "web-production-config")
	if err := run([]string{"new", "Web Production Config", "--module", "example.com/web-production-config", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	webDockerfile, _ := os.ReadFile(filepath.Join(dir, "apps/web/Dockerfile"))
	compose, _ := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if !strings.Contains(string(webDockerfile), "WEB_PRODUCTION_CONFIG_APP_ENV=production") {
		t.Fatal("web production image must default to fail-closed production configuration")
	}
	if !strings.Contains(string(compose), "WEB_PRODUCTION_CONFIG_APP_ENV: development") {
		t.Fatal("local Compose must explicitly select the loopback development contract")
	}
	config, err := os.ReadFile(filepath.Join(dir, "apps/web/src/lib/config.ts"))
	if err != nil {
		t.Fatal("web runtime configuration adapter is missing")
	}
	for _, evidence := range []string{"clientRuntimeConfig", "parseWebConfig"} {
		if !strings.Contains(string(config), evidence) {
			t.Errorf("web runtime configuration lacks %q", evidence)
		}
	}
	for _, path := range []string{"apps/web/src/lib/auth.ts", "apps/web/src/routes/+page.svelte"} {
		source, _ := os.ReadFile(filepath.Join(dir, path))
		if strings.Contains(string(source), "localhost:8080") || strings.Contains(string(source), "localhost:5556") {
			t.Errorf("%s retains an implicit production localhost fallback", path)
		}
	}
	coreTests, _ := os.ReadFile(filepath.Join(dir, "packages/client-core/src/index.test.ts"))
	for _, evidence := range []string{"APP_ENV", "unknown deployment environment", "missing production string configuration"} {
		if !strings.Contains(string(coreTests), evidence) {
			t.Errorf("shared public configuration tests lack %q", evidence)
		}
	}
	liveAcceptance, _ := os.ReadFile(filepath.Join(dir, "scripts/live-acceptance.sh"))
	if !strings.Contains(string(liveAcceptance), "production_config_probe") || !strings.Contains(string(liveAcceptance), "API_URL") {
		t.Error("live acceptance must prove the production web image rejects absent public configuration")
	}
	for _, rejected := range []string{"malformed-environment", "malformed-api-url", "local-api-url", "credentialed-issuer", "non-https-issuer", "blank-client-id"} {
		if !strings.Contains(string(liveAcceptance), rejected) {
			t.Errorf("live acceptance does not exercise rejected production image configuration %q", rejected)
		}
	}
	if !strings.Contains(string(liveAcceptance), "production_config_logs=") || strings.Contains(string(liveAcceptance), `docker logs "$production_config_probe" 2>&1 | grep -q`) {
		t.Error("production configuration assertion can fail nondeterministically with SIGPIPE under pipefail")
	}
}

func TestGeneratedClientsUseCanonicalApplicationConfigurationPrefix(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "canonical-client-config")
	if err := run([]string{"new", "Canonical Client Config", "--module", "example.com/canonical-client-config", "--output", dir}); err != nil {
		t.Fatal(err)
	}
	files := map[string][]string{
		".env.example":                          {"CANONICAL_CLIENT_CONFIG_APP_ENV=development", "CANONICAL_CLIENT_CONFIG_API_URL=http://localhost:8080", "CANONICAL_CLIENT_CONFIG_WEB_OIDC_CLIENT_ID=canonical-client-config-web", "CANONICAL_CLIENT_CONFIG_MOBILE_OIDC_CLIENT_ID=canonical-client-config-mobile"},
		"compose.yaml":                          {"CANONICAL_CLIENT_CONFIG_APP_ENV: development", "CANONICAL_CLIENT_CONFIG_API_URL:", "CANONICAL_CLIENT_CONFIG_WEB_OIDC_CLIENT_ID:"},
		"apps/mobile/app.config.ts":             {"process.env.CANONICAL_CLIENT_CONFIG_APP_ENV", "process.env.CANONICAL_CLIENT_CONFIG_API_URL", "process.env.CANONICAL_CLIENT_CONFIG_MOBILE_OIDC_CLIENT_ID"},
		"apps/mobile/src/config.ts":             {"loadMobileConfig", "clientRuntimeConfig"},
		"apps/web/src/lib/server/config.ts":     {"$env/dynamic/private", "CANONICAL_CLIENT_CONFIG_APP_ENV", "CANONICAL_CLIENT_CONFIG_WEB_OIDC_CLIENT_ID"},
		"apps/web/src/routes/+layout.server.ts": {"config: webConfig"},
	}
	for path, evidence := range files {
		body, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Errorf("generated configuration file %s is missing: %v", path, err)
			continue
		}
		for _, required := range evidence {
			if !strings.Contains(string(body), required) {
				t.Errorf("%s lacks %q", path, required)
			}
		}
	}
	for _, path := range []string{".env.example", "compose.yaml", "apps/mobile/app/index.tsx", "apps/web/src/lib/config.ts", "apps/web/src/routes/+page.svelte"} {
		body, _ := os.ReadFile(filepath.Join(dir, path))
		for _, forbidden := range []string{"PUBLIC_APP_ENV", "PUBLIC_API_URL", "PUBLIC_OIDC_ISSUER", "PUBLIC_OIDC_CLIENT_ID", "EXPO_PUBLIC_"} {
			if strings.Contains(string(body), forbidden) {
				t.Errorf("%s exposes framework-specific configuration %q", path, forbidden)
			}
		}
	}
}

func TestGeneratedBuildKitVerificationIsPinnedAndCacheAware(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "buildkit-verification")
	if err := run([]string{"new", "BuildKit Verification", "--module", "example.com/buildkit-verification", "--output", dir, "--without-example"}); err != nil {
		t.Fatal(err)
	}

	read := func(relative string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(dir, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(body)
	}

	installer := read("scripts/install-docker-buildx.sh")
	for _, evidence := range []string{
		`readonly version="v0.35.0"`,
		`readonly asset="buildx-v0.35.0.linux-amd64"`,
		`readonly sha256="d41ece72044243b4f58b343441ae37446d9c29a7d6b5e11c61847bbcf8f7dfda"`,
		"gh release download", "sha256sum --check",
	} {
		if !strings.Contains(installer, evidence) {
			t.Errorf("immutable Buildx installer lacks %q", evidence)
		}
	}
	if strings.Contains(installer, "latest") || strings.Contains(installer, "curl ") || strings.Contains(installer, "wget ") {
		t.Error("Buildx installer may not resolve a floating or alternate download")
	}

	requirement := read("scripts/require-docker-buildkit.sh")
	for _, evidence := range []string{`required_version="v0.35.0"`, `BUILDX_BUILDER:-default`, `buildx inspect "$builder" --bootstrap`} {
		if !strings.Contains(requirement, evidence) {
			t.Errorf("BuildKit capability gate lacks %q", evidence)
		}
	}

	verification := read("scripts/verify-docker-builds.sh")
	for _, evidence := range []string{
		`export BUILDX_BUILDER="${BUILDX_BUILDER:-default}"`,
		"export COMPOSE_BAKE=true",
		`docker compose build`,
		`docker buildx build --builder "$BUILDX_BUILDER" --load --progress plain`,
		`assert-buildkit-cache-hit.sh "$api_build_log" 'CGO_ENABLED=0 go build'`,
		`assert-buildkit-cache-hit.sh "$web_build_log" 'pnpm --dir apps/web build'`,
		`docker image inspect buildkit-verification-api:verification buildkit-verification-web:verification`,
		`mktemp -d "${TMPDIR:-/tmp}/buildkit-verification-build-logs.XXXXXX"`,
	} {
		if !strings.Contains(verification, evidence) {
			t.Errorf("Docker verification lacks %q", evidence)
		}
	}
	if strings.Contains(verification, "docker build ") {
		t.Error("Docker verification may not fall back to the legacy builder")
	}

	makefile := read("Makefile")
	for _, fixture := range []string{"test-install-docker-buildx.sh", "test-require-docker-buildkit.sh", "test-verify-docker-builds.sh"} {
		if !strings.Contains(makefile, fixture) {
			t.Errorf("generated check omits fixture %s", fixture)
		}
	}
	workflow := read(".github/workflows/ci.yml")
	if !strings.Contains(workflow, "./scripts/verify-docker-builds.sh") || strings.Contains(workflow, "docker build --file") {
		t.Error("generated CI must use the fail-closed BuildKit verifier")
	}
}

func TestGeneratorPublishesCrossHostGenerationMatrix(t *testing.T) {
	body, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(body)
	for _, evidence := range []string{"ubuntu-24.04", "macos-15", "default", "without-example", "all-field-types", "irregular-plural", "multiple-domains", "example-removal", "schema-compatibility", "identifier-boundaries", "existing-repository-adoption", "mobile:build:android", "mobile:build:ios"} {
		if !strings.Contains(workflow, evidence) {
			t.Errorf("generator CI does not publish %q case", evidence)
		}
	}
	matrix, err := os.ReadFile("scripts/generation-case.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, evidence := range []string{"go test ./...", "go run ./cmd/openapi", "verify_project"} {
		if !strings.Contains(string(matrix), evidence) {
			t.Errorf("generation matrix does not verify %q", evidence)
		}
	}
}

func TestGeneratorAcceptanceExercisesGeneratedBuildKitVerification(t *testing.T) {
	body, err := os.ReadFile("scripts/acceptance.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"$work/secure-app/scripts/verify-docker-builds.sh"`) {
		t.Error("generator acceptance must execute the generated BuildKit verifier")
	}
}
