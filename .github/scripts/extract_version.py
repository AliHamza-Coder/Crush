#!/usr/bin/env python3
"""
Extracts the app's current version from its manifest file.

Checked in order (edit MANIFEST_CANDIDATES for your repo's actual layout):
  1. crush-core/src/lib.rs  ->  pub const VERSION: &str = "..."  (Rust apps)
  2. Cargo.toml             ->  version = "..."                  (Rust workspace)
  3. package.json           ->  "version"                        (Node apps)

If a root .env defines VITE_APP_VERSION (or another override var), this
script treats a MISMATCH between the override and the manifest as a hard
failure rather than silently picking one. See references/common-patterns.md
rule 1 for why: a silent precedence rule is how a release ships with the
wrong version number.

Usage:
    python extract_version.py
    -> prints the version to stdout, exits 1 on error/mismatch
"""

import json
import os
import re
import sys

MANIFEST_CANDIDATES = [
    "crates/crush-core/src/lib.rs",
    "crush-core/src/lib.rs",
    "Cargo.toml",
    "package.json",
]

ENV_OVERRIDE_FILE = ".env"
ENV_OVERRIDE_VAR = "VITE_APP_VERSION"


def read_rust_version(path):
    """Extract version from Rust source: pub const VERSION: &str = "...";"""
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
    match = re.search(r'pub\s+const\s+VERSION:\s*&str\s*=\s*"([^"]+)"', content)
    if match:
        version = match.group(1)
        # Strip leading 'v' prefix for semver comparison (e.g. "v3.0.0" -> "3.0.0")
        return version.lstrip("v")
    return None


def read_toml_version(path):
    """Extract version from Cargo.toml: version = "..."""
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            match = re.match(r'^version\s*=\s*"([^"]+)"', line.strip())
            if match:
                return match.group(1)
    return None


def read_manifest_version():
    for path in MANIFEST_CANDIDATES:
        if os.path.isfile(path):
            if path.endswith(".rs"):
                version = read_rust_version(path)
            elif path.endswith(".toml"):
                version = read_toml_version(path)
            else:
                with open(path, "r", encoding="utf-8") as f:
                    data = json.load(f)
                version = data.get("version") or data.get("package", {}).get("version")
            if version:
                return version, path
    return None, None


def read_env_override():
    if not os.path.isfile(ENV_OVERRIDE_FILE):
        return None
    with open(ENV_OVERRIDE_FILE, "r", encoding="utf-8") as f:
        for line in f:
            match = re.match(rf"^\s*{ENV_OVERRIDE_VAR}\s*=\s*(.+?)\s*$", line)
            if match:
                return match.group(1).strip('"').strip("'")
    return None


def main():
    manifest_version, manifest_path = read_manifest_version()
    if not manifest_version:
        print(
            "::error::No version field found in any of: "
            + ", ".join(MANIFEST_CANDIDATES),
            file=sys.stderr,
        )
        sys.exit(1)

    override_version = read_env_override()
    if override_version and override_version != manifest_version:
        print(
            f"::error::Version mismatch — {manifest_path} says {manifest_version}, "
            f"but {ENV_OVERRIDE_FILE} sets {ENV_OVERRIDE_VAR}={override_version}. "
            "Fix one of them; refusing to guess which is correct.",
            file=sys.stderr,
        )
        sys.exit(1)

    print(manifest_version)


if __name__ == "__main__":
    main()
