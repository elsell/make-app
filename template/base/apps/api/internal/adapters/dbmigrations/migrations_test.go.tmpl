package dbmigrations

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestLatestVersionMatchesEmbeddedMigrations(t *testing.T) {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)
	var latest uint
	for _, entry := range entries {
		match := pattern.FindStringSubmatch(entry.Name())
		if len(match) != 2 {
			continue
		}
		version, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if uint(version) > latest {
			latest = uint(version)
		}
	}
	if latest == 0 {
		t.Fatal("no embedded up migrations found")
	}
	if LatestVersion != latest {
		t.Fatalf("LatestVersion=%d, embedded migration version=%d", LatestVersion, latest)
	}
}

func TestPriorReleaseMigrationChecksums(t *testing.T) {
	lines := strings.Split(strings.TrimSpace(string(priorReleaseChecksums)), "\n")
	want := map[string]string{}
	previous := ""
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("invalid prior-release checksum entry %q", line)
		}
		checksum, name := fields[0], fields[1]
		if name <= previous {
			t.Fatalf("prior-release checksum inventory is not strictly ordered at %q", name)
		}
		if _, err := hex.DecodeString(checksum); err != nil || len(checksum) != sha256.Size*2 {
			t.Fatalf("invalid SHA-256 checksum for %s", name)
		}
		if _, exists := want[name]; exists {
			t.Fatalf("duplicate prior-release checksum entry for %s", name)
		}
		want[name] = checksum
		previous = name
	}

	pattern := regexp.MustCompile(`^(\d+)_.*\.(up|down)\.sql$`)
	seen := map[string]bool{}
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		match := pattern.FindStringSubmatch(entry.Name())
		if len(match) != 3 {
			continue
		}
		version, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if uint(version) > PriorReleaseVersion {
			continue
		}
		body, err := fs.ReadFile(Files, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != want[entry.Name()] {
			t.Fatalf("released migration %s changed: got %s, want %s", entry.Name(), got, want[entry.Name()])
		}
		seen[entry.Name()] = true
	}
	for version := uint(1); version <= PriorReleaseVersion; version++ {
		prefix := strconv.FormatUint(uint64(version), 10)
		prefix = strings.Repeat("0", 6-len(prefix)) + prefix + "_"
		for _, direction := range []string{"up", "down"} {
			found := false
			for name := range seen {
				if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, "."+direction+".sql") {
					found = true
				}
			}
			if !found {
				t.Fatalf("released migration %d is missing its %s direction", version, direction)
			}
		}
	}
	if len(want) != len(seen) {
		t.Fatalf("prior-release checksum inventory has %d entries for %d migration files", len(want), len(seen))
	}
}
