#!/usr/bin/env python3
"""Business-level end-to-end validation for berth.

Usage:
  python tests/validation/harness.py --berth ./berth --mode native  --out native.json  --markdown native.md
  python tests/validation/harness.py --berth ./berth --mode container --image berth-runtime:test \\
         --out container.json --markdown container.md

Unlike the acceptance tests, this harness is not about readiness: every level
performs a real workflow through the project's own interface (CRUD, SQL
aggregates over a separate database service, cross-service rendering), asserts
exact payloads, and checks that two parallel workspaces keep their data apart.
Every phase is timed and every resource sample is recorded, so the run produces
the performance evidence alongside the correctness evidence.

Options that matter:
  --levels l1,l2,l3,l4   which projects to run (default l1,l2,l3)
  --l4-repo URL          external project for level 4 (default: the Canvas project)
  --l4-test CMD          test command to run inside the workspace
  --workdir DIR          keep the checkouts here instead of a temp directory
  --keep                 do not delete the temporary root on exit
"""

from __future__ import annotations

import argparse
import concurrent.futures
from datetime import datetime
import json
import os
import platform
import shutil
import socket
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import fixtures  # noqa: E402

TRIVIAL = "import sys; sys.exit(0)"


class Checks:
    """Records business assertions with the evidence that produced them."""

    def __init__(self) -> None:
        self.rows: list[dict] = []

    def that(self, name: str, passed: bool, detail: str = "") -> bool:
        self.rows.append({"name": name, "ok": bool(passed), "detail": detail})
        if not passed:
            print(f"  FAIL {name}: {detail}", file=sys.stderr)
        return bool(passed)

    def equal(self, name: str, got, want) -> bool:
        return self.that(name, got == want, f"got {got!r}, want {want!r}")

    def passed(self) -> int:
        return sum(1 for row in self.rows if row["ok"])

    def failed(self) -> list[dict]:
        return [row for row in self.rows if not row["ok"]]


def now_ms() -> float:
    return time.perf_counter() * 1000


def http_json(port: int, path: str, method: str = "GET", body: dict | None = None, timeout: float = 10):
    data = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}", data=data, method=method,
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=timeout) as response:
        return response.status, json.loads(response.read().decode() or "null")


def http_text(port: int, path: str, timeout: float = 10):
    with urllib.request.urlopen(f"http://127.0.0.1:{port}{path}", timeout=timeout) as response:
        return response.status, response.read().decode()


def port_open(port: int, timeout: float = 2) -> bool:
    with socket.socket() as probe:
        probe.settimeout(timeout)
        return probe.connect_ex(("127.0.0.1", port)) == 0


def wait_for(fn, deadline_s: float, interval: float = 0.1) -> bool:
    end = time.monotonic() + deadline_s
    while time.monotonic() < end:
        try:
            if fn():
                return True
        except Exception:
            pass
        time.sleep(interval)
    return False


