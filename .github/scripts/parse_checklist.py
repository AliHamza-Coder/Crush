#!/usr/bin/env python3
"""
Parses the TOPMOST "### v..." entry under "## History" in RELEASE_CHECKLIST.md.

This file is the source of truth for the current release's version and
changelog body — see references/common-patterns.md rule 9. Only the first
"### v..." block is ever read; everything below the second such heading is
human-readable history that CI ignores.

Usage:
    python parse_checklist.py --expect-version 1.2.1 [--write-body RELEASE_BODY.txt]

Exits non-zero (with a clear ::error:: message) if:
  - the file or the History section is missing
  - no "### v..." entry is found
  - --expect-version is given and doesn't match the top entry's version
    (this is what stops a release from shipping with a stale or empty changelog)
"""

import argparse
import re
import sys

CHECKLIST_PATH = ".github/RELEASE_CHECKLIST.md"
ENTRY_HEADING_RE = re.compile(r"^### v(\S+)\s*(?:—|-)?\s*(.*)$", re.MULTILINE)


def parse_top_entry(text: str):
    matches = list(ENTRY_HEADING_RE.finditer(text))
    if not matches:
        return None
    first = matches[0]
    version = first.group(1)
    start = first.end()
    end = matches[1].start() if len(matches) > 1 else len(text)
    body = text[start:end].strip("\n")
    return version, body


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--expect-version", required=True)
    parser.add_argument("--write-body", default=None)
    args = parser.parse_args()

    try:
        with open(CHECKLIST_PATH, "r", encoding="utf-8") as f:
            text = f.read()
    except FileNotFoundError:
        print(f"::error::{CHECKLIST_PATH} not found.", file=sys.stderr)
        sys.exit(1)

    if "## History" not in text:
        print(f"::error::{CHECKLIST_PATH} has no '## History' section.", file=sys.stderr)
        sys.exit(1)

    parsed = parse_top_entry(text)
    if not parsed:
        print(
            f"::error::No '### v...' entry found under History in {CHECKLIST_PATH}. "
            "Add a dated entry for this release before pushing.",
            file=sys.stderr,
        )
        sys.exit(1)

    entry_version, body = parsed

    if entry_version != args.expect_version:
        print(
            f"::error::Checklist's top entry is v{entry_version}, but the app "
            f"manifest version is {args.expect_version}. Update the checklist "
            "(and add a real changelog entry) before this can release.",
            file=sys.stderr,
        )
        sys.exit(1)

    if not body or all(line.strip() in ("", "-") for line in body.splitlines()):
        print(
            f"::error::Checklist entry for v{entry_version} has no changelog "
            "content. Fill in real bullets before pushing.",
            file=sys.stderr,
        )
        sys.exit(1)

    if args.write_body:
        with open(args.write_body, "w", encoding="utf-8") as f:
            f.write(body + "\n")

    print(f"version={entry_version}")


if __name__ == "__main__":
    main()
