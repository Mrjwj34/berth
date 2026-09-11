#!/usr/bin/env python3
"""Exercise the real CLI and native services on Linux, macOS and Windows."""
import concurrent.futures
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import urllib.request


def main():
    lane = str(Path(sys.argv[1]).resolve())
    python = "python" if os.name == "nt" else "python3"
    with tempfile.TemporaryDirectory(prefix="lane-native-e2e-") as tmp:
        root = Path(tmp)
        repo = root / "project with spaces"
        repo.mkdir()
        env = dict(os.environ, LANE_HOME=str(root / "lane-home"))
        for key in ("GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"):
            env.pop(key, None)

        def command(argv, cwd=repo, code=0):
            result = subprocess.run(argv, cwd=cwd, env=env, capture_output=True,
                                    text=True, timeout=90)
            assert result.returncode == code, (argv, result.returncode, result.stdout, result.stderr)
            return result.stdout

        def run(*args, cwd=repo, code=0):
            return command([lane, *args], cwd, code)

        def get(ws):
            with urllib.request.urlopen(f"http://127.0.0.1:{ws['ports']['web']}/", timeout=5) as response:
                return response.read().decode()

        for args in [("init", "-b", "main"), ("config", "user.name", "lane e2e"),
                     ("config", "user.email", "lane-e2e@example.invalid")]:
            command(["git", *args])
        run("init")
        (repo / "server.py").write_text('''import http.server, os
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(os.environ["LANE_SLUG"].encode())
http.server.ThreadingHTTPServer(("127.0.0.1", int(os.environ["LANE_PORT_WEB"])), Handler).serve_forever()
''')
        (repo / "setup.py").write_text('''import os
from pathlib import Path
Path(os.environ["LANE_DATA_DIR"], "seed").write_text(os.environ["LANE_SLUG"])
print("setup output belongs on stderr")
''')
        (repo / "lane.yaml").write_text(f'''version: 1
base: main
ports: [web]
hooks:
  setup: ['{python} setup.py']
processes:
  web:
    command: {python} server.py
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: '${{LANE_PORT_WEB}}', path: /}}
      initial_delay_seconds: 0
      period_seconds: 1
''')
        command(["git", "add", "."])
        command(["git", "commit", "-m", "native acceptance fixture"])
        assert json.loads(run("doctor", "--json"))["ok"]
        names = ("alpha", "beta")
        try:
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                workspaces = list(pool.map(lambda name: json.loads(run("new", name, "--up", "--json")), names))
            first, second = workspaces
            assert first["ports"]["web"] != second["ports"]["web"]
            for ws in workspaces:
                assert get(ws) == ws["slug"]
                assert Path(ws["path"], ".lane", "data", "seed").read_text() == ws["slug"]
                assert not json.loads(run("status", ws["path"], "--json"))["dirty"]
                assert json.loads(run("plan", ws["path"]))["backend"] == "native"
                assert json.loads(run("attach", ws["path"], "--json"))["env"]["LANE_SLUG"] == ws["slug"]
                literal = "spaces 'quotes' $HOME & pipes|"
                out = run("run", "--", python, "-c", "import sys; print(sys.argv[1])", literal, cwd=ws["path"])
                assert out.strip() == literal
                run("run", "--", python, "-c", "raise SystemExit(7)", cwd=ws["path"], code=7)
            run("down", first["path"])
            assert not json.loads(run("status", first["path"], "--json"))["running"]
            with socket.socket() as stopped:
                stopped.settimeout(2)
                assert stopped.connect_ex(("127.0.0.1", first["ports"]["web"])) != 0
            assert get(second) == "beta"
            run("up", first["path"])
            assert get(first) == "alpha"
            stale = Path(first["path"], ".lane", "data", "stale")
            stale.write_text("discard on reset")
            run("reset", first["path"])
            assert not stale.exists()
            assert Path(first["path"], ".lane", "data", "seed").read_text() == "alpha"
            run("up", first["path"])
            assert get(first) == "alpha" and get(second) == "beta"
            for ws in workspaces:
                run("done", ws["path"])
                assert not Path(ws["path"]).exists()
                assert not command(["git", "branch", "--list", ws["branch"]]).strip()
            assert json.loads(run("ls", "--json"))["workspaces"] == []
            print("PASS: native parallel workspaces, HTTP, JSON, hooks, argv, exit codes, restart, reset and cleanup")
        finally:
            for name in names:
                subprocess.run([lane, "done", name, "--force"], cwd=repo, env=env,
                               capture_output=True, timeout=90)


if __name__ == "__main__":
    main()
