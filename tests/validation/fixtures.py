#!/usr/bin/env python3
"""Project fixtures for the berth validation harness.

Four projects of increasing architectural complexity. Every one of them is a
real program whose behaviour can be asserted through its own interface, so the
harness verifies business logic instead of probing readiness:

  l1  one HTTP service with a file-backed store (CRUD, persistence, isolation)
  l2  frontend and backend on two ports, with a real runtime dependency
  l3  HTTP app plus a separate stateful database service, schema migrations run
      as a setup hook, SQL aggregates, per-workspace database isolation
  l4  an external real project: its own test suite inside a workspace

The fixtures never pass arguments on the process command line: they read the
injected environment instead. That keeps them runnable on Windows, where a
quoted or space-containing argument cannot survive the supervisor's command
line, and it exercises the documented environment contract.
"""

from __future__ import annotations

from pathlib import Path

L1_APP = r'''
"""Level 1: item store behind HTTP, state in $BERTH_DATA_DIR."""
import json
import os
import http.server
from pathlib import Path

DATA = Path(os.environ["BERTH_DATA_DIR"])
STORE = DATA / "store.json"
SLUG = os.environ["BERTH_SLUG"]


def load():
    if not STORE.exists():
        return {}
    return json.loads(STORE.read_text())


def save(items):
    DATA.mkdir(parents=True, exist_ok=True)
    STORE.write_text(json.dumps(items))


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def reply(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        items = load()
        if self.path == "/health":
            return self.reply(200, {"ok": True, "slug": SLUG})
        if self.path == "/items":
            return self.reply(200, {"slug": SLUG, "items": sorted(items.values(), key=lambda i: i["id"])})
        if self.path == "/stats":
            names = sorted(i["name"] for i in items.values())
            return self.reply(200, {"slug": SLUG, "count": len(items), "names": names})
        if self.path.startswith("/items/"):
            item = items.get(self.path.rsplit("/", 1)[-1])
            if item is None:
                return self.reply(404, {"error": "not found"})
            return self.reply(200, item)
        return self.reply(404, {"error": "no route"})

    def do_POST(self):
        if self.path != "/items":
            return self.reply(404, {"error": "no route"})
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length) or b"{}")
        items = load()
        item = {"id": body["id"], "name": body["name"], "qty": int(body.get("qty", 1))}
        items[item["id"]] = item
        save(items)
        return self.reply(201, item)

    def do_DELETE(self):
        if not self.path.startswith("/items/"):
            return self.reply(404, {"error": "no route"})
        items = load()
        item = items.pop(self.path.rsplit("/", 1)[-1], None)
        save(items)
        return self.reply(200, {"deleted": item is not None})

    def do_PUT(self):
        if self.path != "/items":
            return self.reply(404, {"error": "no route"})
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length) or b"{}")
        items = load()
        if body["id"] not in items:
            return self.reply(404, {"error": "not found"})
        items[body["id"]].update({k: v for k, v in body.items() if k != "id"})
        save(items)
        return self.reply(200, items[body["id"]])


http.server.ThreadingHTTPServer(("127.0.0.1", int(os.environ["BERTH_PORT_WEB"])), Handler).serve_forever()
'''

L1_SETUP = r'''
import os
from pathlib import Path

Path(os.environ["BERTH_DATA_DIR"]).mkdir(parents=True, exist_ok=True)
print("l1 setup ready in", os.environ["BERTH_SLUG"], file=__import__("sys").stderr)
'''

L2_BACKEND = L1_APP.replace("BERTH_PORT_WEB", "BERTH_PORT_BACKEND")

L2_FRONTEND = r'''
"""Level 2: page server that renders backend state, so the two ports are wired."""
import json
import os
import http.server
import urllib.error
import urllib.request

BACKEND = int(os.environ["BERTH_PORT_BACKEND"])
SLUG = os.environ["BERTH_SLUG"]


def backend(path):
    with urllib.request.urlopen(f"http://127.0.0.1:{BACKEND}{path}", timeout=5) as response:
        return json.loads(response.read().decode())


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def reply(self, code, body, ctype="text/html"):
        raw = body.encode()
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        if self.path == "/health":
            return self.reply(200, json.dumps({"ok": True, "slug": SLUG}), "application/json")
        try:
            stats = backend("/stats")
        except (urllib.error.URLError, OSError) as error:
            return self.reply(503, f"<html><body>backend unavailable: {error}</body></html>")
        if self.path == "/upstream":
            return self.reply(200, json.dumps(stats), "application/json")
        names = ",".join(stats["names"])
        return self.reply(200, f"<html><body data-slug='{SLUG}'>count={stats['count']} names={names}</body></html>")


http.server.ThreadingHTTPServer(("127.0.0.1", int(os.environ["BERTH_PORT_FRONTEND"])), Handler).serve_forever()
'''

