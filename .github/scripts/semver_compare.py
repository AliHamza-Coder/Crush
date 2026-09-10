#!/usr/bin/env python3
"""
Compares two semver-ish version strings (major.minor.patch, extra parts ignored).

Usage:
    python semver_compare.py <current> <latest_released>
    -> prints one of: gt | eq | lt

Used by version-gate.yml to decide whether to tag a new release, skip
(no change), or fail loudly (a version regression). See
references/common-patterns.md rule 3 for what each outcome means.
"""

import re
import sys


def parse(version: str):
    parts = re.findall(r"\d+", version)
    nums = [int(p) for p in parts[:3]]
    while len(nums) < 3:
        nums.append(0)
    return tuple(nums)


def main():
    if len(sys.argv) != 3:
        print("Usage: semver_compare.py <current> <latest>", file=sys.stderr)
        sys.exit(2)

    current = parse(sys.argv[1])
    latest = parse(sys.argv[2])

    if current > latest:
        result = "gt"
    elif current == latest:
        result = "eq"
    else:
        result = "lt"

    print(f"result={result}")


if __name__ == "__main__":
    main()
