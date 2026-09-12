#!/usr/bin/env python3
"""Level 4: a real external project, described by a JSON spec.

The same code path validates the private business repository on a developer
machine and a public repository in CI, because everything project-specific
lives in the spec file:

  {
    "name": "canvas",
    "repo": "https://github.com/qbox/canvas",
    "setup": ["pnpm --dir frontend install --frozen-lockfile"],
    "checks": [
      {"name": "go build ./...", "cmd": ["go", "build", "./..."]},
      {"name": "frontend typecheck", "cmd": ["npm", "--prefix", "frontend", "run", "check"]},
      {"name": "go test ./...",  "cmd": ["go", "test", "./..."], "allow_failure": true}
    ],
    "parallel_workspaces": 2
  }

`setup` becomes the workspace setup hook, so a heavy real-world install happens
where berth intends it to happen: once per workspace, inside the workspace.
`checks` are business commands run through `berth run`, and `allow_failure`
records a command whose failure is an environment gap (a missing database, for
example) instead of a berth defect.

Private repositories are cloned with the credentials already configured for the
`gh` CLI; if the clone fails the level reports a skip with the exact error rather
than pretending to have passed.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import time
from pathlib import Path

from harness import rss_kb, dir_kb


def now_ms() -> float:
    return time.perf_counter() * 1000


def clone(spec: dict, target: Path, env: dict) -> tuple[bool, str]:
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists():
        # A workspace from a previous run may keep the checkout busy; a fresh
        # clone is cheaper than debugging file locks.
        shutil.rmtree(target, ignore_errors=True)
    args = ["git", "clone", "--depth", "1"]
    if spec.get("branch"):
        args += ["--branch", spec["branch"]]
    args += [spec["repo"], str(target)]
    result = subprocess.run(args, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=1800, env=env)
    if result.returncode == 0:
        return True, ""
    # Private repository: fall back to the authenticated gh client.
    gh = subprocess.run(["gh", "repo", "clone", spec["repo"], str(target), "--", "--depth", "1"],
                        capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=1800, env=env)
    if gh.returncode == 0:
        return True, "cloned with gh credentials"
    return False, (result.stderr + gh.stderr).strip()[:600]


def write_config(spec: dict, repo: Path, harness) -> None:
    ports = spec.get("ports", [])
    runtime = ""
    if harness.mode == "container":
        image = spec.get("container_image") or harness.args.image
        runtime = f"""runtime:
  backend: container
  engine: {harness.args.engine}
  image: {image}
