#!/usr/bin/env python3
from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path


SECRET_ASSIGNMENT_RE = re.compile(
    r"(?<![A-Z0-9_])"
    r"(?P<key>"
    r"STRIPE_SECRET_KEY|STRIPE_WEBHOOK_SECRET|GEMINI_API_KEY|OPENAI_API_KEY|"
    r"FIREBASE_PRIVATE_KEY|FIREBASE_CREDENTIALS|AD_REWARD_HMAC_SECRET|CSRF_SECRET|"
    r"FIREBASE_APP_CHECK_DEBUG_TOKEN|APP_CHECK_TOKEN|APP_CHECK_SECRET|APP_CHECK_KEY|"
    r"VITE_STRIPE_SECRET|VITE_GEMINI_API_KEY|VITE_OPENAI_API_KEY|VITE_FIREBASE_PRIVATE_KEY"
    r")"
    r"\s*[:=]\s*[\"']?(?P<value>[^\"',\s#]+)"
)

HIGH_CONFIDENCE_PATTERNS = (
    ("stripe_live_secret", re.compile(r"sk_live_[A-Za-z0-9]{12,}")),
    ("stripe_webhook_secret", re.compile(r"whsec_[A-Za-z0-9]{12,}")),
    ("private_key_block", re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----")),
)

PLACEHOLDER_MARKERS = (
    "your",
    "dummy",
    "example",
    "placeholder",
    "change_me",
    "changeme",
    "local",
    "xxxx",
)

SKIP_PARTS = {
    ".git",
    ".gocache",
    ".expo",
    "node_modules",
    "dist",
    "build",
    "coverage",
    "Pods",
}


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def tracked_files(root: Path) -> list[Path]:
    result = subprocess.run(
        ["git", "-C", str(root), "ls-files"],
        check=True,
        text=True,
        capture_output=True,
    )
    return [root / line for line in result.stdout.splitlines() if line]


def should_skip(path: Path) -> bool:
    return any(part in SKIP_PARTS for part in path.parts)


def is_placeholder(value: str) -> bool:
    lowered = value.strip("\"',").lower()
    if lowered == "" or lowered.endswith(":latest"):
        return True
    return any(marker in lowered for marker in PLACEHOLDER_MARKERS)


def read_text(path: Path) -> str | None:
    try:
        raw = path.read_bytes()
    except OSError:
        return None
    if b"\x00" in raw:
        return None
    try:
        return raw.decode("utf-8")
    except UnicodeDecodeError:
        return None


def scan_file(path: Path) -> list[str]:
    text = read_text(path)
    if text is None:
        return []

    findings: list[str] = []
    for line_number, line in enumerate(text.splitlines(), start=1):
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        for name, pattern in HIGH_CONFIDENCE_PATTERNS:
            if pattern.search(line) and not is_placeholder(line):
                findings.append(f"{path}:{line_number}: {name}")

        match = SECRET_ASSIGNMENT_RE.search(line)
        if match and not is_placeholder(match.group("value")):
            findings.append(f"{path}:{line_number}: {match.group('key')}")

    return findings


def main() -> int:
    root = repo_root()
    if len(sys.argv) > 1:
        files = [Path(arg) for arg in sys.argv[1:]]
    else:
        files = tracked_files(root)

    findings: list[str] = []
    for path in files:
        candidate = path if path.is_absolute() else root / path
        if should_skip(candidate) or not candidate.is_file():
            continue
        findings.extend(scan_file(candidate))

    if findings:
        print("Potential secret material found:", file=sys.stderr)
        for finding in findings:
            print(f"  {finding}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
