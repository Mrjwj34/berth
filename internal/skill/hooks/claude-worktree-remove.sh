#!/usr/bin/env bash
set -euo pipefail

payload="$(cat)"
worktree_path="$(printf '%s' "$payload" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("worktree_path",""))')"

if [[ -z "$worktree_path" ]]; then
  echo "worktree_path is empty. Provide .worktree_path in the WorktreeRemove JSON payload." >&2
  exit 1
fi

if [[ ! -d "$worktree_path" ]]; then
  exit 0
fi

cd "$worktree_path"
lane down >&2