"""
    hooks = ""
    if spec.get("setup"):
        hooks = "hooks:\n  setup:\n" + "".join(f"    - {c}\n" for c in spec["setup"])
    (repo / "berth.yaml").write_text(
        f"version: 1\nbase: {spec.get('base', 'main')}\n{runtime}"
        f"ports: [{', '.join(ports)}]\n{hooks}processes: {{}}\n"
    )


def run_level4(harness, checks) -> dict:
    spec = json.loads(Path(harness.args.l4_json).read_text())
    timings: dict[str, float] = harness.partial if hasattr(harness, "partial") else {}
    notes: list[str] = [f"spec: {harness.args.l4_json}"]
    repo = harness.root / f"l4-{spec['name']}-{harness.token}"

    started = now_ms()
    ok, detail = clone(spec, repo, harness.env)
    timings["clone_ms"] = round(now_ms() - started, 2)
    if not ok:
        checks.that(f"external project {spec['name']} cloned", False, detail)
        notes.append(f"SKIPPED: clone failed: {detail}")
        return {"checks": checks.rows, "timings_ms": timings, "resources": {}, "notes": notes,
                "skipped": True}
    if detail:
        notes.append(detail)
    checks.that(f"external project {spec['name']} cloned", True, f"{spec['repo']} -> {repo}")
    harness.git("config", "user.name", "berth validation", cwd=repo)
    harness.git("config", "user.email", "validation@example.invalid", cwd=repo)

    write_config(spec, repo, harness)
    harness.git("add", "berth.yaml", cwd=repo)
    harness.git("commit", "-m", "berth validation config", cwd=repo)

    slugs = [f"real{i}" for i in range(1, max(1, int(spec.get("parallel_workspaces", 1))) + 1)]
    created = []
    for slug in slugs:
        started = now_ms()
        workspace = harness.new(repo, slug, "--json")["value"]
        timings[f"new_{slug}_ms"] = round(now_ms() - started, 2)
        created.append(workspace)
    checks.that("every workspace checkout contains the project",
                all((Path(w["path"]) / ".git").exists() for w in created),
                f"{[w['path'] for w in created]}")

    primary = Path(created[0]["path"])
    resources = {
        "workspace_disk_mb": dir_kb(primary) // 1024,
        "project_disk_mb": dir_kb(repo) // 1024,
        "container_mem_bytes": harness.container_mem_bytes(created[0]["path"]),
    }

    # The business commands, run through berth run so they see the workspace
    # runtime, environment and data directory.
    for check in spec["checks"]:
        started = now_ms()
        result = harness.cli("run", "--", *check["cmd"], cwd=primary, timeout=harness.args.l4_timeout)
        elapsed = round(now_ms() - started, 2)
        timings[f"run_{check['name']}_ms"] = elapsed
        output = (result.stdout + result.stderr).strip()
        lines = [line for line in output.splitlines() if line.strip()]
        safe = "".join(c if c.isalnum() else "-" for c in check["name"]).strip("-")
        log = harness.root / "logs" / f"{spec['name']}-{safe}.log"
        log.parent.mkdir(parents=True, exist_ok=True)
        log.write_text(output, encoding="utf-8")
        notes.append(f"{check['name']}: exit {result.returncode} in {elapsed} ms | "
                     f"full output: {log} | tail: " + " | ".join(lines[-3:]))
        if result.returncode != 0:
            interesting = [line for line in lines
                           if any(word in line for word in ("FAIL", "Error", "error:", "panic",
                                                            "cannot", "expected", "✗"))][:10]
            notes.append(f"{check['name']} failure lines: " + " | ".join(interesting or lines[:10]))
        if check.get("allow_failure"):
            checks.that(f"{check['name']} runs inside the workspace (informational)",
                        True, f"exit {result.returncode} in {elapsed} ms")
        else:
            checks.that(f"{check['name']} succeeds inside the workspace",
                        result.returncode == 0, "\n".join(tail) or f"exit {result.returncode}")

    # Parallel workspaces must each do real work without interfering.
    if len(created) > 1 and spec["checks"]:
        command = spec["checks"][0]["cmd"]
        started = now_ms()
        outcomes = []
        for workspace in created:
            outcomes.append(harness.cli("run", "--", *command, cwd=Path(workspace["path"]),
                                        timeout=harness.args.l4_timeout).returncode)
        timings["parallel_all_workspaces_first_check_ms"] = round(now_ms() - started, 2)
        checks.equal(f"the same business command succeeds in all {len(created)} workspaces",
                     outcomes, [0] * len(created))

    pids, count = harness.supervised(created[0]["slug"], repo)
    resources["supervised_processes"] = count
    resources["supervised_rss_kb"] = rss_kb(pids)

    released, release_errors = [], []
    for workspace in created:
        harness.release(repo, workspace["slug"])
        done = not any(slug == workspace["slug"] for _, slug in harness.created)
        released.append(done)
        if not done:
            failure = harness.cli("done", workspace["slug"], "--force", cwd=repo, timeout=180)
            release_errors.append(f"{workspace['slug']}: "
                                  f"{(failure.stdout + failure.stderr).strip()[:200]}")
    checks.that("every workspace is released after the business commands",
                all(released), f"still registered: {[slug for _, slug in harness.created]}")
    if not all(released):
        notes.append("DEFECT: a workspace could not be released. Git refused to delete the "
                     "checkout (a real dependency tree is large) and berth deliberately does "
                     "not fall back to a recursive delete, so the workspace stays pending "
                     "removal and berth run refuses to enter it.")
        notes.append("release errors: " + " | ".join(release_errors))
        notes.append("cleanup: " + harness.cleanup_stuck_repo(repo))
    return {"checks": checks.rows, "timings_ms": timings, "resources": resources, "notes": notes}
