package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

var optionalFeatures = []string{"notifications", "media", "testflight", "selfhost"}

// Features install only new files. They never rewrite a product's composition root.
func addFeature(args []string) (returnErr error) {
	f := flag.NewFlagSet("feature add", flag.ContinueOnError)
	dir := f.String("dir", ".", "generated repository")
	if err := f.Parse(positionalLast(args)); err != nil {
		return err
	}
	if f.NArg() != 1 || !slices.Contains(optionalFeatures, f.Arg(0)) {
		return fmt.Errorf("feature must be one of %v", optionalFeatures)
	}
	root, err := filepath.Abs(*dir)
	if err != nil {
		return err
	}
	// Reject destination symlinks before reading or writing application files.
	if err := rejectDestinationSymlink(root, filepath.Join(root, ".make-app.json"), ".make-app.json"); err != nil {
		return err
	}
	manifest, err := readProjectManifest(root)
	if err != nil {
		return err
	}
	feature := f.Arg(0)
	if slices.Contains(manifest.Features, feature) {
		return errors.New("feature is already installed")
	}
	stage, err := os.MkdirTemp(filepath.Dir(root), ".make-app-feature-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	v := appValues(manifest.Name, manifest.Module)
	v.BundlePrefix = manifest.BundlePrefix
	if err := renderTree("template/features/"+feature, stage, v); err != nil {
		return err
	}
	if err := formatGoTree(stage); err != nil {
		return err
	}
	if err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		return rejectDestinationSymlink(root, filepath.Join(root, rel), rel)
	}); err != nil {
		return err
	}
	oldManifest, err := os.ReadFile(filepath.Join(root, ".make-app.json"))
	if err != nil {
		return err
	}
	installed, err := installStagedTree(stage, root)
	defer func() {
		if returnErr != nil {
			for _, path := range installed {
				if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
					returnErr = errors.Join(returnErr, e)
				}
			}
			returnErr = errors.Join(returnErr, os.WriteFile(filepath.Join(root, ".make-app.json"), oldManifest, 0o644))
		}
	}()
	if err != nil {
		return err
	}
	manifest.Features = append(manifest.Features, feature)
	return writeProjectManifest(root, manifest)
}