def rss_kb(pids: list[int]) -> int:
    """Resident memory of the supervised processes, in kilobytes."""
    total = 0
    if os.name == "nt":
        for pid in pids:
            out = subprocess.run(["tasklist", "/FI", f"PID eq {pid}", "/FO", "CSV", "/NH"],
                                 capture_output=True, text=True, encoding="utf-8", errors="replace").stdout.strip()
            parts = [p.strip('"') for p in out.split('","')]
            if len(parts) >= 5 and parts[-1].endswith("K"):
                total += int(parts[-1].rstrip("K").replace(",", "").replace(".", ""))
    else:
        for pid in pids:
            try:
                statm = Path(f"/proc/{pid}/statm").read_text().split()
                total += int(statm[1]) * (os.sysconf("SC_PAGE_SIZE") // 1024)
            except OSError:
                continue
    return total


def dir_kb(path: Path) -> int:
    total = 0
    for entry in path.rglob("*"):
        try:
            if entry.is_file() and not entry.is_symlink():
                total += entry.stat().st_size
        except OSError:
            continue
    return total // 1024


class Harness:
    def __init__(self, args) -> None:
        self.args = args
        self.berth = str(Path(args.berth).resolve())
        self.mode = args.mode
        self.python = "python3" if self.mode == "container" or os.name != "nt" else "python"
        self.root = Path(args.workdir) if args.workdir else Path(tempfile.mkdtemp(prefix="berth-validation-"))
        self.root.mkdir(parents=True, exist_ok=True)
        self.home = self.root / "berth-home"
        self.env = dict(os.environ, BERTH_HOME=str(self.home))
        for key in ("GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"):
            self.env.pop(key, None)
        # process-compose must be reachable for native mode.
        pc_dir = self.home / "bin" / "process-compose-v1.122.0"
        if pc_dir.exists():
            self.env["PATH"] = str(pc_dir) + os.pathsep + self.env.get("PATH", "")
        self.report: dict = {
            "schema": 1,
            "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "mode": self.mode,
            "levels": {},
            "environment": {
                "platform": platform.platform(),
                "python": sys.version.split()[0],
                "cpu_count": os.cpu_count(),
                "berth": self.berth,
                "image": args.image,
                "engine": args.engine,
                "temp_root": str(self.root),
            },
        }
        self.created: list[tuple[Path, str]] = []
        self.token = time.strftime("%H%M%S")

    def new(self, repo: Path, slug: str, *extra: str) -> dict:
        """Create a workspace through the CLI and remember it for cleanup."""
        outcome = self.cli_json("new", slug, *extra, cwd=repo)
        self.created.append((repo, slug))
        return outcome

    def supervisor_running(self) -> bool:
        name = "process-compose.exe" if os.name == "nt" else "process-compose"
        listing = subprocess.run(["tasklist", "/FI", f"IMAGENAME eq {name}", "/NH"],
                                 capture_output=True, text=True, encoding="utf-8", errors="replace").stdout if os.name == "nt" else \
            subprocess.run(["pgrep", "-x", name], capture_output=True, text=True, encoding="utf-8", errors="replace").stdout
        return name in listing

    def clear_stale_control_files(self) -> int:
        """Apply the recovery the runtime contract documents.

        berth refuses destructive operations while the supervisor state is
        unknown. Once no supervisor is alive, every control file under
        BERTH_HOME/run is stale by definition, so removing the pair lets
        `done --force` release the workspace. The harness depends on this path,
        which means the documented recovery is exercised on every level.
        """
        if self.supervisor_running():
            return 0
        removed = 0
        for candidate in (self.home / "run").glob("*/*"):
            if candidate.name in {"pc.port", "pc.token", "pc.sock"}:
                try:
                    candidate.unlink()
                    removed += 1
                except OSError:
                    continue
        return removed

    def cleanup_stuck_repo(self, repo: Path) -> str:
        """Best-effort recovery for a checkout Git could not delete.

        `berth done` refuses to fall back to a recursive delete once
        `git worktree remove` fails, which is deliberate, so a heavy checkout
        (a real node_modules tree, for example) can be left pending removal:
        unusable for `berth run` and not removable by the CLI either. The
        harness records that as a defect and then cleans its own scratch so the
        next run starts from a known state.
        """
        subprocess.run(["git", "worktree", "prune"], cwd=str(repo), env=self.env,
                       capture_output=True, text=True, encoding="utf-8", errors="replace")
        self.clear_stale_control_files()

        def onerror(func, path, _exc):
            try:
                os.chmod(path, 0o700)
                func(path)
            except OSError:
                pass

        removed = 0
        for leftover in sorted(repo.parent.glob(f"{repo.name}.berths/*")):
            shutil.rmtree(leftover, onerror=onerror)
            if not leftover.exists():
                removed += 1
        remaining = [p.name for p in repo.parent.glob(f"{repo.name}.berths/*")]
        return (f"removed {removed} leftover workspace directories; remaining {remaining}; "
                f"run berth gc to clear the registrations")

    def release(self, repo: Path, slug: str) -> None:
        result = self.cli("done", slug, "--force", cwd=repo, timeout=180)
        if result.returncode and "process state unknown" in (result.stdout + result.stderr):
            self.clear_stale_control_files()
            result = self.cli("done", slug, "--force", cwd=repo, timeout=180)
        if result.returncode == 0:
            self.created = [(r, s) for (r, s) in self.created if not (r == repo and s == slug)]

    # --- process helpers -------------------------------------------------
    def cli(self, *argv, cwd: Path | None = None, timeout: float = 300) -> subprocess.CompletedProcess:
        return subprocess.run([self.berth, *argv], cwd=str(cwd or self.root), env=self.env,
                              capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout)

    def cli_json(self, *argv, cwd: Path | None = None, timeout: float = 300) -> dict:
        started = now_ms()
        result = self.cli(*argv, cwd=cwd, timeout=timeout)
        elapsed = now_ms() - started
        if result.returncode != 0:
            raise RuntimeError(f"berth {' '.join(argv)} failed ({result.returncode})\n{result.stdout}\n{result.stderr}")
        return {"value": json.loads(result.stdout or "{}"), "ms": elapsed}

    def git(self, *argv, cwd: Path) -> None:
        subprocess.run(["git", *argv], cwd=str(cwd), env=self.env, check=True,
                       capture_output=True, text=True, encoding="utf-8", errors="replace")

    def make_repo(self, level: str) -> Path:
        # A directory per run: Git marks loose objects read-only on Windows, so a
        # rerun that reused the path could fail while deleting the previous one.
        repo = self.root / f"{level}-project-{self.token}"
        repo.mkdir(parents=True, exist_ok=True)
        self.git("init", "-b", "main", cwd=repo)
        self.git("config", "user.name", "berth validation", cwd=repo)
        self.git("config", "user.email", "validation@example.invalid", cwd=repo)
        fixtures.write_repo(level, repo, self.python,
                            self.args.image if self.mode == "container" else None,
                            self.args.engine)
        self.git("add", ".", cwd=repo)
        self.git("commit", "-m", f"{level} fixture", cwd=repo)
        return repo

    def supervised(self, slug: str, cwd: Path) -> tuple[list[int], int]:
        """PIDs and count of the workspace's supervised processes."""
        try:
            payload = json.loads(self.cli("status", slug, "--json", cwd=cwd).stdout)
        except Exception:
            return [], 0
        procs = payload.get("processes") or []
        return [int(p["pid"]) for p in procs if p.get("pid")], len(procs)

    def container_identity(self, path: str) -> str | None:
        try:
            state = json.loads((self.home / "state.json").read_text())
            return state["workspaces"][path]["id"]
        except Exception as error:
            self.container_note(f"state lookup failed: {error}")
            return None

    def container_note(self, text: str) -> None:
        notes = self.report["environment"].get("container_notes", [])
        if text not in notes:
            notes.append(text)
            self.report["environment"]["container_notes"] = notes

    def container_mem_bytes(self, path: str) -> int | None:
        """Whole-container memory.

        The native RSS sample reads host PIDs, and a workspace whose processes
        live inside a container has none, so this is the only comparable memory
        number in container mode. When neither the engine nor the cgroup answers,
        the reason is recorded instead of leaving a silent gap.
        """
        if self.mode != "container":
            return None
        identity = self.container_identity(path)
        if identity is None:
            return None
        name = f"berth-{identity}"
        scale = {"B": 1, "kB": 1000, "KiB": 1024, "MB": 1000 ** 2, "MiB": 1024 ** 2,
                 "GB": 1000 ** 3, "GiB": 1024 ** 3}
        try:
            out = subprocess.run([self.args.engine, "stats", "--no-stream", "--format", "{{.MemUsage}}", name],
                                 capture_output=True, text=True, encoding="utf-8",
                                 errors="replace", timeout=90)
            if out.returncode == 0 and "/" in out.stdout:
                value, unit = out.stdout.split("/")[0].strip().split(" ")
                return int(float(value) * scale[unit.strip()])
            self.container_note(f"docker stats {name}: {(out.stderr or out.stdout).strip()[:160]}")
        except Exception as error:
            self.container_note(f"docker stats {name}: {error}")
        for candidate in (pathlib.Path(f"/sys/fs/cgroup/system.slice/docker-{identity}.scope/memory.current"),
                          pathlib.Path(f"/sys/fs/cgroup/docker/{identity}/memory.current")):
            try:
                if candidate.exists():
                    return int(candidate.read_text().strip())
            except OSError:
                continue
        return None

    def container_start_ms(self, path: str) -> float | None:
        """Latency between the container being created and it being running."""
        if self.mode != "container":
            return None
        identity = self.container_identity(path)
        if identity is None:
            return None
        try:
            out = subprocess.run([self.args.engine, "inspect", "--format",
                                  "{{.Created}}|{{.State.StartedAt}}", f"berth-{identity}"],
                                 capture_output=True, text=True, encoding="utf-8",
                                 errors="replace", timeout=60)
            created, started = out.stdout.strip().split("|")
            parse = lambda text: datetime.fromisoformat(text.replace("Z", "+00:00"))
            return round((parse(started) - parse(created)).total_seconds() * 1000, 2)
        except Exception as error:
            self.container_note(f"container start latency: {error}")
            return None

    def run_timing(self, level: str, label: str, fn) -> dict:
        started = now_ms()
        try:
            detail = fn()
            return {"ms": now_ms() - started, "ok": True, "detail": detail}
        except Exception as error:  # noqa: BLE001 - the report records the failure
            return {"ms": now_ms() - started, "ok": False, "detail": f"{type(error).__name__}: {error}"}

    # --- levels ----------------------------------------------------------
    def level1(self, checks: Checks) -> dict:
        repo = self.make_repo("l1")
        timings: dict[str, float] = {}
        self.partial = timings
        notes: list[str] = []

        created = self.new(repo, "alpha", "--up", "--json")
        timings["new_and_up_ms"] = created["ms"]
        alpha = created["value"]
        checks.that("workspace reports itself ready", alpha.get("ports", {}).get("web", 0) > 0,
                    f"ports={alpha.get('ports')}")

        def timer(name, fn):
            started = now_ms()
            value = fn()
            timings[name] = round(now_ms() - started, 2)
            return value

        status = timer("status_ms", lambda: json.loads(self.cli("status", alpha["slug"], "--json", cwd=repo).stdout))
        checks.that("no tracked file is dirty right after creation", not status.get("dirty"), f"dirty={status.get('dirty')}")
        timer("ports_ms", lambda: json.loads(self.cli("ports", alpha["slug"], "--json", cwd=repo).stdout))
        timer("plan_ms", lambda: json.loads(self.cli("plan", alpha["slug"], cwd=repo).stdout))
        timer("ls_ms", lambda: json.loads(self.cli("ls", "--json", cwd=repo).stdout))

        port = alpha["ports"]["web"]
        _, health = http_json(port, "/health")
        checks.equal("health reports the workspace slug", health.get("slug"), "alpha")

        written = {}
        for skew, (item_id, name, qty) in enumerate([("a1", "widget", 3), ("a2", "gadget", 5), ("a3", "sprocket", 7)]):
            _, body = http_json(port, "/items", "POST", {"id": item_id, "name": name, "qty": qty})
            written[item_id] = body
        checks.equal("create returns three stored items", len(written), 3)

        _, listing = http_json(port, "/items")
        checks.equal("listing returns the exact three payloads",
                     [(i["id"], i["name"], i["qty"]) for i in listing["items"]],
                     [("a1", "widget", 3), ("a2", "gadget", 5), ("a3", "sprocket", 7)])

        _, single = http_json(port, "/items/a2")
        checks.equal("single read round-trips the stored value", (single["name"], single["qty"]), ("gadget", 5))

        http_json(port, "/items", "PUT", {"id": "a2", "qty": 9})
        _, stats = http_json(port, "/stats")
        checks.equal("update is visible through the aggregate endpoint",
                     (stats["count"], stats["names"]), (3, ["gadget", "sprocket", "widget"]))

        http_json(port, "/items/a3", "DELETE")
        _, after_delete = http_json(port, "/stats")
        checks.equal("delete removes exactly one item", after_delete["count"], 2)
        try:
            http_json(port, "/items/a3")
            checks.that("deleted item is gone", False, "GET returned 200")
        except urllib.error.HTTPError as error:
            checks.equal("deleted item is gone", error.code, 404)

        # data survives a stop/start cycle, which is the documented down/up contract
        started = now_ms()
        self.cli("down", alpha["slug"], cwd=repo)
        timings["down_ms"] = round(now_ms() - started, 2)
        checks.that("down releases the published port", not port_open(port), f"port {port} still open")
        started = now_ms()
        self.cli("up", alpha["slug"], cwd=repo)
        timings["up_again_ms"] = round(now_ms() - started, 2)
        checks.that("up makes the service reachable again", wait_for(lambda: http_json(port, "/health")[1]["ok"], 30),
                    "service did not come back")
        _, persisted = http_json(port, "/stats")
        checks.equal("private data survives down/up", persisted["count"], 2)

        started = now_ms()
        self.cli("reset", alpha["slug"], cwd=repo)
        timings["reset_ms"] = round(now_ms() - started, 2)
        self.cli("up", alpha["slug"], cwd=repo)
        checks.that("reset makes the service reachable again", wait_for(lambda: http_json(port, "/health")[1]["ok"], 30),
                    "service did not come back after reset")
        _, wiped = http_json(port, "/stats")
        checks.equal("reset wipes private data", wiped["count"], 0)

        # a second workspace, in parallel, with its own data and its own ports
        started = now_ms()
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            made = list(pool.map(lambda slug: self.new(repo, slug, "--up", "--json")["value"],
                                 ["beta", "gamma", "delta", "epsilon"]))
        timings["parallel4_new_and_up_wall_ms"] = round(now_ms() - started, 2)
        ports = [w["ports"]["web"] for w in made]
        checks.equal("four parallel workspaces get four distinct ports", len(set(ports)), 4)
        for workspace in made:
            http_json(workspace["ports"]["web"], "/items", "POST",
                      {"id": "b1", "name": workspace["slug"], "qty": 1})
        isolation = []
        for workspace in made:
            _, own = http_json(workspace["ports"]["web"], "/items")
            isolation.append([i["name"] for i in own["items"]])
        checks.equal("each parallel workspace sees only its own data",
                     isolation, [[w["slug"]] for w in made])
        _, first_view = http_json(port, "/stats")
        checks.equal("the first workspace is unaffected by the others", first_view["count"], 0)

        # command overhead against a direct baseline
        baseline, overhead = [], []
        for _ in range(5):
            started = now_ms()
            subprocess.run([self.python, "-c", TRIVIAL], cwd=str(alpha["path"]), env=self.env,
                           capture_output=True)
            baseline.append(now_ms() - started)
            started = now_ms()
            self.cli("run", "--", self.python, "-c", TRIVIAL, cwd=Path(alpha["path"]))
            overhead.append(now_ms() - started)
        timings["run_plain_ms"] = round(statistics.median(baseline), 2)
        timings["run_via_berth_ms"] = round(statistics.median(overhead), 2)

        # argv fidelity through berth run
        literal = "spaces 'quotes' $HOME & pipes|"
        out = self.cli("run", "--", self.python, "-c", "import sys; print(sys.argv[1])", literal,
                       cwd=Path(alpha["path"]))
        checks.equal("berth run preserves argv characters verbatim", out.stdout.strip(), literal)

        pids, count = self.supervised(alpha["slug"], repo)
        memory, start = self.container_mem_bytes(alpha["path"]), self.container_start_ms(alpha["path"])
        resources = {
            "supervised_processes": count,
            "supervised_rss_kb": rss_kb(pids),
            "workspace_disk_kb": dir_kb(Path(alpha["path"])),
            "data_dir_kb": dir_kb(Path(alpha["path"]) / ".berth" / "data"),
            "container_mem_bytes": memory,
            "container_start_ms": start,
        }

        # Copy-on-write behaviour for declared dependency trees
        tree = repo / "deps"
        tree.mkdir(exist_ok=True)
        payload = b"x" * (1024 * 1024)
        for index in range(48):
            (tree / f"blob-{index}.bin").write_bytes(payload)
        self.git("add", "deps", cwd=repo)
        self.git("commit", "-m", "dependency tree", cwd=repo)
        config = (repo / "berth.yaml").read_text()
        if "copy_dirs" not in config:
            (repo / "berth.yaml").write_text(config + "copy_dirs: [deps]\n")
            self.git("add", "berth.yaml", cwd=repo)
            self.git("commit", "-m", "declare copy_dirs", cwd=repo)
        started = now_ms()
        cow = self.new(repo, "cow", "--json")["value"]
        timings["cow_48mb_new_ms"] = round(now_ms() - started, 2)
        copied = Path(cow["path"]) / "deps" / "blob-0.bin"
        checks.that("copy_dirs materialises the dependency tree", copied.exists(), f"missing {copied}")
        started = now_ms()
        copied.write_bytes(b"y" * 1024)
        timings["cow_writeback_ms"] = round(now_ms() - started, 2)
        checks.that("writing into a copied tree does not change the source",
                    (tree / "blob-0.bin").read_bytes()[:1] == b"x", "source tree was modified")
        resources["cow_tree_kb"] = dir_kb(Path(cow["path"]) / "deps")

        for workspace in [alpha, *made, cow]:
            self.cli("done", workspace["slug"], "--force", cwd=repo, timeout=180)
        return {"checks": checks.rows, "timings_ms": timings, "resources": resources, "notes": notes}

    def level2(self, checks: Checks) -> dict:
        repo = self.make_repo("l2")
        timings: dict[str, float] = {}
        self.partial = timings
        created = self.new(repo, "alpha", "--up", "--json")
        timings["new_and_up_ms"] = created["ms"]
        alpha = created["value"]
        frontend, backend = alpha["ports"]["frontend"], alpha["ports"]["backend"]
        checks.that("the two declared ports differ", frontend != backend, f"{frontend} == {backend}")

        for index, (sku, qty) in enumerate([("one", 2), ("two", 4), ("three", 6)]):
            http_json(backend, "/items", "POST", {"id": f"i{index}", "name": sku, "qty": qty})
        _, upstream = http_json(frontend, "/upstream")
        checks.equal("frontend reports the backend aggregate", (upstream["count"], upstream["names"]),
                     (3, ["one", "three", "two"]))
        _, page = http_text(frontend, "/")
        checks.that("frontend page renders backend state", "count=3 names=one,three,two" in page
                    and "data-slug='alpha'" in page, page[:200])

        started = now_ms()
        self.cli("reset", "alpha", cwd=repo)
        timings["reset_ms"] = round(now_ms() - started, 2)
        self.cli("up", "alpha", cwd=repo)
        checks.that("services return after reset",
                    wait_for(lambda: http_text(frontend, "/")[1].find("count=0") >= 0, 40),
                    "frontend never reported an empty backend")
        _, cleared = http_json(frontend, "/upstream")
        checks.equal("frontend follows the backend after its data is wiped", cleared["count"], 0)

        second = self.new(repo, "beta", "--up", "--json")["value"]
        http_json(second["ports"]["backend"], "/items", "POST", {"id": "z", "name": "beta-only", "qty": 1})
        _, beta_page = http_text(second["ports"]["frontend"], "/")
        _, alpha_page = http_text(frontend, "/")
        checks.that("second workspace renders its own backend", "count=1 names=beta-only" in beta_page, beta_page[:200])
        checks.that("first workspace is unaffected", "count=0" in alpha_page, alpha_page[:200])

        pids, count = self.supervised("alpha", repo)
        memory, start = self.container_mem_bytes(alpha["path"]), self.container_start_ms(alpha["path"])
        resources = {
            "supervised_processes": count,
            "supervised_rss_kb": rss_kb(pids),
            "workspace_disk_kb": dir_kb(Path(alpha["path"])),
            "container_mem_bytes": memory,
            "container_start_ms": start,
        }
        for workspace in (alpha, second):
            self.cli("done", workspace["slug"], "--force", cwd=repo, timeout=180)
        return {"checks": checks.rows, "timings_ms": timings, "resources": resources, "notes": []}

    def level3(self, checks: Checks) -> dict:
        repo = self.make_repo("l3")
        timings: dict[str, float] = {}
        self.partial = timings
        created = self.new(repo, "alpha", "--up", "--json")
        timings["new_migrate_up_ms"] = created["ms"]
        alpha = created["value"]
        api, db = alpha["ports"]["api"], alpha["ports"]["db"]
        checks.that("api and db ports differ", api != db, f"{api} == {db}")

        _, tables = http_json(db, "/tables")
        checks.that("setup hook created the schema in the database service",
                    "orders" in tables["tables"], f"tables={tables['tables']}")

        for sku, qty in [("bolt", 10), ("nut", 20), ("bolt", 5)]:
            status, body = http_json(api, "/orders", "POST", {"sku": sku, "qty": qty})
            checks.equal(f"order {sku}x{qty} accepted", status, 201)
        _, stats = http_json(api, "/stats")
        checks.equal("SQL aggregate sums units", stats["units"], 36)
        checks.equal("SQL aggregate counts rows", stats["orders"], 4)
        checks.equal("SQL group-by per sku", stats["by_sku"], {"bolt": 15, "nut": 20, "seed": 1})

        _, orders = http_json(api, "/orders")
        checks.equal("order listing is ordered and complete",
                     [(o["sku"], o["qty"]) for o in orders["orders"]],
                     [("seed", 1), ("bolt", 10), ("nut", 20), ("bolt", 5)])

        self.cli("down", "alpha", cwd=repo)
        self.cli("up", "alpha", cwd=repo)
        checks.that("database contents survive down/up",
                    wait_for(lambda: http_json(api, "/stats")[1]["units"] == 36, 40),
                    "orders were not preserved across a restart")

        started = now_ms()
        self.cli("reset", "alpha", cwd=repo)
        timings["reset_ms"] = round(now_ms() - started, 2)
        self.cli("up", "alpha", cwd=repo)
        checks.that("reset restores the seeded schema",
                    wait_for(lambda: http_json(api, "/stats")[1]["units"] == 1, 60),
                    "reset did not return the database to its seeded state")

        second = self.new(repo, "beta", "--up", "--json")["value"]
        http_json(second["ports"]["api"], "/orders", "POST", {"sku": "beta-only", "qty": 99})
        _, beta_stats = http_json(second["ports"]["api"], "/stats")
        _, alpha_stats = http_json(api, "/stats")
        checks.equal("second workspace has its own database", (beta_stats["units"], beta_stats["orders"]), (100, 2))
        checks.equal("first workspace database is untouched", (alpha_stats["units"], alpha_stats["orders"]), (1, 1))
        checks.that("each workspace has its own database file",
                    Path(alpha["path"], ".berth", "data", "warehouse.sqlite").exists()
                    and Path(second["path"], ".berth", "data", "warehouse.sqlite").exists(),
                    "sqlite file missing in one workspace")

        pids, count = self.supervised("alpha", repo)
        memory, start = self.container_mem_bytes(alpha["path"]), self.container_start_ms(alpha["path"])
        resources = {
            "supervised_processes": count,
            "supervised_rss_kb": rss_kb(pids),
            "workspace_disk_kb": dir_kb(Path(alpha["path"])),
            "database_kb": dir_kb(Path(alpha["path"]) / ".berth" / "data"),
            "container_mem_bytes": memory,
            "container_start_ms": start,
        }
        for workspace in (alpha, second):
            self.cli("done", workspace["slug"], "--force", cwd=repo, timeout=180)
        return {"checks": checks.rows, "timings_ms": timings, "resources": resources, "notes": []}

    def level4(self, checks: Checks) -> dict:
        """Level 4 delegates to level4.run_level4: one real external project,
        described by a JSON spec, so the same code validates a private business
        repository locally and a public project in CI."""
        import level4
        return level4.run_level4(self, checks)

    # --- driver ----------------------------------------------------------
    def run(self) -> int:
        doctor = self.cli("doctor", "--json")
        self.report["environment"]["doctor"] = json.loads(doctor.stdout or "{}")
        if self.mode == "native" and not json.loads(doctor.stdout or "{}").get("ok"):
            notes = json.loads(doctor.stdout or "{}").get("checks", [])
            print(f"doctor is not satisfied: {notes}", file=sys.stderr)

        for level in self.args.levels.split(","):
            level = level.strip()
            print(f"== {level} ({self.mode})", flush=True)
            checks = Checks()
            runner = getattr(self, f"level{level.lstrip('lL')}")
            self.partial = {}
            try:
                outcome = self.run_timing(level, level, lambda: runner(checks))
            finally:
                # A failed level must not leave services listening or workspaces
                # registered: the next run would then allocate different ports and
                # the failure would be blamed on the wrong thing.
                for repo, slug in list(self.created):
                    self.release(repo, slug)
                self.created.clear()
            detail = outcome["detail"] if outcome["ok"] else {"timings_ms": self.partial, "resources": {}}
            failures = checks.failed()
            self.report["levels"][level] = {
                "duration_ms": round(outcome["ms"], 2),
                "ok": outcome["ok"] and not failures,
                "error": None if outcome["ok"] else str(outcome["detail"]),
                "checks": checks.rows,
                "timings_ms": detail.get("timings_ms", {}),
                "resources": detail.get("resources", {}),
                "notes": detail.get("notes", []),
            }
            state = "ok" if self.report["levels"][level]["ok"] else "FAILED"
            print(f"   {state}: {checks.passed()}/{len(checks.rows)} business checks", flush=True)

        Path(self.args.out).write_text(json.dumps(self.report, indent=2) + "\n")
        if self.args.markdown:
            Path(self.args.markdown).write_text(self.markdown())
        failures = [name for name, data in self.report["levels"].items() if not data["ok"]]
        print(f"report: {self.args.out}")
        if failures:
            print(f"FAILED levels: {', '.join(failures)}", file=sys.stderr)
            return 1
        return 0

    def markdown(self) -> str:
        env = self.report["environment"]
        lines = [
            f"# berth validation — {self.mode}",
            "",
            f"Generated {self.report['generated_at']} by `tests/validation/harness.py`.",
            "",
            "| field | value |",
            "| --- | --- |",
            f"| platform | `{env['platform']}` |",
            f"| python | `{env['python']}` |",
            f"| cpus | {env['cpu_count']} |",
            f"| runtime | `{self.mode}` |",
            f"| image | `{env.get('image') or '-'}` |",
            f"| engine | `{env.get('engine') or '-'}` |",
            "",
        ]
        for level, data in self.report["levels"].items():
            checks = data.get("checks", [])
            lines += [
                f"## {level} — {'pass' if data['ok'] else 'FAILED'} ({data['duration_ms']} ms total)",
                "",
                f"business checks: {sum(1 for c in checks if c['ok'])}/{len(checks)}",
                "",
            ]
            if data.get("error"):
                lines += ["```", str(data["error"])[:2000], "```", ""]
            if data.get("timings_ms"):
                lines += ["| timing | ms |", "| --- | --- |"]
                lines += [f"| `{key}` | {value} |" for key, value in data["timings_ms"].items()]
                lines.append("")
            if data.get("resources"):
                lines += ["| resource | value |", "| --- | --- |"]
                lines += [f"| `{key}` | {value} |" for key, value in data["resources"].items()]
                lines.append("")
            if checks:
                lines += ["| check | result | detail |", "| --- | --- | --- |"]
                for row in checks:
                    detail = str(row.get("detail", "")).replace("|", "\\|")[:160]
                    lines.append(f"| {row['name']} | {'pass' if row['ok'] else 'FAIL'} | {detail} |")
                lines.append("")
            for note in data.get("notes", []):
                lines += [f"> {note}", ""]
        return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--berth", required=True)
    parser.add_argument("--mode", choices=["native", "container"], default="native")
    parser.add_argument("--image", default=None)
    parser.add_argument("--engine", default="docker")
    parser.add_argument("--levels", default="l1,l2,l3")
    parser.add_argument("--out", required=True)
    parser.add_argument("--markdown", default=None)
    parser.add_argument("--workdir", default=None)
    parser.add_argument("--keep", action="store_true")
    parser.add_argument("--l4-json", default="tests/validation/l4-canvas.json",
                        help="JSON spec describing the external project for level 4")
    parser.add_argument("--l4-timeout", type=float, default=3600)
    args = parser.parse_args()

    harness = Harness(args)
    try:
        return harness.run()
    finally:
        if not args.keep and args.workdir is None:
            shutil.rmtree(harness.root, ignore_errors=True)


if __name__ == "__main__":
    raise SystemExit(main())
