#!/usr/bin/env python3
"""Write gobble's embedded model catalog from a models.dev document.

docs/decisions/0005-PLAN-v1-feature-scope.md, the Amendments entry "F4-2
re-made on go-llmprovider-sdk v1.12.0's model metadata", F4-2a step 1: keep
the nine sections go-llmprovider-sdk reads, and record where and when they
came from. Nothing in gobble fetches the document at run time; this script
is how the snapshot is made, and refreshed.

Writes:
  llm/catalog/data/models.json   the nine sections, compact, keys sorted
  llm/catalog/data/source.json   its source, fetch time, sha256 and counts

Usage:
  scripts/gen_catalog.py --fetch
  scripts/gen_catalog.py --input api.json

Exit status is 1 when the document lacks a section or a section's models.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

URL = "https://models.dev/api.json"
SECTIONS = ("anthropic", "google", "huggingface", "kilo", "openai", "opencode", "opencode-go", "togetherai", "xai")
DATA = Path(__file__).resolve().parent.parent / "llm" / "catalog" / "data"


def fetch() -> bytes:
    """The document at URL."""
    req = urllib.request.Request(URL, headers={"User-Agent": "gobble-gen-catalog"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return resp.read()


def sections(doc: dict) -> dict:
    """The nine sections of doc; exits 1 naming the first one missing."""
    out = {}
    for key in SECTIONS:
        section = doc.get(key)
        if not isinstance(section, dict) or not isinstance(section.get("models"), dict):
            sys.exit(f"gen_catalog: the document has no {key!r} section with models")
        out[key] = section
    return out


def write(path: Path, text: str) -> None:
    """text to path as UTF-8 with LF line ends."""
    path.write_bytes(text.encode("utf-8"))


def main() -> int:
    ap = argparse.ArgumentParser(description="Write gobble's embedded model catalog from a models.dev document.")
    src = ap.add_mutually_exclusive_group(required=True)
    src.add_argument("--fetch", action="store_true", help=f"download {URL}")
    src.add_argument("--input", type=Path, help="a models.dev api.json already downloaded")
    args = ap.parse_args()
    body = fetch() if args.fetch else args.input.read_bytes()
    kept = sections(json.loads(body))
    DATA.mkdir(parents=True, exist_ok=True)
    write(DATA / "models.json", json.dumps(kept, sort_keys=True, separators=(",", ":")) + "\n")
    source = {
        "source": URL if args.fetch else "file",
        "fetched": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%MZ"),
        "sha256": hashlib.sha256(body).hexdigest(),
        "providers": {key: len(kept[key]["models"]) for key in SECTIONS},
    }
    write(DATA / "source.json", json.dumps(source, indent=2) + "\n")
    for key in SECTIONS:
        print(f"{key}: {source['providers'][key]} models")
    return 0


if __name__ == "__main__":
    sys.exit(main())
