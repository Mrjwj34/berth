#!/usr/bin/env python3
"""Real Linux-container acceptance test; engine failures are failures, not skips.

Usage: python3 tests/container_e2e.py /absolute/path/to/lane lane-runtime:test
Only temporary checkouts and containers owned by this test are mutated.
"""
import concurrent.futures
import http.server
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import threading
import time
import urllib.request

LANE = str(Path(sys.argv[1]).resolve())
IMAGE = sys.argv[2]
ENGINE = os.environ.get("LANE_TEST_ENGINE", "docker")


def command(argv, cwd, env, timeout=90):
    p = subprocess.run(argv, cwd=cwd, env=env, text=True, capture_output=True,
                       timeout=timeout)
    if p.returncode:
        raise RuntimeError(f"{argv}: exit {p.returncode}\n{p.stdout}\n{p.stderr}")
    return p.stdout


def get(port):
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/", timeout=5) as r:
        return r.read().decode()


class HostSentinel(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"HOST-NOT-WORKSPACE")

    def log_message(self, *_):
        pass


def main():
    with tempfile.TemporaryDirectory(prefix="lane-container-e2e-") as tmp:
        root = Path(tmp)
        repo = root / "project with spaces"
        repo.mkdir()
        env = dict(os.environ, LANE_HOME=str(root / "lane-home"))
        for key in ("GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"):
            env.pop(key, None)
        run = lambda *args, cwd=repo: command([LANE, *args], cwd, env)
        git = lambda *args: command(["git", *args], repo, env)
        command([ENGINE, "info"], repo, env)
        command([ENGINE, "image", "inspect", IMAGE], repo, env)
        # Hold a host port in the old remapper's problematic 20000..39999 range.
        sentinel = None
        for listen in range(30000, 30100):
            try:
                sentinel = http.server.ThreadingHTTPServer(("127.0.0.1", listen), HostSentinel)
                break
            except OSError:
                continue
        if sentinel is None:
            raise RuntimeError("could not reserve host sentinel")
        threading.Thread(target=sentinel.serve_forever, daemon=True).start()
        git("init", "-b", "main")
        git("config", "user.name", "lane e2e")
        git("config", "user.email", "lane-e2e@example.invalid")
        (repo / "server.py").write_text(f'''import http.server, os
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(os.environ["LANE_SLUG"].encode())
http.server.ThreadingHTTPServer(("127.0.0.1", {listen}), Handler).serve_forever()
''')
        (repo / "lane.yaml").write_text(f'''version: 1
base: main
runtime:
  backend: container
  engine: {ENGINE}
  image: {IMAGE}
ports: [web]
listen: {{web: {listen}}}
hooks:
  setup:
    - 'printf "$LANE_SLUG" > "$LANE_DATA_DIR/seed"'
processes:
  web:
    command: python3 server.py
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: {listen}, path: /}}
      initial_delay_seconds: 0
      period_seconds: 1
''')
        git("add", ".")
        git("commit", "-m", "acceptance fixture")
        names = ("alpha", "beta")
        try:
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                started = list(pool.map(lambda n: json.loads(run("new", n, "--up", "--json")), names))
            first, second = started
            assert first["ports"]["web"] != second["ports"]["web"], started
            for ws in started:
                assert get(ws["ports"]["web"]) == ws["slug"], ws
                probe = ("import urllib.request,os; "
                         f"assert urllib.request.urlopen('http://127.0.0.1:{listen}/',timeout=5).read().decode()==os.environ['LANE_SLUG']")
                run("run", "--", "python3", "-c", probe, cwd=ws["path"])
                assert Path(ws["path"], ".lane", "data", "seed").read_text() == ws["slug"]
                assert run("run", "--", "git", "rev-parse", "--show-toplevel", cwd=ws["path"]).strip() == "/workspace"
                run("run", "--", "sh", "-c", "FOO=bar; cd /workspace && test \"$FOO\" = bar", cwd=ws["path"])
            assert get(listen) == "HOST-NOT-WORKSPACE"
            state = json.loads((root / "lane-home" / "state.json").read_text())
            identity = state["workspaces"][first["path"]]["id"]
            container_name = "lane-" + identity
            inspect = lambda: json.loads(command([ENGINE, "inspect", container_name], repo, env))[0]
            original_id = inspect()["Id"]
            run("up", first["path"])
            assert inspect()["Id"] == original_id, "up recreated the workspace runtime"
            run("down", first["path"])
            assert not inspect()["State"]["Running"]
            assert get(second["ports"]["web"]) == "beta"
            run("up", first["path"])
            assert inspect()["Id"] == original_id, "restart did not reuse the container"
            assert get(first["ports"]["web"]) == "alpha"
            # Cancellation must stop the command's runtime, not just docker exec.
            proc = subprocess.Popen([LANE, "run", "--", "sh", "-c",
                                     "touch .lane/data/started; sleep 300"],
                                    cwd=first["path"], env=env,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            try:
                deadline = time.monotonic() + 15
                while not Path(first["path"], ".lane", "data", "started").exists():
                    if proc.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("one-off command did not start")
                    time.sleep(.1)
                proc.send_signal(signal.SIGINT)
                proc.communicate(timeout=45)
                assert proc.returncode != 0, "cancelled command returned success"
                assert not inspect()["State"]["Running"], "cancelled command runtime leaked"
                assert get(second["ports"]["web"]) == "beta"
            finally:
                if proc.poll() is None:
                    proc.kill()
                    proc.communicate(timeout=5)
            run("done", first["path"])
            run("done", second["path"])
            assert not json.loads((root / "lane-home" / "state.json").read_text())["workspaces"]
            print("PASS: parallel same-port workspaces, host-loopback separation, shared execution context, private data, Git, reuse, cancellation and cleanup")
        finally:
            for n in names:
                subprocess.run([LANE, "done", n, "--force"], cwd=repo, env=env,
                               capture_output=True, timeout=90)
            sentinel.shutdown()
            sentinel.server_close()


if __name__ == "__main__":
    main()