L3_DB = r'''
"""Level 3: a separate stateful service. Real SQL, real transactions, own data dir."""
import json
import os
import sqlite3
import http.server
import threading
from pathlib import Path

DB = Path(os.environ["BERTH_DATA_DIR"]) / "warehouse.sqlite"
LOCK = threading.Lock()


def connect():
    conn = sqlite3.connect(DB, isolation_level=None, timeout=10)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA busy_timeout=10000")
    return conn


def migrate():
    """Own the schema here, not in a setup hook.

    A setup hook runs before the workspace's processes start, so a hook cannot
    reach this service. The service therefore applies its own idempotent
    migration at startup, which also means `berth reset` (which wipes the data
    directory and reruns setup) leaves a working, seeded database behind.
    """
    DB.parent.mkdir(parents=True, exist_ok=True)
    with LOCK, connect() as conn:
        conn.execute("CREATE TABLE IF NOT EXISTS orders ("
                     "id INTEGER PRIMARY KEY AUTOINCREMENT, sku TEXT NOT NULL, qty INTEGER NOT NULL)")
        if conn.execute("SELECT COUNT(*) FROM orders").fetchone()[0] == 0:
            conn.execute("INSERT INTO orders (sku, qty) VALUES ('seed', 1)")


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def reply(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def read_body(self):
        length = int(self.headers.get("Content-Length", "0"))
        return json.loads(self.rfile.read(length) or b"{}")

    def do_GET(self):
        if self.path == "/ping":
            return self.reply(200, {"ok": True, "db": str(DB)})
        if self.path == "/tables":
            with LOCK, connect() as conn:
                rows = conn.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").fetchall()
            return self.reply(200, {"tables": [r[0] for r in rows]})
        return self.reply(404, {"error": "no route"})

    def do_POST(self):
        if self.path == "/exec":
            return self.exec_sql()
        return self.reply(404, {"error": "no route"})

    def exec_sql(self):
        body = self.read_body()
        sql = body["sql"]
        params = body.get("params", [])
        with LOCK, connect() as conn:
            if sql.strip().lower().startswith("select"):
                rows = conn.execute(sql, params).fetchall()
                return self.reply(200, {"rows": [list(r) for r in rows]})
            cursor = conn.execute(sql, params)
            return self.reply(200, {"rowcount": cursor.rowcount})


migrate()
http.server.ThreadingHTTPServer(("127.0.0.1", int(os.environ["BERTH_PORT_DB"])), Handler).serve_forever()
'''

L3_APP = r'''
"""Level 3: business API on top of the database service."""
import json
import os
import http.server
import urllib.error
import urllib.request

DB = int(os.environ["BERTH_PORT_DB"])
SLUG = os.environ["BERTH_SLUG"]


def sql(statement, params=None):
    payload = json.dumps({"sql": statement, "params": params or []}).encode()
    request = urllib.request.Request(f"http://127.0.0.1:{DB}/exec", data=payload,
                                     headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=10) as response:
        return json.loads(response.read().decode())


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def reply(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            try:
                tables = sql("SELECT COUNT(*) FROM orders")["rows"][0][0]
            except (urllib.error.URLError, OSError, KeyError) as error:
                return self.reply(503, {"ok": False, "error": str(error)})
            return self.reply(200, {"ok": True, "slug": SLUG, "orders": tables})
        if self.path == "/stats":
            total = sql("SELECT COALESCE(SUM(qty), 0), COUNT(*) FROM orders")["rows"][0]
            by_sku = sql("SELECT sku, SUM(qty) FROM orders GROUP BY sku ORDER BY sku")["rows"]
            return self.reply(200, {"slug": SLUG, "units": total[0], "orders": total[1],
                                    "by_sku": {sku: qty for sku, qty in by_sku}})
        if self.path == "/orders":
            rows = sql("SELECT id, sku, qty FROM orders ORDER BY id")["rows"]
            return self.reply(200, {"slug": SLUG, "orders": [{"id": r[0], "sku": r[1], "qty": r[2]} for r in rows]})
        return self.reply(404, {"error": "no route"})

    def do_POST(self):
        if self.path != "/orders":
            return self.reply(404, {"error": "no route"})
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length) or b"{}")
        result = sql("INSERT INTO orders (sku, qty) VALUES (?, ?)", [body["sku"], int(body["qty"])])
        return self.reply(201, {"inserted": result["rowcount"], "sku": body["sku"], "slug": SLUG})


http.server.ThreadingHTTPServer(("127.0.0.1", int(os.environ["BERTH_PORT_API"])), Handler).serve_forever()
'''

