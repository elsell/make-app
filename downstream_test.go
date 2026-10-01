package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestGeneratedCIRoutingCoversEveryExpensiveStep(t *testing.T) {
	body, err := os.ReadFile("template/base/DOTgithub/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	body = body[strings.Index(string(body), "jobs:\n")+6:]
	jobs := regexp.MustCompile(`(?m)^  ([a-z][a-z-]*):\n`).FindAllStringSubmatchIndex(string(body), -1)
	for i, m := range jobs {
		name := string(body[m[2]:m[3]])
		end := len(body)
		if i+1 < len(jobs) {
			end = jobs[i+1][0]
		}
		job := string(body[m[0]:end])
		if name == "changes" {
			if strings.Contains(job, "needs.changes") {
				t.Fatal("classifier depends on itself")
			}
			continue
		}
		if !strings.Contains(job, "needs: changes") || !strings.Contains(job, "id: routing") {
			t.Fatalf("%s lacks routing failure gate", name)
		}
		if name == "verify" {
			continue
		}
		steps := strings.Split(job, "      - ")
		for _, step := range steps[2:] {
			if !strings.Contains(step, "if: steps.routing.outcome == 'success' && needs.changes.outputs.") {
				t.Fatalf("ungated %s step: %s", name, step)
			}
		}
	}
}

func TestReleasePackagesOptionalSelfhostWithAttestedDigests(t *testing.T) {
	body, err := os.ReadFile("template/base/DOTgithub/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"hashFiles('scripts/build-selfhost-release.py')", "${{ steps.publish.outputs.api_digest }}", "${{ steps.publish.outputs.web_digest }}", "release-dist/__APP_SLUG__-selfhost.tar.gz.sha256", "--generate-notes \"${assets[@]}\""} {
		if !strings.Contains(string(body), required) {
			t.Fatalf("release lacks %s", required)
		}
	}
}
