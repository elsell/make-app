package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOptionalFeatureInstallationPreservesProductFiles(t *testing.T) {
	root := t.TempDir()
	v := appValues("Feature App", "example.com/features")
	v.BundlePrefix = "com.example"
	if err := renderApplication(root, v, true); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, root)
	if err := addFeature([]string{"notifications", "--dir", root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "apps/api/internal/platform/notifications/provider.go")); err != nil {
		t.Fatal(err)
	}
	after := snapshotTree(t, root)
	for name, content := range before {
		if name != ".make-app.json" && after[name] != content {
			t.Fatalf("changed existing file %s", name)
		}
	}

	if err := addFeature([]string{"notifications", "--dir", root}); err == nil {
		t.Fatal("duplicate feature accepted")
	}
}

func TestUnknownFeatureDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	if err := addFeature([]string{"../../escape", "--dir", root}); err == nil {
		t.Fatal("unsafe feature accepted")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("failed install wrote files")
	}
}

func TestFeatureRefusesConflictAndSymlinkWithoutMutatingManifest(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(map[bool]string{false: "conflict", true: "symlink"}[symlink], func(t *testing.T) {
			root := t.TempDir()
			v := appValues("Safe App", "example.com/safe")
			v.BundlePrefix = "com.example"
			if err := renderApplication(root, v, true); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, "apps/api/internal/platform")
			if symlink {
				if err := os.Symlink(t.TempDir(), target); err != nil {
					t.Fatal(err)
				}
			} else {
				_ = os.MkdirAll(filepath.Join(target, "notifications"), 0755)
				_ = os.WriteFile(filepath.Join(target, "notifications/provider.go"), []byte("product file"), 0644)
			}
			before, err := os.ReadFile(filepath.Join(root, ".make-app.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := addFeature([]string{"notifications", "--dir", root}); err == nil {
				t.Fatal("unsafe installation accepted")
			}
			after, err := os.ReadFile(filepath.Join(root, ".make-app.json"))
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("manifest changed on failure")
			}
			if !symlink {
				body, _ := os.ReadFile(filepath.Join(target, "notifications/provider.go"))
				if string(body) != "product file" {
					t.Fatal("conflicting file changed")
				}
			}
		})
	}
}
