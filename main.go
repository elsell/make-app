package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/token"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	modmodule "golang.org/x/mod/module"
)

const templateSchemaVersion = 4

var commandName = func(name string) string { return name }

//go:embed template
var templates embed.FS

//go:embed LICENSE
var apacheLicense string

type values struct {
	Name, Slug, NativeID, BundlePrefix, Module, EnvPrefix, Domain, DomainGoName, DomainPlural, MigrationVersion string
	DomainGoFields, DomainSQLFields, DomainModelFields                                                          string
	DomainEntityToModelFields, DomainModelToEntityFields                                                        string
	DomainDTOFields, DomainDTOImports                                                                           string
	DomainDTOToAttributesFields, DomainAttributesToDTOFields                                                    string
	DomainUpdateMapFields, DomainTestFields, DomainStringValidation                                             string
	DomainTestJSON                                                                                              string
}

type fieldDefinition struct{ Name, GoName, Kind, GoType, SQLType string }

type fileSnapshot struct {
	body []byte
	mode fs.FileMode
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "make-app:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: make-app version | make-app doctor | make-app new NAME --module MODULE [--bundle-prefix PREFIX] [--output DIR] [--without-example] | make-app init NAME --module MODULE [--bundle-prefix PREFIX] [--dir DIR] [--without-example] | make-app domain add NAME [--dir DIR] [--plural PLURAL] [--fields SPEC] | make-app example remove [--dir DIR]")
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			return errors.New("version accepts no arguments")
		}
		fmt.Println(generatorVersion())
		return nil
	case "doctor":
		return doctor(args[1:])
	case "new":
		return newApp(args[1:])
	case "init":
		return initApp(args[1:])
	case "domain":
		if len(args) > 1 && args[1] == "add" {
			return addDomain(args[2:])
		}
	case "example":
		if len(args) > 1 && args[1] == "remove" {
			return removeExample(args[2:])
		}
	}
	return fmt.Errorf("unknown command %q", strings.Join(args, " "))
}

func generatorVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

type doctorFailure struct{ check, detail string }

func doctor(args []string) error {
	f := flag.NewFlagSet("doctor", flag.ContinueOnError)
	skipPorts := f.Bool("skip-ports", false, "skip local development port availability checks")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("doctor accepts only --skip-ports")
	}
	checks := []struct {
		name string
		args []string
	}{
		{"git", []string{"--version"}}, {"go", []string{"version"}}, {"gofmt", nil},
		{"node", []string{"--version"}}, {"pnpm", []string{"--version"}}, {"python3", []string{"--version"}},
		{"make", []string{"--version"}}, {"docker", []string{"compose", "version"}},
	}
	failures := make([]doctorFailure, 0)
	warnings := []string{}
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath(commandName("xcodebuild")); err != nil {
			warnings = append(warnings, "iOS native builds require Xcode and xcodebuild")
		}
		if ruby, err := exec.LookPath(commandName("ruby")); err != nil {
			warnings = append(warnings, "iOS native builds require Ruby 3.2.9")
		} else if err := exec.Command(ruby, "-e", `exit(RUBY_VERSION == "3.2.9" ? 0 : 1)`).Run(); err != nil {
			warnings = append(warnings, "iOS native builds require the reviewed Ruby 3.2.9 toolchain")
		}
		if bundle, err := exec.LookPath(commandName("bundle")); err != nil {
			warnings = append(warnings, "iOS native builds require Bundler 2.6.9")
		} else if output, versionErr := exec.Command(bundle, "--version").CombinedOutput(); versionErr != nil || !strings.Contains(string(output), "2.6.9") {
			warnings = append(warnings, "iOS native builds require Bundler 2.6.9 from Gemfile.lock")
		}
		if _, err := exec.LookPath(commandName("java")); err != nil {
			warnings = append(warnings, "Android native builds require a JDK and Android SDK")
		}
		if os.Getenv("ANDROID_HOME") == "" && os.Getenv("ANDROID_SDK_ROOT") == "" {
			warnings = append(warnings, "Android native builds require ANDROID_HOME or ANDROID_SDK_ROOT")
		}
		if !androidSDKToolAvailable() {
			warnings = append(warnings, "Android native builds require adb or sdkmanager from a usable Android SDK")
		}
	case "linux":
		warnings = append(warnings, "iOS native compilation requires a supported macOS runner")
		if _, err := exec.LookPath(commandName("java")); err != nil {
			warnings = append(warnings, "Android native builds require a JDK and Android SDK")
		}
		if os.Getenv("ANDROID_HOME") == "" && os.Getenv("ANDROID_SDK_ROOT") == "" {
			warnings = append(warnings, "Android native builds require ANDROID_HOME or ANDROID_SDK_ROOT")
		}
		if !androidSDKToolAvailable() {
			warnings = append(warnings, "Android native builds require adb or sdkmanager from a usable Android SDK")
		}
	case "windows":
		failures = append(failures, doctorFailure{"host", "native Windows is unsupported; use WSL2 with the documented Linux workflow"})
	default:
		failures = append(failures, doctorFailure{"host", "unsupported host; use macOS or Linux"})
	}
	for _, check := range checks {
		if len(check.args) == 0 {
			if _, err := exec.LookPath(commandName(check.name)); err != nil {
				failures = append(failures, doctorFailure{check.name, err.Error()})
			}
			continue
		}
		cmd := exec.Command(commandName(check.name), check.args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			detail := strings.TrimSpace(string(output))
			if detail == "" {
				detail = err.Error()
			}
			failures = append(failures, doctorFailure{check.name, detail})
		} else if !validToolVersion(check.name, string(output)) {
			failures = append(failures, doctorFailure{check.name, "unsupported version: " + strings.TrimSpace(string(output))})
		}
	}
	if !*skipPorts {
		for _, port := range []string{"5432", "50051", "5556", "8080", "5173"} {
			listener, err := net.Listen("tcp", "127.0.0.1:"+port)
			if err != nil {
				failures = append(failures, doctorFailure{"port " + port, "already in use"})
				continue
			}
			_ = listener.Close()
		}
	}
	if len(failures) > 0 {
		var details []string
		for _, failure := range failures {
			details = append(details, failure.check+": "+failure.detail)
		}
		return fmt.Errorf("prerequisite checks failed:\n  - %s", strings.Join(details, "\n  - "))
	}
	fmt.Printf("supported host: %s; macOS and Linux are supported, Windows requires WSL2\n", runtime.GOOS)
	for _, warning := range warnings {
		fmt.Printf("native mobile note: %s\n", warning)
	}
	fmt.Println("make-app doctor passed")
	return nil
}

