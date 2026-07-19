#!/usr/bin/env python3
import datetime as dt
import importlib.util
import json
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MODULE_PATH = ROOT / "scripts" / "check-dependency-age.py"
UTC = dt.timezone.utc


def load_module():
    spec = importlib.util.spec_from_file_location("check_dependency_age", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main():
    module = load_module()
    with tempfile.TemporaryDirectory() as tmp:
        lock = Path(tmp) / "Gemfile.lock"
        lock.write_text("GEM\n  specs:\n    cocoapods (1.16.2)\n      addressable (~> 2.8)\n    addressable (2.9.0)\n\nPLATFORMS\n  ruby\n", encoding="utf-8")
        assert module.ruby_gem_versions(lock) == {("cocoapods", "1.16.2", str(lock)), ("addressable", "2.9.0", str(lock))}
    original_which = module.shutil.which
    original_run = module.subprocess.run
    captured = {}
    try:
        module.shutil.which = lambda name: "/usr/bin/curl" if name == "curl" else None
        def fake_run(command, **kwargs):
            captured["command"] = command
            captured["kwargs"] = kwargs
            return subprocess.CompletedProcess(command, 0, stdout="{}", stderr="")
        module.subprocess.run = fake_run
        module.fetch_json("https://registry.example/package")
    finally:
        module.shutil.which = original_which
        module.subprocess.run = original_run
    command = captured["command"]
    assert command[command.index("--connect-timeout") + 1] == "10"
    assert command[command.index("--max-time") + 1] == "30"
    assert captured["kwargs"]["timeout"] == 35
    module.subprocess.run = fake_run
    try:
        with tempfile.TemporaryDirectory() as tmp:
            module.go_modules(Path(tmp) / "go.mod")
    finally:
        module.subprocess.run = original_run
    assert captured["kwargs"]["timeout"] == 60
    timeout = subprocess.TimeoutExpired(["go", "list"], 60)
    assert "timed out" in module.subprocess_failure(timeout)
    module.shutil.which = lambda _name: None
    try:
        try:
            module.fetch_json("https://registry.example/package")
            raise AssertionError("missing curl did not fail closed")
        except RuntimeError:
            pass
    finally:
        module.shutil.which = original_which
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "dependency-age-allowlist.json").write_text(
            json.dumps(
                [
                    {
                        "kind": "npm",
                        "name": "expo-auth-session",
                        "version": "55.0.17",
                        "reason": "Required for mobile OIDC.",
                        "compensatingVerification": "Mobile tests and typecheck pass.",
                    }
                ]
            ),
            encoding="utf-8",
        )

        allowlist, failures = module.load_allowlist(root)
        assert not failures, failures
        assert ("npm", "expo-auth-session", "55.0.17") in allowlist

    cutoff = dt.datetime(2026, 6, 20, tzinfo=UTC)
    published_at = dt.datetime(2026, 6, 25, tzinfo=UTC)
    allowed = {("npm", "expo-auth-session", "55.0.17")}
    assert module.check_age(
        "npm",
        "expo-auth-session",
        "55.0.17",
        published_at,
        "pnpm-lock.yaml",
        cutoff,
        allowed,
    ) is None
    assert module.check_age(
        "npm",
        "other-package",
        "1.0.0",
        published_at,
        "pnpm-lock.yaml",
        cutoff,
        allowed,
    )

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "dependency-age-allowlist.json").write_text(
            json.dumps([{"kind": "npm", "name": "expo-auth-session"}]),
            encoding="utf-8",
        )

        _, failures = module.load_allowlist(root)
        assert failures, "expected malformed allowlist entry to fail closed"

    print("dependency age allowlist tests passed")


if __name__ == "__main__":
    main()
