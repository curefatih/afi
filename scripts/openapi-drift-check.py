#!/usr/bin/env python3
"""Compare Go ServeMux route registrations to OpenAPI 3 paths.

Usage:
  scripts/openapi-drift-check.py \\
    --go-file internal/controlplane/http.go \\
    --openapi api/openapi/platform.openapi.yaml \\
    --include-prefix /api/v1/platform \\
    --also /healthz \\
    --exclude-prefix /internal/

  scripts/openapi-drift-check.py \\
    --go-file internal/dataplane/pipeline.go \\
    --openapi api/openapi/gateway.openapi.yaml \\
    --exclude-prefix /metrics
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML required: pip install pyyaml", file=sys.stderr)
    sys.exit(2)

HANDLE_RE = re.compile(
    r'''HandleFunc\(\s*"(GET|POST|PUT|PATCH|DELETE)\s+([^"]+)"'''
)


def go_routes(path: Path) -> set[tuple[str, str]]:
    text = path.read_text()
    out: set[tuple[str, str]] = set()
    for m in HANDLE_RE.finditer(text):
        out.add((m.group(1).upper(), m.group(2)))
    return out


def openapi_routes(path: Path) -> set[tuple[str, str]]:
    doc = yaml.safe_load(path.read_text())
    paths = doc.get("paths") or {}
    methods = {"get", "post", "put", "patch", "delete"}
    out: set[tuple[str, str]] = set()
    for p, item in paths.items():
        if not isinstance(item, dict):
            continue
        for method, op in item.items():
            if method.lower() not in methods:
                continue
            if not isinstance(op, dict):
                continue
            out.add((method.upper(), p))
    return out


def filter_routes(
    routes: set[tuple[str, str]],
    *,
    include_prefix: str | None,
    also: list[str],
    exclude_prefix: list[str],
) -> set[tuple[str, str]]:
    also_set = set(also)
    out: set[tuple[str, str]] = set()
    for method, path in routes:
        if any(path.startswith(ex) for ex in exclude_prefix):
            continue
        if path in also_set:
            out.add((method, path))
            continue
        if include_prefix is None or path.startswith(include_prefix):
            out.add((method, path))
    return out


def fmt(routes: set[tuple[str, str]]) -> str:
    return "\n".join(f"  {m} {p}" for m, p in sorted(routes))


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--go-file", type=Path, required=True)
    ap.add_argument("--openapi", type=Path, required=True)
    ap.add_argument("--include-prefix", default=None)
    ap.add_argument("--also", action="append", default=[])
    ap.add_argument("--exclude-prefix", action="append", default=[])
    args = ap.parse_args()

    go = filter_routes(
        go_routes(args.go_file),
        include_prefix=args.include_prefix,
        also=args.also,
        exclude_prefix=args.exclude_prefix,
    )
    spec = filter_routes(
        openapi_routes(args.openapi),
        include_prefix=args.include_prefix,
        also=args.also,
        exclude_prefix=args.exclude_prefix,
    )

    missing_in_spec = go - spec
    extra_in_spec = spec - go
    ok = True
    if missing_in_spec:
        ok = False
        print(f"ERROR: routes in {args.go_file} missing from {args.openapi}:")
        print(fmt(missing_in_spec))
    if extra_in_spec:
        ok = False
        print(f"ERROR: routes in {args.openapi} missing from {args.go_file}:")
        print(fmt(extra_in_spec))
    if ok:
        print(f"OK: {len(go)} routes match ({args.go_file.name} ↔ {args.openapi.name})")
        return 0
    return 1


if __name__ == "__main__":
    sys.exit(main())
