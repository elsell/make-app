package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed template
var templates embed.FS

type values struct{ Name, Slug, Module, EnvPrefix, Domain, DomainPlural, MigrationVersion string }

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "make-app:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: make-app new NAME --module MODULE [--output DIR] | make-app domain add NAME [--dir DIR]")
	}
	switch args[0] {
	case "new":
		return newApp(args[1:])
	case "domain":
		if len(args) > 1 && args[1] == "add" {
			return addDomain(args[2:])
		}
	}
	return fmt.Errorf("unknown command %q", strings.Join(args, " "))
}

func newApp(args []string) error {
	args = positionalLast(args)
	f := flag.NewFlagSet("new", flag.ContinueOnError)
	module := f.String("module", "", "Go module")
	output := f.String("output", "", "output directory")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 1 || *module == "" {
		return errors.New("new requires NAME and --module")
	}
	v := appValues(f.Arg(0), *module)
	dest := *output
	if dest == "" {
		dest = v.Slug
	}
	if v.Slug == "" || strings.ContainsAny(*module, " \\") || !strings.Contains(*module, ".") {
		return errors.New("NAME and --module must be valid project identifiers")
	}
	if entries, err := os.ReadDir(dest); err == nil && len(entries) > 0 {
		return fmt.Errorf("destination %s is not empty", dest)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := renderTree("template/base", dest, v); err != nil {
		return err
	}
	if err := renderTree("template/domain", dest, withDomain(v, "example")); err != nil {
		return err
	}
	if err := writeDomainRegistry(dest, []string{"example"}); err != nil {
		return err
	}
	if err := formatGeneratedGo(dest); err != nil {
		return err
	}
	if err := initializeGit(dest); err != nil {
		return err
	}
	fmt.Printf("generated %s; run: cd %s && make bootstrap\n", v.Name, dest)
	return nil
}
func initializeGit(dest string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dest
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("initialize git: %w: %s", err, output)
	}
	hook := filepath.Join(dest, ".git", "hooks", "pre-commit")
	return os.WriteFile(hook, []byte("#!/usr/bin/env sh\nset -eu\nmake verify\n"), 0o755)
}

func addDomain(args []string) error {
	args = positionalLast(args)
	f := flag.NewFlagSet("domain add", flag.ContinueOnError)
	dir := f.String("dir", ".", "application directory")
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
	v := withDomain(appValues(filepath.Base(*dir), rootModule), f.Arg(0))
	target := filepath.Join(*dir, "apps/api/internal/domain", v.Domain)
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(v.Domain) {
		return errors.New("domain name must begin with a letter and contain only letters, digits, or underscores")
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
	domains, err := readDomains(*dir)
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
		rollbackDomainAdd(*dir, installed, migrationPath, originalMigration, manifestPath, originalManifest)
		return err
	}
	committed := false
	defer func() {
		if !committed {
			rollbackDomainAdd(*dir, installed, migrationPath, originalMigration, manifestPath, originalManifest)
		}
	}()
	if err := updateLatestMigrationVersion(*dir, migrationVersion); err != nil {
		return err
	}
	domains = append(domains, v.Domain)
	if err = writeProjectManifest(*dir, domains); err != nil {
		return err
	}
	committed = true
	return nil
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

func rollbackDomainAdd(root string, installed []string, migrationPath string, migration []byte, manifestPath string, manifest []byte) {
	_ = os.WriteFile(migrationPath, migration, 0o644)
	_ = os.WriteFile(manifestPath, manifest, 0o644)
	for i := len(installed) - 1; i >= 0; i-- {
		_ = os.Remove(installed[i])
		for parent := filepath.Dir(installed[i]); parent != root && parent != "."; parent = filepath.Dir(parent) {
			if err := os.Remove(parent); err != nil {
				break
			}
		}
	}
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

type projectManifest struct {
	Domains []string `json:"domains"`
}

func readDomains(dir string) ([]string, error) {
	body, err := os.ReadFile(filepath.Join(dir, ".make-app.json"))
	if err != nil {
		return nil, err
	}
	var m projectManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m.Domains, nil
}
func writeDomainRegistry(dir string, domains []string) error {
	if err := writeProjectManifest(dir, domains); err != nil {
		return err
	}
	path := filepath.Join(dir, "apps/api/internal/generated/domains.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("// Code generated by make-app. DO NOT EDIT.\npackage generated\n\nvar Domains = []string{\n")
	for _, d := range domains {
		fmt.Fprintf(&b, "\t%q,\n", d)
	}
	b.WriteString("}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeProjectManifest(dir string, domains []string) error {
	sort.Strings(domains)
	m := projectManifest{Domains: domains}
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
	return values{Name: name, Slug: slug, Module: module, EnvPrefix: strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))}
}
func withDomain(v values, domain string) values {
	v.Domain = strings.ReplaceAll(slugify(domain), "-", "_")
	v.DomainPlural = v.Domain + "s"
	return v
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
	r := strings.NewReplacer(".go.tmpl", ".go", "GOMOD_TOKEN", "go.mod", "DOTgithub", ".github", "DOTgitignore", ".gitignore", "DOTdockerignore", ".dockerignore", "DOTnpmrc", ".npmrc", "DOTenv", ".env", "MIGRATION_TOKEN", v.MigrationVersion, "DOMAIN_TOKEN", v.Domain, "__APP_NAME__", v.Name, "__APP_SLUG__", v.Slug, "__MODULE__", v.Module, "__ENV_PREFIX__", v.EnvPrefix, "__DOMAIN_PLURAL__", v.DomainPlural, "__DOMAIN__", v.Domain)
	return r.Replace(s)
}

func positionalLast(args []string) []string {
	if len(args) > 1 && !strings.HasPrefix(args[0], "-") {
		return append(append([]string{}, args[1:]...), args[0])
	}
	return args
}