func androidSDKToolAvailable() bool {
	for _, tool := range []string{"adb", "sdkmanager"} {
		if _, err := exec.LookPath(commandName(tool)); err == nil {
			return true
		}
	}
	root := os.Getenv("ANDROID_HOME")
	if root == "" {
		root = os.Getenv("ANDROID_SDK_ROOT")
	}
	if root == "" {
		return false
	}
	for _, relative := range []string{filepath.Join("platform-tools", "adb"), filepath.Join("cmdline-tools", "latest", "bin", "sdkmanager"), filepath.Join("tools", "bin", "sdkmanager")} {
		if info, err := os.Stat(filepath.Join(root, relative)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func validToolVersion(name, output string) bool {
	match := regexp.MustCompile(`(?i)(?:go|node|python|git version |docker compose version v?|gnu make )?v?(\d+)\.(\d+)(?:\.(\d+))?`).FindStringSubmatch(output)
	if len(match) < 3 {
		return name == "make"
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	switch name {
	case "go":
		return major > 1 || (major == 1 && minor >= 25)
	case "node":
		return major >= 22 && major < 25
	case "pnpm":
		return major == 11
	case "python3":
		return major > 3 || (major == 3 && minor >= 10)
	case "git":
		return major > 2 || (major == 2 && minor >= 40)
	case "docker":
		return major > 2 || (major == 2 && minor >= 20)
	default:
		return true
	}
}

func newApp(args []string) error {
	args = positionalLast(args)
	f := flag.NewFlagSet("new", flag.ContinueOnError)
	module := f.String("module", "", "Go module")
	output := f.String("output", "", "output directory")
	withoutExample := f.Bool("without-example", false, "omit the removable example product slice")
	bundlePrefix := f.String("bundle-prefix", "com.example", "reverse-DNS iOS and Android bundle prefix")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 1 || *module == "" {
		return errors.New("new requires NAME and --module")
	}
	if !validApplicationName(f.Arg(0)) {
		return errors.New("NAME must be 1-80 characters and contain only letters, digits, spaces, periods, underscores, or hyphens")
	}
	v := appValues(f.Arg(0), *module)
	if !regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$`).MatchString(*bundlePrefix) {
		return errors.New("--bundle-prefix must be a lowercase reverse-DNS identifier")
	}
	v.BundlePrefix = *bundlePrefix
	dest := *output
	if dest == "" {
		dest = v.Slug
	}
	if v.Slug == "" || modmodule.CheckPath(*module) != nil {
		return errors.New("NAME and --module must be valid project identifiers")
	}
	destinationExists := false
	if entries, err := os.ReadDir(dest); err == nil {
		destinationExists = true
		if len(entries) > 0 {
			return fmt.Errorf("destination %s is not empty", dest)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(parent, ".make-app-new-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := renderApplication(stage, v, *withoutExample); err != nil {
		return err
	}
	if err := initializeGit(stage); err != nil {
		return err
	}
	if destinationExists {
		if err := os.Remove(dest); err != nil {
			return err
		}
	}
	if err := os.Rename(stage, dest); err != nil {
		return fmt.Errorf("install generated application: %w", err)
	}
	fmt.Printf("generated %s; run: cd %s && make bootstrap\n", v.Name, dest)
	return nil
}

func renderApplication(stage string, v values, withoutExample bool) error {
	if err := renderTree("template/base", stage, v); err != nil {
		return err
	}
	projectLicense := strings.Replace(apacheLicense, "Copyright 2026 make-app contributors", "Copyright 2026 "+v.Name+" contributors", 1)
	if err := os.WriteFile(filepath.Join(stage, "LICENSE"), []byte(projectLicense), 0o644); err != nil {
		return err
	}
	domains := []domainManifest{}
	if !withoutExample {
		example, err := withFields(withDomain(v, "example"), "name:string")
		if err != nil {
			return err
		}
		if err := renderTree("template/domain", stage, example); err != nil {
			return err
		}
		domains = append(domains, domainManifest{Name: "example", Plural: "examples", Fields: "name:string"})
	} else if err := omitExampleStorage(stage); err != nil {
		return err
	} else if err := renderNoExampleOverlay(stage, v); err != nil {
		return err
	}
	manifest := projectManifest{SchemaVersion: templateSchemaVersion, GeneratorVersion: generatorVersion(), Name: v.Name, Slug: v.Slug, BundlePrefix: v.BundlePrefix, Module: v.Module, Domains: domains}
	if err := writeDomainRegistry(stage, manifest); err != nil {
		return err
	}
	if err := writeProjectManifest(stage, manifest); err != nil {
		return err
	}
	return formatGeneratedGo(stage)
}

var adoptionTopLevel = map[string]bool{
	".git": true, ".codex": true, ".gitignore": true, "AGENTS.md": true,
	"README.md": true, "LICENSE": true, "specs": true, "docs": true,
}

func initApp(args []string) (returnErr error) {
	args = positionalLast(args)
	f := flag.NewFlagSet("init", flag.ContinueOnError)
	module := f.String("module", "", "Go module")
	dir := f.String("dir", ".", "existing Git repository")
	withoutExample := f.Bool("without-example", false, "omit the removable example product slice")
	bundlePrefix := f.String("bundle-prefix", "com.example", "reverse-DNS iOS and Android bundle prefix")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 1 || *module == "" {
		return errors.New("init requires NAME and --module")
	}
	if !validApplicationName(f.Arg(0)) {
		return errors.New("NAME must be 1-80 characters and contain only letters, digits, spaces, periods, underscores, or hyphens")
	}
	v := appValues(f.Arg(0), *module)
	if v.Slug == "" || modmodule.CheckPath(*module) != nil {
		return errors.New("NAME and --module must be valid project identifiers")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$`).MatchString(*bundlePrefix) {
		return errors.New("--bundle-prefix must be a lowercase reverse-DNS identifier")
	}
	v.BundlePrefix = *bundlePrefix
	if err := validateGitWorktreeRoot(*dir); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(*dir, ".make-app.json")); err == nil {
		return errors.New("repository is already managed by make-app")
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(*dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !adoptionTopLevel[entry.Name()] {
			return fmt.Errorf("existing path %s is not allowed for spec-first adoption", entry.Name())
		}
	}
	if err := rejectAdoptionSymlinks(*dir); err != nil {
		return err
	}
	parent := filepath.Dir(filepath.Clean(*dir))
	stage, err := os.MkdirTemp(parent, ".make-app-init-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := renderApplication(stage, v, *withoutExample); err != nil {
		return err
	}
	if err := mergeAdoptionContent(*dir, stage); err != nil {
		return err
	}
	snapshot, err := snapshotAdoption(*dir)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed && returnErr != nil {
			if restoreErr := restoreAdoption(*dir, snapshot); restoreErr != nil {
				returnErr = fmt.Errorf("%w; rollback failed: %v", returnErr, restoreErr)
			}
		}
	}()
	if err := copyTree(stage, *dir, true); err != nil {
		return err
	}
	if err := installGitHooks(*dir); err != nil {
		return err
	}
	committed = true
	fmt.Printf("adopted %s in %s without replacing Git history; run: cd %s && make bootstrap\n", v.Name, *dir, *dir)
	return nil
}

func validateGitWorktreeRoot(dir string) error {
	cmd := exec.Command(commandName("git"), "-C", dir, "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("init requires an existing Git worktree: %s", strings.TrimSpace(string(output)))
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	want, err = filepath.Abs(want)
	if err != nil {
		return err
	}
	got, err := filepath.EvalSymlinks(strings.TrimSpace(string(output)))
	if err != nil {
		return err
	}
	got, err = filepath.Abs(got)
	if err != nil {
		return err
	}
	if filepath.Clean(want) != filepath.Clean(got) {
		return errors.New("init --dir must be the Git worktree root")
	}
	return nil
}

func rejectAdoptionSymlinks(root string) error {
	for name := range adoptionTopLevel {
		if name == ".git" {
			continue
		}
		path := filepath.Join(root, name)
		err := filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if os.IsNotExist(walkErr) {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				relative, _ := filepath.Rel(root, current)
				return fmt.Errorf("existing path %s is a symlink; adoption refuses symlinks", filepath.ToSlash(relative))
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func mergeAdoptionContent(existing, stage string) error {
	for _, file := range []struct{ name, begin, end string }{
		{"AGENTS.md", "<!-- BEGIN MAKE-APP BASELINE GUIDANCE -->", "<!-- END MAKE-APP BASELINE GUIDANCE -->"},
		{"README.md", "<!-- BEGIN MAKE-APP GENERATED PLATFORM -->", "<!-- END MAKE-APP GENERATED PLATFORM -->"},
	} {
		current, err := os.ReadFile(filepath.Join(existing, file.name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		generated, err := os.ReadFile(filepath.Join(stage, file.name))
		if err != nil {
			return err
		}
		if strings.Contains(string(current), file.begin) {
			return fmt.Errorf("existing %s already contains make-app merge markers", file.name)
		}
		merged := strings.TrimRight(string(current), "\n") + "\n\n" + file.begin + "\n" + string(generated) + file.end + "\n"
		if err := os.WriteFile(filepath.Join(stage, file.name), []byte(merged), 0o644); err != nil {
			return err
		}
	}
	if current, err := os.ReadFile(filepath.Join(existing, ".gitignore")); err == nil {
		generated, readErr := os.ReadFile(filepath.Join(stage, ".gitignore"))
		if readErr != nil {
			return readErr
		}
		seen := map[string]bool{}
		lines := []string{}
		for _, body := range [][]byte{current, generated} {
			for _, line := range strings.Split(string(body), "\n") {
				if !seen[line] {
					seen[line] = true
					lines = append(lines, line)
				}
			}
		}
		if err := os.WriteFile(filepath.Join(stage, ".gitignore"), []byte(strings.TrimRight(strings.Join(lines, "\n"), "\n")+"\n"), 0o644); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if current, err := os.ReadFile(filepath.Join(existing, "LICENSE")); err == nil {
		if !strings.Contains(string(current), "Apache License") || !strings.Contains(string(current), "Version 2.0") {
			return errors.New("existing LICENSE conflicts with required Apache-2.0 license")
		}
		if err := os.WriteFile(filepath.Join(stage, "LICENSE"), current, 0o644); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, directory := range []string{"specs", "docs", ".codex"} {
		root := filepath.Join(existing, directory)
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := copyTree(root, filepath.Join(stage, directory), false); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(source, destination string, overwrite bool) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("source path %s is a symlink", filepath.ToSlash(relative))
		}
		target := filepath.Join(destination, relative)
		if err := rejectDestinationSymlink(destination, target, relative); err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if existing, statErr := os.ReadFile(target); statErr == nil && !overwrite {
			if string(existing) != string(body) {
				return fmt.Errorf("existing path %s conflicts with generated content", filepath.ToSlash(relative))
			}
			return nil
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, info.Mode().Perm())
	})
}

func rejectDestinationSymlink(root, target, relative string) error {
	path := filepath.Clean(root)
	parts, err := filepath.Rel(path, target)
	if err != nil || parts == ".." || strings.HasPrefix(parts, ".."+string(filepath.Separator)) {
		return fmt.Errorf("destination path %s escapes its root", filepath.ToSlash(relative))
	}
	for _, part := range strings.Split(parts, string(filepath.Separator)) {
		path = filepath.Join(path, part)
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination path %s traverses a symlink", filepath.ToSlash(relative))
		}
	}
	return nil
}

func snapshotAdoption(root string) (map[string]fileSnapshot, error) {
	result := map[string]fileSnapshot{}
	for name := range adoptionTopLevel {
		if name == ".git" {
			continue
		}
		path := filepath.Join(root, name)
		walkErr := filepath.Walk(path, func(current string, info os.FileInfo, err error) error {
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			body, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			result[relative] = fileSnapshot{body: body, mode: info.Mode().Perm()}
			return nil
		})
		if walkErr != nil && !os.IsNotExist(walkErr) {
			return nil, walkErr
		}
	}
	return result, nil
}

func restoreAdoption(root string, snapshot map[string]fileSnapshot) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != ".git" {
			if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
				return err
			}
		}
	}
	for relative, saved := range snapshot {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, saved.body, saved.mode); err != nil {
			return err
		}
	}
	return nil
}

func omitExampleStorage(root string) error {
	baselinePath := filepath.Join(root, "apps/api/internal/adapters/dbmigrations/000001_baseline.up.sql")
	body, err := os.ReadFile(baselinePath)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`(?s)CREATE TABLE resource_models \(.*?CREATE INDEX resource_models_owner_user_id_idx ON resource_models\(owner_user_id\);\n`)
	updated := pattern.ReplaceAll(body, nil)
	if len(updated) == len(body) {
		return errors.New("example storage block is missing from baseline migration")
	}
	if err := os.WriteFile(baselinePath, updated, 0o644); err != nil {
		return err
	}
	for _, name := range []string{"000003_resource_created_at.up.sql", "000003_resource_created_at.down.sql"} {
		if err := os.WriteFile(filepath.Join(root, "apps/api/internal/adapters/dbmigrations", name), []byte("-- No-op: this project was generated without the example resource slice.\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

var exampleClientPaths = []string{
	"apps/web/src/routes/+page.svelte",
	"apps/mobile/app/index.tsx",
	"packages/api-client/src/index.test.ts",
	"scripts/seed.sh",
	"scripts/live-acceptance.sh",
	"scripts/scalar-browser-acceptance.mjs",
}

func renderNoExampleOverlay(root string, v values) error {
	for _, path := range exampleClientPaths {
		if err := os.Remove(filepath.Join(root, path)); err != nil {
			return err
		}
	}
	return renderTree("template/no_example", root, v)
}

func ensureNoExampleDependencies(root, modulePath string) error {
	const knownPlatformReference = "apps/api/internal/adapters/httpserver/server_test.go"
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == ".svelte-kit" || entry.Name() == "build" || entry.Name() == "dist" || relative == "apps/api/internal/domain/example" {
				return filepath.SkipDir
			}
			return nil
		}
		for _, owned := range exampleClientPaths {
			if relative == owned {
				return nil
			}
		}
		if relative == "packages/api-client/openapi.json" || relative == "packages/api-client/src/schema.d.ts" {
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".ts", ".tsx", ".js", ".mjs", ".svelte":
		default:
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		depends := strings.Contains(string(body), modulePath+"/apps/api/internal/domain/example") || strings.Contains(string(body), "/v1/examples")
		if !depends {
			return nil
		}
		if relative == knownPlatformReference {
			return nil
		}
		return fmt.Errorf("refusing to remove example: %s depends on the example domain or REST collection", relative)
	})
}

func removeExample(args []string) (returnErr error) {
	f := flag.NewFlagSet("example remove", flag.ContinueOnError)
	dir := f.String("dir", ".", "application directory")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("example remove accepts only --dir")
	}
	manifest, err := readProjectManifest(*dir)
	if err != nil {
		return err
	}
	index := -1
	for i, domain := range manifest.Domains {
		if domain.Name == "example" {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("example domain is not installed")
	}
	v := appValues(manifest.Name, manifest.Module)
	v.BundlePrefix = manifest.BundlePrefix
	originalClients := make(map[string]fileSnapshot, len(exampleClientPaths))
	for _, relative := range exampleClientPaths {
		path := filepath.Join(*dir, relative)
		current, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		baseline, readErr := templates.ReadFile("template/base/" + relative)
		if readErr != nil {
			return readErr
		}
		if string(current) != replace(string(baseline), v) {
			return fmt.Errorf("refusing to remove example: %s was modified and may depend on it", relative)
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return statErr
		}
		originalClients[path] = fileSnapshot{body: current, mode: info.Mode()}
	}
	if err := ensureNoExampleDependencies(*dir, manifest.Module); err != nil {
		return err
	}
	contractSnapshots, err := snapshotFiles([]string{filepath.Join(*dir, "packages/api-client/openapi.json"), filepath.Join(*dir, "packages/api-client/src/schema.d.ts")})
	if err != nil {
		return err
	}
	version, err := nextMigrationVersion(*dir)
	if err != nil {
		return err
	}
	upPath := filepath.Join(*dir, "apps/api/internal/adapters/dbmigrations", fmt.Sprintf("%06d_remove_example_resources.up.sql", version))
	downPath := filepath.Join(*dir, "apps/api/internal/adapters/dbmigrations", fmt.Sprintf("%06d_remove_example_resources.down.sql", version))
	for _, path := range []string{upPath, downPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return fmt.Errorf("refusing to overwrite %s", path)
		}
	}
	migrationMetadata := filepath.Join(*dir, "apps/api/internal/adapters/dbmigrations/migrations.go")
	registryPath := filepath.Join(*dir, "apps/api/internal/generated/domains.go")
	manifestPath := filepath.Join(*dir, ".make-app.json")
	originalMetadata, err := os.ReadFile(migrationMetadata)
	if err != nil {
		return err
	}
	originalRegistry, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}
	originalManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	examplePath := filepath.Join(*dir, "apps/api/internal/domain/example")
	backup, err := os.MkdirTemp(*dir, ".make-app-example-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(backup)
	backupExample := filepath.Join(backup, "example")
	if err := os.Rename(examplePath, backupExample); err != nil {
		return fmt.Errorf("remove example source: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		var rollbackErr error
		for _, path := range []string{upPath, downPath} {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		for path, body := range map[string][]byte{migrationMetadata: originalMetadata, registryPath: originalRegistry, manifestPath: originalManifest} {
			if err := os.WriteFile(path, body, 0o644); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		rollbackErr = errors.Join(rollbackErr, restoreFiles(originalClients))
		rollbackErr = errors.Join(rollbackErr, restoreFiles(contractSnapshots))
		if err := os.Rename(backupExample, examplePath); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		returnErr = errors.Join(returnErr, rollbackErr)
	}()
	up := "DROP TABLE resource_models;\n"
	down := `CREATE TABLE resource_models (
    id text PRIMARY KEY,
    domain text NOT NULL DEFAULT 'example',
    owner_user_id text NOT NULL REFERENCES user_models(id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX resource_models_domain_idx ON resource_models(domain);
CREATE INDEX resource_models_owner_user_id_idx ON resource_models(owner_user_id);
CREATE INDEX resource_models_owner_domain_created_id_idx ON resource_models(owner_user_id, domain, created_at, id);
GRANT SELECT, INSERT, UPDATE, DELETE ON resource_models TO app;
`
	if err := os.WriteFile(upPath, []byte(up), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(downPath, []byte(down), 0o644); err != nil {
		return err
	}
	if err := updateLatestMigrationVersion(*dir, version); err != nil {
		return err
	}
	manifest.Domains = append(manifest.Domains[:index], manifest.Domains[index+1:]...)
	if err := writeProjectManifest(*dir, manifest); err != nil {
		return err
	}
	if err := writeDomainRegistry(*dir, manifest); err != nil {
		return err
	}
	if err := renderNoExampleOverlay(*dir, v); err != nil {
		return err
	}
	contractStep := "contract generation deferred until make bootstrap"
	if _, statErr := os.Stat(filepath.Join(*dir, "node_modules")); statErr == nil {
		generate := exec.Command(commandName("make"), "generate")
		generate.Dir = *dir
		if output, generateErr := generate.CombinedOutput(); generateErr != nil {
			return fmt.Errorf("contract generation failed; example removal was rolled back: %w: %s", generateErr, strings.TrimSpace(string(output)))
		}
		contractStep = "OpenAPI and TypeScript contracts regenerated"
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	committed = true
	fmt.Printf("removed example slice; %s; run make verify\n", contractStep)
	return nil
}
func initializeGit(dest string) error {
	cmd := exec.Command(commandName("git"), "init", "-b", "main")
	cmd.Dir = dest
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("initialize git: %w: %s", err, output)
	}
	return installGitHooks(dest)
}

func installGitHooks(dest string) error {
	hooks := []struct{ name, body string }{
		{"pre-commit", "#!/usr/bin/env sh\nset -eu\nmake pre-commit\n"},
		{"pre-push", "#!/usr/bin/env sh\nset -eu\nmake pre-push\n"},
	}
	existing := map[string]*fileSnapshot{}
	for _, hook := range hooks {
		path, err := gitHookPath(dest, hook.name)
		if err != nil {
			return err
		}
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("existing Git hook %s is a symlink", hook.name)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
		if current, err := os.ReadFile(path); err == nil {
			if string(current) != hook.body {
				return fmt.Errorf("existing Git hook %s conflicts with make-app", hook.name)
			}
			info, statErr := os.Stat(path)
			if statErr != nil {
				return statErr
			}
			snapshot := fileSnapshot{body: current, mode: info.Mode().Perm()}
			existing[hook.name] = &snapshot
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	rollback := func() {
		for _, hook := range hooks {
			path, err := gitHookPath(dest, hook.name)
			if err != nil {
				continue
			}
			if saved := existing[hook.name]; saved != nil {
				_ = os.WriteFile(path, saved.body, saved.mode)
			} else {
				_ = os.Remove(path)
			}
		}
	}
	for _, hook := range hooks {
		name, body := hook.name, hook.body
		path, err := gitHookPath(dest, name)
		if err != nil {
			rollback()
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			rollback()
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			rollback()
			return err
		}
	}
	return nil
}

func gitHookPath(dest, name string) (string, error) {
	cmd := exec.Command(commandName("git"), "-C", dest, "rev-parse", "--git-path", filepath.ToSlash(filepath.Join("hooks", name)))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve Git hook path: %w: %s", err, strings.TrimSpace(string(output)))
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", errors.New("Git returned an empty hook path")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dest, path)
	}
	path, err = filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	commonCommand := exec.Command(commandName("git"), "-C", dest, "rev-parse", "--git-common-dir")
	commonOutput, commonErr := commonCommand.CombinedOutput()
	if commonErr != nil {
		return "", fmt.Errorf("resolve Git common directory: %w: %s", commonErr, strings.TrimSpace(string(commonOutput)))
	}
	common := strings.TrimSpace(string(commonOutput))
	if !filepath.IsAbs(common) {
		common = filepath.Join(dest, common)
	}
	common, err = filepath.Abs(filepath.Clean(common))
	if err != nil {
		return "", err
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	resolvedPath, err := resolveExistingPathPrefix(path)
	if err != nil {
		return "", err
	}
	resolvedDest, err := filepath.EvalSymlinks(absDest)
	if err != nil {
		return "", err
	}
	resolvedCommon, err := filepath.EvalSymlinks(filepath.Clean(common))
	if err != nil {
		return "", err
	}
	if !pathInside(resolvedPath, resolvedDest) && !pathInside(resolvedPath, resolvedCommon) {
		return "", fmt.Errorf("configured Git hooks path is outside the repository")
	}
	return path, nil
}

func pathInside(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func resolveExistingPathPrefix(path string) (string, error) {
	current := path
	var suffix []string
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(current)
			if resolveErr != nil {
				return "", resolveErr
			}
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("cannot resolve Git hook path %s", path)
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

func addDomain(args []string) (returnErr error) {
	args = positionalLast(args)
	f := flag.NewFlagSet("domain add", flag.ContinueOnError)
	dir := f.String("dir", ".", "application directory")
	plural := f.String("plural", "", "explicit plural resource name")
	fieldSpec := f.String("fields", "name:string", "comma-separated domain fields")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 1 {
		return errors.New("domain add requires NAME")
	}
	moduleBytes, err := os.ReadFile(filepath.Join(*dir, "apps/api/go.mod"))
	if err != nil {
		return errors.New("target is not a generated make-app project")
	}
	fields := strings.Fields(string(moduleBytes))
	if len(fields) < 2 {
		return errors.New("invalid Go module")
	}
	rootModule := strings.TrimSuffix(fields[1], "/apps/api")
	if rootModule == fields[1] {
		return errors.New("generated API module must end in /apps/api")
	}
	manifest, err := readProjectManifest(*dir)
	if err != nil {
		return err
	}
	if manifest.Module != rootModule {
		return errors.New("generated project manifest module does not match apps/api/go.mod")
	}
	v := withDomain(appValues(manifest.Name, rootModule), f.Arg(0))
	if *plural != "" {
		v.DomainPlural = strings.ReplaceAll(slugify(*plural), "-", "_")
		if len(v.DomainPlural) > 40 || !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(v.DomainPlural) {
			return errors.New("plural must be at most 40 characters, begin with a letter, and contain only letters, digits, or underscores")
		}
	}
	target := filepath.Join(*dir, "apps/api/internal/domain", v.Domain)
	if len(v.Domain) > 40 || !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(v.Domain) || token.IsKeyword(v.Domain) {
		return errors.New("domain name must be at most 40 characters, begin with a letter, and contain only letters, digits, or underscores")
	}
	reservedCollections := map[string]struct{}{"me": {}, "session": {}, "sessions": {}, "invitations": {}, "audit_events": {}}
	if _, reserved := reservedCollections[v.DomainPlural]; reserved {
		return fmt.Errorf("REST collection %q is reserved by the platform", v.DomainPlural)
	}
	for _, installedDomain := range manifest.Domains {
		if installedDomain.Plural == v.DomainPlural {
			return fmt.Errorf("REST collection %q is already used by domain %s", v.DomainPlural, installedDomain.Name)
		}
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("domain %s already exists", v.Domain)
	} else if !os.IsNotExist(err) {
		return err
	}
	migrationVersion, err := nextMigrationVersion(*dir)
	if err != nil {
		return err
	}
	v.MigrationVersion = fmt.Sprintf("%06d", migrationVersion)
	if err := validateFieldSpec(*fieldSpec); err != nil {
		return err
	}
	v, err = withFields(v, *fieldSpec)
	if err != nil {
		return err
	}
	migrationPath := filepath.Join(*dir, "apps/api/internal/adapters/dbmigrations/migrations.go")
	originalMigration, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(*dir, ".make-app.json")
	originalManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	registryPath := filepath.Join(*dir, "apps/api/internal/generated/domains.go")
	originalRegistry, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}
	contractPaths := []string{
		filepath.Join(*dir, "packages/api-client/openapi.json"),
		filepath.Join(*dir, "packages/api-client/src/schema.d.ts"),
	}
	contractSnapshots, err := snapshotFiles(contractPaths)
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(*dir, ".make-app-domain-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := renderTree("template/domain", stage, v); err != nil {
		return err
	}
	if err := renderTree("template/domain_add", stage, v); err != nil {
		return err
	}
	if err := formatGeneratedGo(stage); err != nil {
		return err
	}
	installed, err := installStagedTree(stage, *dir)
	if err != nil {
		return errors.Join(err, rollbackDomainAdd(*dir, installed, migrationPath, originalMigration, manifestPath, originalManifest, registryPath, originalRegistry))
	}
	committed := false
	defer func() {
		if !committed {
			returnErr = errors.Join(returnErr, rollbackDomainAdd(*dir, installed, migrationPath, originalMigration, manifestPath, originalManifest, registryPath, originalRegistry), restoreFiles(contractSnapshots))
		}
	}()
	if err := updateLatestMigrationVersion(*dir, migrationVersion); err != nil {
		return err
	}
	manifest.Domains = append(manifest.Domains, domainManifest{Name: v.Domain, Plural: v.DomainPlural, Fields: *fieldSpec})
	if err = writeProjectManifest(*dir, manifest); err != nil {
		return err
	}
	if err = writeDomainRegistry(*dir, manifest); err != nil {
		return err
	}
	contractStep := "contract generation deferred until make bootstrap"
	if _, statErr := os.Stat(filepath.Join(*dir, "node_modules")); statErr == nil {
		generate := exec.Command(commandName("make"), "generate")
		generate.Dir = *dir
		if output, generateErr := generate.CombinedOutput(); generateErr != nil {
			return fmt.Errorf("contract generation failed; domain addition was rolled back: %w: %s", generateErr, strings.TrimSpace(string(output)))
		}
		contractStep = "OpenAPI and TypeScript contracts regenerated"
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	committed = true
	fmt.Printf("added domain %s (%s); DI and routes registered with a fail-closed service; %s; next: specify and implement its authorization, application, audit, and client behavior, then run make verify\n", v.Domain, v.DomainPlural, contractStep)
	return nil
}

func snapshotFiles(paths []string) (map[string]fileSnapshot, error) {
	snapshots := make(map[string]fileSnapshot, len(paths))
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		snapshots[path] = fileSnapshot{body: body, mode: info.Mode()}
	}
	return snapshots, nil
}

func restoreFiles(snapshots map[string]fileSnapshot) error {
	var restoreErr error
	for path, snapshot := range snapshots {
		if err := os.WriteFile(path, snapshot.body, snapshot.mode.Perm()); err != nil {
			restoreErr = errors.Join(restoreErr, fmt.Errorf("restore %s: %w", path, err))
			continue
		}
		if err := os.Chmod(path, snapshot.mode.Perm()); err != nil {
			restoreErr = errors.Join(restoreErr, fmt.Errorf("restore permissions for %s: %w", path, err))
		}
	}
	return restoreErr
}

func validateFieldSpec(spec string) error {
	_, err := parseFieldSpec(spec)
	return err
}

func parseFieldSpec(spec string) ([]fieldDefinition, error) {
	declarations := strings.Split(spec, ",")
	if len(declarations) > 25 {
		return nil, errors.New("a generated domain supports at most 25 fields")
	}
	seen := map[string]struct{}{}
	seenGo := map[string]struct{}{}
	reserved := map[string]struct{}{"id": {}, "owner_user_id": {}, "created_at": {}, "updated_at": {}, "attributes": {}, "table_name": {}}
	definitions := make([]fieldDefinition, 0)
	for _, declaration := range declarations {
		parts := strings.Split(strings.TrimSpace(declaration), ":")
		if len(parts) != 2 || len(parts[0]) > 40 || !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(parts[0]) {
			return nil, fmt.Errorf("invalid field declaration %q; expected name:type", declaration)
		}
		if _, exists := seen[parts[0]]; exists {
			return nil, fmt.Errorf("duplicate field %q", parts[0])
		}
		if _, exists := reserved[parts[0]]; exists {
			return nil, fmt.Errorf("field %q is reserved by the platform", parts[0])
		}
		seen[parts[0]] = struct{}{}
		definition := fieldDefinition{Name: parts[0], GoName: exportedName(parts[0]), Kind: parts[1]}
		if _, exists := seenGo[definition.GoName]; exists {
			return nil, fmt.Errorf("field %q conflicts after Go name conversion", parts[0])
		}
		seenGo[definition.GoName] = struct{}{}
		switch parts[1] {
		case "string":
			definition.GoType, definition.SQLType = "string", "text"
		case "bool":
			definition.GoType, definition.SQLType = "bool", "boolean"
		case "int":
			definition.GoType, definition.SQLType = "int64", "bigint"
		case "float":
			definition.GoType, definition.SQLType = "float64", "double precision"
		case "time":
			definition.GoType, definition.SQLType = "time.Time", "timestamptz"
		default:
			return nil, fmt.Errorf("unsupported field type %q", parts[1])
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func exportedName(name string) string {
	var result strings.Builder
	for _, part := range strings.Split(name, "_") {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]))
		result.WriteString(part[1:])
	}
	return result.String()
}

// exportedDomainName preserves separators so distinct valid domain names stay
// distinct in Huma's global schema registry (for example, foo_1 and foo1).
func exportedDomainName(name string) string {
	parts := strings.Split(name, "_")
	for i := range parts {
		parts[i] = exportedName(parts[i])
	}
	return strings.Join(parts, "_")
}

func withFields(v values, spec string) (values, error) {
	fields, err := parseFieldSpec(spec)
	if err != nil {
		return values{}, err
	}
	var goFields, sqlFields, modelFields, entityToModel, modelToEntity strings.Builder
	var dtoFields, dtoToAttributes, attributesToDTO, updateMap, testFields, testJSON, validation strings.Builder
	hasTime := false
	for _, field := range fields {
		fmt.Fprintf(&goFields, "\t%s %s\n", field.GoName, field.GoType)
		fmt.Fprintf(&modelFields, "\t%s %s\n", field.GoName, field.GoType)
		fmt.Fprintf(&sqlFields, "    \"%s\" %s NOT NULL,\n", field.Name, field.SQLType)
		fmt.Fprintf(&entityToModel, "%s: entity.%s, ", field.GoName, field.GoName)
		fmt.Fprintf(&modelToEntity, "%s: row.%s, ", field.GoName, field.GoName)
		tag := fmt.Sprintf("`json:\"%s\" required:\"true\"`", field.Name)
		if field.Kind == "string" {
			tag = fmt.Sprintf("`json:\"%s\" required:\"true\" minLength:\"1\"`", field.Name)
			fmt.Fprintf(&validation, "\tattributes.%s = strings.TrimSpace(attributes.%s)\n\tif attributes.%s == \"\" { return Entity{}, ErrInvalidFields }\n", field.GoName, field.GoName, field.GoName)
		}
		fmt.Fprintf(&dtoFields, "\t%s %s %s\n", field.GoName, field.GoType, tag)
		fmt.Fprintf(&dtoToAttributes, "%s: input.%s, ", field.GoName, field.GoName)
		fmt.Fprintf(&attributesToDTO, "%s: entity.%s, ", field.GoName, field.GoName)
		fmt.Fprintf(&updateMap, "\"%s\": entity.%s, ", field.Name, field.GoName)
		fmt.Fprintf(&testFields, "%s: %s, ", field.GoName, testValue(field))
		if testJSON.Len() > 0 {
			testJSON.WriteString(",")
		}
		fmt.Fprintf(&testJSON, "%q:%s", field.Name, testJSONValue(field))
		hasTime = hasTime || field.Kind == "time"
	}
	v.DomainGoFields = goFields.String()
	v.DomainSQLFields = sqlFields.String()
	v.DomainModelFields = modelFields.String()
	v.DomainEntityToModelFields = entityToModel.String()
	v.DomainModelToEntityFields = modelToEntity.String()
	v.DomainDTOFields = dtoFields.String()
	v.DomainDTOToAttributesFields = dtoToAttributes.String()
	v.DomainAttributesToDTOFields = attributesToDTO.String()
	v.DomainUpdateMapFields = updateMap.String()
	v.DomainTestFields = testFields.String()
	v.DomainTestJSON = testJSON.String()
	v.DomainStringValidation = validation.String()
	if hasTime {
		v.DomainDTOImports = "import \"time\""
	}
	return v, nil
}

func testValue(field fieldDefinition) string {
	switch field.Kind {
	case "string":
		return `"value"`
	case "bool":
		return "true"
	case "int":
		return "42"
	case "float":
		return "4.2"
	case "time":
		return "now"
	default:
		return "nil"
	}
}

func testJSONValue(field fieldDefinition) string {
	switch field.Kind {
	case "string":
		return `"value"`
	case "bool":
		return "true"
	case "int":
		return "42"
	case "float":
		return "1.5"
	case "time":
		return `"2026-07-18T12:00:00Z"`
	default:
		panic("unsupported field")
	}
}

func installStagedTree(stage, destination string) ([]string, error) {
	var relativeFiles []string
	err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(destination, relative)); err == nil {
			return fmt.Errorf("generated path already exists: %s", relative)
		} else if !os.IsNotExist(err) {
			return err
		}
		relativeFiles = append(relativeFiles, relative)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(relativeFiles)
	installed := make([]string, 0, len(relativeFiles))
	for _, relative := range relativeFiles {
		target := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return installed, err
		}
		if err := os.Rename(filepath.Join(stage, relative), target); err != nil {
			return installed, err
		}
		installed = append(installed, target)
	}
	return installed, nil
}

func rollbackDomainAdd(root string, installed []string, migrationPath string, migration []byte, manifestPath string, manifest []byte, registryPath string, registry []byte) error {
	var rollbackErr error
	if err := os.WriteFile(migrationPath, migration, 0o644); err != nil {
		rollbackErr = errors.Join(rollbackErr, err)
	}
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		rollbackErr = errors.Join(rollbackErr, err)
	}
	if err := os.WriteFile(registryPath, registry, 0o644); err != nil {
		rollbackErr = errors.Join(rollbackErr, err)
	}
	for i := len(installed) - 1; i >= 0; i-- {
		if err := os.Remove(installed[i]); err != nil && !os.IsNotExist(err) {
			rollbackErr = errors.Join(rollbackErr, err)
		}
		for parent := filepath.Dir(installed[i]); parent != root && parent != "."; parent = filepath.Dir(parent) {
			if err := os.Remove(parent); err != nil {
				break
			}
		}
	}
	return rollbackErr
}

func formatGeneratedGo(dir string) error {
	var files []string
	err := filepath.WalkDir(filepath.Join(dir, "apps/api"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	cmd := exec.Command("gofmt", append([]string{"-w"}, files...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("format generated Go: %w: %s", err, output)
	}
	return nil
}

type domainManifest struct {
	Name   string `json:"name"`
	Plural string `json:"plural"`
	Fields string `json:"fields"`
}

type projectManifest struct {
	SchemaVersion    int              `json:"schemaVersion"`
	GeneratorVersion string           `json:"generatorVersion,omitempty"`
	Name             string           `json:"name"`
	Slug             string           `json:"slug"`
	BundlePrefix     string           `json:"bundlePrefix"`
	Module           string           `json:"module"`
	Domains          []domainManifest `json:"domains"`
}

func readProjectManifest(dir string) (projectManifest, error) {
	body, err := os.ReadFile(filepath.Join(dir, ".make-app.json"))
	if err != nil {
		return projectManifest{}, err
	}
	var m projectManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return projectManifest{}, err
	}
	if m.SchemaVersion != templateSchemaVersion {
		if m.SchemaVersion == 3 && templateSchemaVersion == 4 {
			return projectManifest{}, fmt.Errorf("generated project schema version 3 requires the v3-to-v4 upgrade procedure: https://github.com/elsell/make-app/blob/main/docs/upgrading-v3-to-v4.md")
		}
		return projectManifest{}, fmt.Errorf("generated project schema version %d is incompatible with make-app schema version %d; use the matching make-app release", m.SchemaVersion, templateSchemaVersion)
	}
	if m.Name == "" || m.Slug == "" || m.BundlePrefix == "" || m.Module == "" {
		return projectManifest{}, errors.New("generated project manifest is missing application identity")
	}
	return m, nil
}
func writeDomainRegistry(dir string, manifest projectManifest) error {
	path := filepath.Join(dir, "apps/api/internal/generated/domains.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("// Code generated by make-app. DO NOT EDIT.\npackage generated\n\n")
	b.WriteString("import (\n")
	for _, domain := range manifest.Domains {
		if domain.Name == "example" {
			continue
		}
		fmt.Fprintf(&b, "\t%sstore %q\n", domain.Name, manifest.Module+"/apps/api/internal/adapters/gormstore/"+domain.Name)
		fmt.Fprintf(&b, "\t%sapp %q\n", domain.Name, manifest.Module+"/apps/api/internal/app/"+domain.Name)
		fmt.Fprintf(&b, "\t%sroutes %q\n", domain.Name, manifest.Module+"/apps/api/internal/adapters/httpserver/"+domain.Name+"/routes")
	}
	fmt.Fprintf(&b, "\t%q\n", manifest.Module+"/apps/api/internal/ports")
	b.WriteString("\t\"github.com/danielgtaylor/huma/v2\"\n\t\"gorm.io/gorm\"\n\t\"time\"\n)\n\n")
	b.WriteString("var Domains = []string{\n")
	for _, domain := range manifest.Domains {
		if domain.Name == "example" {
			fmt.Fprintf(&b, "\t%q,\n", domain.Name)
		}
	}
	b.WriteString("}\n\n")
	b.WriteString("type Dependencies struct {\n\tDB *gorm.DB\n\tAuth ports.Authenticator\n\tAuthorizer ports.Authorizer\n\tAuthorizationOutbox ports.AuthorizationOutbox\n\tAuthorizationSerializer ports.AuthorizationSerializer\n\tAudits ports.Audits\n\tClock ports.Clock\n\tProbe ports.Probe\n\tNewID func() string\n\tAuthorizationWorker string\n\tAuthorizationLease time.Duration\n\tCursorSigningKey []byte\n}\n\n")
	b.WriteString("func Registrations(dependencies Dependencies) []func(huma.API) {\n")
	b.WriteString("\tregistrations := make([]func(huma.API), 0)")
	for _, domain := range manifest.Domains {
		if domain.Name == "example" {
			continue
		}
		fmt.Fprintf(&b, "\n\tregistrations = append(registrations, func(api huma.API) {\n")
		fmt.Fprintf(&b, "\t\trepository := %sstore.New(dependencies.DB)\n", domain.Name)
		fmt.Fprintf(&b, "\t\tservice := %sapp.New(%sapp.Dependencies{Auth: dependencies.Auth, Authorizer: dependencies.Authorizer, AuthorizationOutbox: dependencies.AuthorizationOutbox, AuthorizationSerializer: dependencies.AuthorizationSerializer, Repository: repository, Audits: dependencies.Audits, Clock: dependencies.Clock, Probe: dependencies.Probe, NewID: dependencies.NewID, AuthorizationWorker: dependencies.AuthorizationWorker, AuthorizationLease: dependencies.AuthorizationLease, CursorSigningKey: dependencies.CursorSigningKey})\n", domain.Name, domain.Name)
		fmt.Fprintf(&b, "\t\t%sroutes.Register(api, service)\n\t})", domain.Name)
	}
	b.WriteString("\n\treturn registrations\n}\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	if output, err := exec.Command("gofmt", "-w", path).CombinedOutput(); err != nil {
		return fmt.Errorf("format domain registry: %w: %s", err, output)
	}
	return nil
}

func writeProjectManifest(dir string, m projectManifest) error {
	sort.Slice(m.Domains, func(i, j int) bool { return m.Domains[i].Name < m.Domains[j].Name })
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err = os.WriteFile(filepath.Join(dir, ".make-app.json"), body, 0o644); err != nil {
		return err
	}
	return nil
}

func domainNames(domains []domainManifest) []string {
	names := make([]string, len(domains))
	for i := range domains {
		names[i] = domains[i].Name
	}
	return names
}

func nextMigrationVersion(dir string) (int, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "apps/api/internal/adapters/dbmigrations"))
	if err != nil {
		return 0, err
	}
	pattern := regexp.MustCompile(`^(\d{6})_.+\.up\.sql$`)
	latest := 0
	for _, entry := range entries {
		match := pattern.FindStringSubmatch(entry.Name())
		if len(match) != 2 {
			continue
		}
		version, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, err
		}
		if version > latest {
			latest = version
		}
	}
	return latest + 1, nil
}

func updateLatestMigrationVersion(dir string, version int) error {
	path := filepath.Join(dir, "apps/api/internal/adapters/dbmigrations/migrations.go")
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`const LatestVersion uint = \d+`)
	updated := pattern.ReplaceAllString(string(body), fmt.Sprintf("const LatestVersion uint = %d", version))
	if updated == string(body) {
		return errors.New("generated migration version constant is missing")
	}
	return os.WriteFile(path, []byte(updated), 0o644)
}

func appValues(name, module string) values {
	slug := slugify(name)
	nativeID := regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(slug, "")
	if nativeID == "" || nativeID[0] < 'a' || nativeID[0] > 'z' {
		nativeID = "app" + nativeID
	}
	envPrefix := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
	if envPrefix == "" || envPrefix[0] < 'A' || envPrefix[0] > 'Z' {
		envPrefix = "APP_" + envPrefix
	}
	return values{Name: name, Slug: slug, NativeID: nativeID, Module: module, EnvPrefix: envPrefix}
}
func validApplicationName(name string) bool {
	return len(name) <= 80 && regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`).MatchString(name)
}
func withDomain(v values, domain string) values {
	v.Domain = strings.ReplaceAll(slugify(domain), "-", "_")
	v.DomainGoName = exportedDomainName(v.Domain)
	v.DomainPlural = pluralize(v.Domain)
	return v
}

func pluralize(word string) string {
	if strings.HasSuffix(word, "y") && len(word) > 1 && !strings.ContainsRune("aeiou", rune(word[len(word)-2])) {
		return strings.TrimSuffix(word, "y") + "ies"
	}
	for _, ending := range []string{"s", "x", "z", "ch", "sh"} {
		if strings.HasSuffix(word, ending) {
			return word + "es"
		}
	}
	return word + "s"
}
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func renderTree(root, dest string, v values) error {
	return fs.WalkDir(templates, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, root+"/")
		rel = replace(rel, v)
		out := filepath.Join(dest, rel)
		if _, err := os.Stat(out); err == nil {
			return fmt.Errorf("refusing to overwrite %s", out)
		}
		body, err := templates.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if strings.HasPrefix(filepath.Base(out), "check-") || (filepath.Ext(out) == ".sh" && filepath.Base(filepath.Dir(out)) == "scripts") {
			mode = 0o755
		}
		return os.WriteFile(out, []byte(replace(string(body), v)), mode)
	})
}
func replace(s string, v values) string {
	r := strings.NewReplacer(
		".go.tmpl", ".go", "GOMOD_TOKEN", "go.mod", "DOTgithub", ".github", "DOTgitignore", ".gitignore", "DOTdockerignore", ".dockerignore", "DOTnpmrc", ".npmrc", "DOTenv", ".env",
		"MIGRATION_TOKEN", v.MigrationVersion, "DOMAIN_TOKEN", v.Domain,
		"__APP_NAME__", v.Name, "__APP_SLUG__", v.Slug, "__APP_NATIVE_ID__", v.NativeID, "__APP_BUNDLE_PREFIX__", v.BundlePrefix, "__MODULE__", v.Module, "__ENV_PREFIX__", v.EnvPrefix,
		"__DOMAIN_PLURAL__", v.DomainPlural, "__DOMAIN__", v.Domain,
		"__DOMAIN_GO_NAME__", v.DomainGoName,
		"__DOMAIN_GO_FIELDS__", v.DomainGoFields, "__DOMAIN_SQL_FIELDS__", v.DomainSQLFields, "__DOMAIN_MODEL_FIELDS__", v.DomainModelFields,
		"__DOMAIN_ENTITY_TO_MODEL_FIELDS__", v.DomainEntityToModelFields, "__DOMAIN_MODEL_TO_ENTITY_FIELDS__", v.DomainModelToEntityFields,
		"__DOMAIN_DTO_FIELDS__", v.DomainDTOFields, "__DOMAIN_DTO_IMPORTS__", v.DomainDTOImports,
		"__DOMAIN_DTO_TO_ATTRIBUTES_FIELDS__", v.DomainDTOToAttributesFields, "__DOMAIN_ATTRIBUTES_TO_DTO_FIELDS__", v.DomainAttributesToDTOFields,
		"__DOMAIN_UPDATE_MAP_FIELDS__", v.DomainUpdateMapFields, "__DOMAIN_TEST_FIELDS__", v.DomainTestFields,
		"__DOMAIN_TEST_JSON__", v.DomainTestJSON,
		"__DOMAIN_STRING_VALIDATION__", v.DomainStringValidation,
	)
	return r.Replace(s)
}

func positionalLast(args []string) []string {
	if len(args) > 1 && !strings.HasPrefix(args[0], "-") {
		return append(append([]string{}, args[1:]...), args[0])
	}
	return args
}
