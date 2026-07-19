#!/usr/bin/env python3
import importlib.util
import json
import shutil
import subprocess
import sys
from pathlib import Path


def load_age_module(root):
    path = root / "scripts" / "check-dependency-age.py"
    spec = importlib.util.spec_from_file_location("dependency_age", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main():
    root = Path.cwd()
    if not shutil.which("curl"):
        print("Ruby vulnerability check requires curl", file=sys.stderr)
        return 1
    age = load_age_module(root)
    gems = set()
    for lockfile in root.glob("**/Gemfile.lock"):
        if "node_modules" not in lockfile.parts:
            gems.update(age.ruby_gem_versions(lockfile))
    ordered = sorted(gems)
    if not ordered:
        print("Ruby vulnerability check passed; no Gemfile.lock graphs found")
        return 0
    payload = json.dumps({"queries": [{"package": {"ecosystem": "RubyGems", "name": name}, "version": version} for name, version, _ in ordered]})
    try:
        result = subprocess.run(
            ["curl", "-fsSL", "--proto", "=https", "--tlsv1.2", "--connect-timeout", "10", "--max-time", "60", "--retry", "3", "--retry-all-errors", "-H", "Content-Type: application/json", "--data-binary", "@-", "https://api.osv.dev/v1/querybatch"],
            input=payload, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=65, check=True,
        )
        results = json.loads(result.stdout).get("results", [])
    except (subprocess.SubprocessError, json.JSONDecodeError) as exc:
        print(f"Ruby vulnerability metadata check failed closed: {exc}", file=sys.stderr)
        return 1
    if len(results) != len(ordered):
        print("Ruby vulnerability metadata response was incomplete", file=sys.stderr)
        return 1
    failures = []
    for (name, version, source), finding in zip(ordered, results):
        for vulnerability in finding.get("vulns", []):
            failures.append(f"{name}@{version} from {source}: {vulnerability.get('id', 'unknown vulnerability')}")
    if failures:
        print("Ruby vulnerability check failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1
    print(f"Ruby vulnerability check passed for {len(ordered)} locked gems")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
