#!/usr/bin/env python3
"""Collect validation reports into docs/validation, with paths sanitised.

Usage:
  python tests/validation/collect.py --ci <downloaded-artifact-dir> \
      --local <local-run-dir> --out docs/validation

What it writes:

  report-native-linux.json          CI, native runtime, Linux runner
  report-native-windows.json        CI, native runtime, Windows runner
  report-container-linux.json       CI, container runtime, one container per workspace
  report-native-windows-local.json  developer machine, native runtime, levels 1-3
  report-native-windows-private-l4.json
                                    developer machine, native runtime, level 4 against
                                    the private business repository

The private report keeps every measurement and every assertion name but drops the
free-form notes, because those quote the project's own failing test names. The
repository is public and the project is not, so the numbers travel and the
internals do not.
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

PERSONAL_PATH = re.compile(
    # A path appears with one backslash in plain text and with up to four in a
    # JSON-encoded repr, so the separator has to tolerate all of them.
    r"(?:[A-Za-z]:\\{1,4}[Uu]sers\\{1,4}[^\\\"']+|/(?:home|Users)/[^/\"']+)"
)


def sanitise(text: str) -> str:
    return PERSONAL_PATH.sub("<home>", text)


def load(path: Path) -> dict:
    return json.loads(sanitise(path.read_text(encoding="utf-8")))


def write(report: dict, target: Path) -> None:
    target.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {target}")


def find(root: Path, name: str) -> Path | None:
    for candidate in root.rglob(name):
        return candidate
    return None


def redact_private(report: dict) -> dict:
    """Keep measurements, drop the notes that quote the project's internals."""
    levels = {}
    for level, data in report.get("levels", {}).items():
        kept_notes = [note for note in data.get("notes", [])
                      if note.startswith(("DEFECT", "RELEASE FAILED", "cleanup:", "SKIPPED"))
                      or note.startswith("spec:")]
        levels[level] = {
            "ok": data.get("ok"),
            "duration_ms": data.get("duration_ms"),
            "checks": [{"name": check["name"], "ok": check["ok"]} for check in data.get("checks", [])],
            "timings_ms": data.get("timings_ms", {}),
            "resources": data.get("resources", {}),
            "notes": kept_notes + [
                "Business test names are not published: this report belongs to a private "
                "repository, so only measurements and assertion outcomes are recorded here.",
            ],
            "error": None,
        }
    environment = report.get("environment", {})
    return {
        "schema": report.get("schema", 1),
        "generated_at": report.get("generated_at"),
        "mode": report.get("mode", "native"),
        "levels": levels,
        "environment": {
            "platform": environment.get("platform"),
            "python": environment.get("python"),
            "cpu_count": environment.get("cpu_count"),
            "berth": "<local build>",
            "image": environment.get("image"),
            "engine": environment.get("engine"),
            "temp_root": "<temp>",
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--ci", required=True, help="directory with downloaded CI artifacts")
    parser.add_argument("--local", required=True, help="directory with local run reports")
    parser.add_argument("--out", required=True)
    args = parser.parse_args()

    ci, local = Path(args.ci), Path(args.local)
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)

    pairs = [
        ("native-Linux.json", "report-native-linux.json"),
        ("native-Windows.json", "report-native-windows.json"),
        ("container-linux.json", "report-container-linux.json"),
    ]
    missing = []
    for source_name, target_name in pairs:
        source = find(ci, source_name)
        if source is None:
            missing.append(source_name)
            continue
        write(load(source), out / target_name)

    local_report = find(local, "native-full.json") if local.exists() else None
    if local_report:
        write(load(local_report), out / "report-native-windows-local.json")
    else:
        missing.append("native-full.json")

    private_report = find(local, "native-l4-canvas.json") if local.exists() else None
    if private_report:
        write(redact_private(load(private_report)), out / "report-native-windows-private-l4.json")
    else:
        missing.append("native-l4-canvas.json")

    if missing:
        print(f"not collected (missing): {', '.join(missing)}", file=sys.stderr)

    reports = sorted(str(path) for path in out.glob("report-*.json"))
    result = subprocess.run([sys.executable, "tests/validation/summarise.py", *reports,
                             "--out", str(out / "report.md")],
                            capture_output=True, text=True, encoding="utf-8", errors="replace")
    sys.stdout.write(result.stdout)
    sys.stderr.write(result.stderr)
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
