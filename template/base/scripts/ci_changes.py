#!/usr/bin/env python3
"""Classify changed repository paths into expensive CI gates.

The classifier deliberately fails safe: missing history, malformed paths, empty
diffs, and paths outside the reviewed dependency map select every expensive
gate. The normal verification job is not routed and always runs.
"""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from pathlib import PurePosixPath
import re
import subprocess
import sys
from typing import Iterable


_SHA = re.compile(r"^[0-9a-f]{40}$")


@dataclass(frozen=True)
class Plan:
    acceptance: bool = False
    api_image: bool = False
    web_image: bool = False
    native: bool = False

    def union(self, other: "Plan") -> "Plan":
        return Plan(
            acceptance=self.acceptance or other.acceptance,
            api_image=self.api_image or other.api_image,
            web_image=self.web_image or other.web_image,
            native=self.native or other.native,
        )


ALL = Plan(acceptance=True, api_image=True, web_image=True, native=True)


def _starts_with(path: str, prefix: str) -> bool:
    return path == prefix.rstrip("/") or path.startswith(prefix)


def _classify_path(path: str) -> Plan | None:
    """Return a gate plan, or None when the path is not explicitly mapped."""
    if path in {"README.md", "LICENSE", "SECURITY.md", "AGENTS.md", ".gitignore"}:
        return Plan()
    if any(_starts_with(path, prefix) for prefix in ("docs/", "specs/", "planning/", ".github/ISSUE_TEMPLATE/")):
        return Plan()
    if path in {"dependency-age-allowlist.json", "lefthook.yml", ".make-app.json"}:
        return Plan()
    if path in {"scripts/ci_changes.py", "scripts/test_ci_changes.py"}:
        return Plan()
    if path in {".github/workflows/ci.yml", "Makefile"}:
        return ALL
    if _starts_with(path, "apps/api/"):
        return Plan(acceptance=True, api_image=True)
    if _starts_with(path, "apps/web/"):
        return Plan(acceptance=True, web_image=True)
    if _starts_with(path, "apps/mobile/"):
        return Plan(native=True)
    if any(_starts_with(path, prefix) for prefix in ("packages/api-client/", "packages/client-core/", "packages/i18n/")):
        return Plan(acceptance=True, web_image=True, native=True)

    if path in {"package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc"}:
        return Plan(acceptance=True, web_image=True, native=True)
    if path in {"go.work", "go.work.sum"}:
        return Plan(acceptance=True, api_image=True)
    if path == ".dockerignore":
        return Plan(acceptance=True, api_image=True, web_image=True)
    if path in {"compose.yaml", ".env.example"} or _starts_with(path, "deploy/"):
        return Plan(acceptance=True)

    if path in {
        "scripts/live-acceptance.sh",
        "scripts/scalar-browser-acceptance.mjs",
        "scripts/scalar-try-retry.mjs",
        "scripts/scalar-try-retry.test.mjs",
        "scripts/test-scalar-browser-acceptance.sh",
        "scripts/web-browser-acceptance.mjs",
        "scripts/web-browser-navigation.mjs",
        "scripts/web-browser-navigation.test.mjs",
        "scripts/test-web-browser-acceptance.sh",
    }:
        return Plan(acceptance=True)
    if path in {
        "scripts/build-ios-simulator.sh",
        "scripts/run-eas.mjs",
        "scripts/run-eas.test.mjs",
        "scripts/validate-expo-native-set.mjs",
        "scripts/validate-mobile-config.mjs",
        "scripts/validate-mobile-release-env.mjs",
    }:
        return Plan(native=True)

    return None


def _valid_path(path: str) -> bool:
    if not path or "\\" in path or path.startswith("/"):
        return False
    pure = PurePosixPath(path)
    return ".." not in pure.parts and "." not in pure.parts


def classify(paths: Iterable[str], *, force_full: bool = False) -> Plan:
    if force_full:
        return ALL
    materialized = list(paths)
    if not materialized:
        return ALL
    result = Plan()
    for path in materialized:
        if not _valid_path(path):
            return ALL
        contribution = _classify_path(path)
        if contribution is None:
            return ALL
        result = result.union(contribution)
    return result


def select_diff_range(event_name: str, ref: str, before: str, base_sha: str, head_sha: str) -> str | None:
    """Select the auditable git range; None means run the full suite."""
    if event_name == "workflow_dispatch" or (event_name == "push" and ref == "refs/heads/main"):
        return None
    if not _SHA.fullmatch(head_sha):
        return None
    if event_name == "pull_request" and _SHA.fullmatch(base_sha):
        return f"{base_sha}...{head_sha}"
    if event_name == "push" and _SHA.fullmatch(before) and before != "0" * 40:
        return f"{before}..{head_sha}"
    return None


def changed_paths(diff_range: str) -> list[str] | None:
    try:
        completed = subprocess.run(
            ["git", "diff", "--name-only", "-z", diff_range, "--"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
    except (OSError, subprocess.CalledProcessError) as error:
        print(f"CI change classification fell back to all gates: {error}", file=sys.stderr)
        return None
    return [part.decode("utf-8", errors="strict") for part in completed.stdout.split(b"\0") if part]


def _write_outputs(output_path: str, plan: Plan, reason: str) -> None:
    values = {
        "acceptance": plan.acceptance,
        "api_image": plan.api_image,
        "web_image": plan.web_image,
        "native": plan.native,
    }
    with open(output_path, "a", encoding="utf-8") as output:
        for name, enabled in values.items():
            output.write(f"{name}={'true' if enabled else 'false'}\n")
        output.write(f"reason={reason}\n")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--event-name", required=True)
    parser.add_argument("--ref", required=True)
    parser.add_argument("--before", default="")
    parser.add_argument("--base-sha", default="")
    parser.add_argument("--head-sha", required=True)
    parser.add_argument("--github-output", required=True)
    arguments = parser.parse_args()

    diff_range = select_diff_range(
        arguments.event_name,
        arguments.ref,
        arguments.before,
        arguments.base_sha,
        arguments.head_sha,
    )
    if diff_range is None:
        _write_outputs(arguments.github_output, ALL, "full-suite event or unavailable history")
        return 0

    try:
        paths = changed_paths(diff_range)
    except UnicodeDecodeError as error:
        print(f"CI change classification fell back to all gates: {error}", file=sys.stderr)
        paths = None
    plan = ALL if paths is None else classify(paths)
    reason = "classification fallback" if paths is None else f"classified {len(paths)} changed path(s)"
    _write_outputs(arguments.github_output, plan, reason)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