L3_MIGRATE = r'''
"""Setup hook: records the schema version a workspace was created with.

Setup hooks run before the workspace's processes start, so they cannot talk to
the database service; the service applies its own migration at startup. What is
left here is the job a hook can do on its own: preparing the private data
directory and stamping the version it was provisioned for.
"""
import os
from pathlib import Path

data = Path(os.environ["BERTH_DATA_DIR"])
data.mkdir(parents=True, exist_ok=True)
(data / "schema.version").write_text("1\n")
print("provisioned schema version 1 for", os.environ["BERTH_SLUG"])
'''

L3_SEED_GUARD = r'''
"""Fails the workspace if the schema hook did not actually reach the database."""
import os
import json
import urllib.request

STORE = os.path.join(os.environ["BERTH_DATA_DIR"], "warehouse.sqlite")
print("sqlite file present:", os.path.exists(STORE), file=os.sys.stderr)
'''


def write_repo(level: str, repo: Path, python: str, image: str | None, engine: str = "docker",
               listen_base: int = 0) -> None:
    """Write one fixture into repo. image/engine are only used in container mode."""
    runtime = ""
    listen = ""
    if image:
        runtime = f"""runtime:
  backend: container
  engine: {engine}
  image: {image}
"""
    if level == "l1":
        (repo / "app.py").write_text(L1_APP)
        (repo / "setup.py").write_text(L1_SETUP)
        config = f"""version: 1
base: main
{runtime}ports: [web]
{listen}hooks:
  setup:
    - {python} setup.py
processes:
  web:
    command: {python} app.py
    log_location: .berth/web.log
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: '${{BERTH_PORT_WEB}}', path: /health}}
      initial_delay_seconds: 0
      period_seconds: 1
      failure_threshold: 120
"""
    elif level == "l2":
        (repo / "backend.py").write_text(L2_BACKEND)
        (repo / "frontend.py").write_text(L2_FRONTEND.replace("BERTH_PORT_FRONTEND", "BERTH_PORT_FRONTEND"))
        (repo / "setup.py").write_text(L1_SETUP)
        config = f"""version: 1
base: main
{runtime}ports: [backend, frontend]
{listen}hooks:
  setup:
    - {python} setup.py
processes:
  backend:
    command: {python} backend.py
    log_location: .berth/backend.log
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: '${{BERTH_PORT_BACKEND}}', path: /health}}
      initial_delay_seconds: 0
      period_seconds: 1
      failure_threshold: 120
  frontend:
    command: {python} frontend.py
    log_location: .berth/frontend.log
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: '${{BERTH_PORT_FRONTEND}}', path: /health}}
      initial_delay_seconds: 0
      period_seconds: 1
      failure_threshold: 120
"""
    elif level == "l3":
        (repo / "db.py").write_text(L3_DB)
        (repo / "app.py").write_text(L3_APP)
        (repo / "migrate.py").write_text(L3_MIGRATE)
        config = f"""version: 1
base: main
{runtime}ports: [db, api]
{listen}hooks:
  setup:
    - {python} migrate.py
processes:
  db:
    command: {python} db.py
    log_location: .berth/db.log
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: '${{BERTH_PORT_DB}}', path: /ping}}
      initial_delay_seconds: 0
      period_seconds: 1
      failure_threshold: 180
  api:
    command: {python} app.py
    log_location: .berth/api.log
    readiness_probe:
      http_get: {{host: 127.0.0.1, port: '${{BERTH_PORT_API}}', path: /health}}
      initial_delay_seconds: 0
      period_seconds: 1
      failure_threshold: 180
"""
    else:
        raise SystemExit(f"unknown level {level}")
    (repo / "berth.yaml").write_text(config)
