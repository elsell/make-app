package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed template
var templates embed.FS

type values struct{ Name, Slug, Module, EnvPrefix, Domain, DomainPlural string }

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
	return os.WriteFile(hook, []byte("#!/usr/bin/env sh\nset -eu\nmake check\n"), 0o755)
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
	v := withDomain(appValues(filepath.Base(*dir), fields[1]), f.Arg(0))
	target := filepath.Join(*dir, "apps/api/internal/domain", v.Domain)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("domain %s already exists", v.Domain)
	}
	return renderTree("template/domain", *dir, v)
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
		if strings.HasPrefix(filepath.Base(out), "check-") {
			mode = 0o755
		}
		return os.WriteFile(out, []byte(replace(string(body), v)), mode)
	})
}
func replace(s string, v values) string {
	r := strings.NewReplacer(".go.tmpl", ".go", "GOMOD_TOKEN", "go.mod", "DOTgithub", ".github", "DOTgitignore", ".gitignore", "DOTenv", ".env", "DOMAIN_TOKEN", v.Domain, "__APP_NAME__", v.Name, "__APP_SLUG__", v.Slug, "__MODULE__", v.Module, "__ENV_PREFIX__", v.EnvPrefix, "__DOMAIN_PLURAL__", v.DomainPlural, "__DOMAIN__", v.Domain)
	return r.Replace(s)
}

func positionalLast(args []string) []string {
	if len(args) > 1 && !strings.HasPrefix(args[0], "-") {
		return append(append([]string{}, args[1:]...), args[0])
	}
	return args
}
