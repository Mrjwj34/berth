#!/usr/bin/env bash
set -euo pipefail

payload="$(cat)"
name="$(printf '%s' "$payload" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("name",""))')"

if [[ -z "$name" ]]; then
  echo "worktree name is empty. Provide .name in the WorktreeCreate JSON payload." >&2
  exit 1
fi

# lane/git chatter stays on stderr; only the worktree path is captured from stdout.
worktree_path="$(lane new "$name" --print-path)"

if [[ -z "$worktree_path" ]]; then
  echo "lane new printed no worktree path. Run lane doctor and inspect stderr." >&2
  exit 1
fi

if [[ -d "$worktree_path" ]]; then
  worktree_path="$(cd "$worktree_path" && pwd)"
fi

printf '%s\n' "$worktree_path"
