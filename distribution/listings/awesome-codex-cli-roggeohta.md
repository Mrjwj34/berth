# RoggeOhta/awesome-codex-cli

- **URL:** https://github.com/RoggeOhta/awesome-codex-cli
- **Stars:** 515 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** CC0-1.0, default branch `main`, `pushed_at` 2026-09-06T12:00:35Z, **187 open issues**, PR creation policy `all`, ships a `CONTRIBUTING.md`.
- **Activity verdict: LOW VALUE — no external entry PR has been merged in the observable window.**
  `GET repos/RoggeOhta/awesome-codex-cli/pulls?state=closed&per_page=100&sort=updated` filtered to `merged_at != null` returned **zero results**. The eight most recently updated closed PRs are all unmerged:
  - `#107` "Add ralph-harness workflow tool" — `merged_at: null`
  - `#247` "Add codex-switch: one-click provider switching with cross-provider session resume" — `merged_at: null`
  - `#130` "Add Taisly Agent Kit Codex plugin" — `merged_at: null`
  - `#179` "Add Suede Creator Skills collection" — `merged_at: null`
  - `#212` "Add Codex Theme Builder" — `merged_at: null`
  - `#223` "docs: add SandBase CLI MCP bridge" — `merged_at: null`
  - `#98` "Add Agentlas Hephaestus to Cross-Agent Tools" — `merged_at: null`
  - `#136` "Add BaiQuant real-world Codex case study" — `merged_at: null`
  The repository is still pushed to (2026-09-06), so content is added — but evidently by direct commits rather than by merging contributor PRs, and **187 open issues** suggests a large unreviewed backlog. Under this project's own rule ("if entries have not been merged in 6+ months, mark the channel low value"), the stronger finding applies: **no merged entry PR found at all** in the last 100 closed PRs.
- **Project facts used:** `Mrjwj34/berth` — 1 star.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> ## How to Contribute
>
> 1. **Found a great resource?** [Open an issue](https://github.com/RoggeOhta/awesome-codex-cli/issues/new) with the link and a brief description of why it's awesome.
> 2. **Want to add it yourself?** Fork, edit `README.md`, and submit a PR.
>
> ## Quality Standards
>
> Every entry must:
>
> - **Be directly related to Codex CLI** — general AI/LLM tools don't belong unless they have specific Codex integration.
> - **Be actively maintained** — no abandoned projects (last commit > 6 months ago) unless they are stable and still useful.
> - **Have a clear description** — one sentence explaining what it does and why it's worth your time.
> - **Include star badge** — append `![GitHub stars](https://img.shields.io/github/stars/owner/repo?style=flat-square)` for GitHub repos. These update automatically.
>
> ## Format
>
> ```markdown
> - [owner/repo](https://github.com/owner/repo) - One-sentence description explaining its value. ![GitHub stars](https://img.shields.io/github/stars/owner/repo?style=flat-square)
> ```
>
> ## What We Won't Accept
>
> - Self-promotion without substance — your project needs real users or a clear unique value.
> - Duplicate entries — check if something similar already exists in the list.
> - Paid products without free tiers — unless they're exceptionally useful and clearly labeled.
> - Tools that only work with the legacy Codex API (2021-2023) — this list is for Codex CLI.
>
> ## Categories
>
> Add entries to the most specific category. If no category fits, propose a new one in your PR description.
>
> ## Star Badges
>
> Star badges are powered by [shields.io](https://shields.io) and update automatically. Use the `flat-square` style for consistency.

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Directly related to Codex CLI, with specific Codex integration | **partially satisfied** | berth's README names Codex among supported agents and the repo ships an agent skill read by Codex, but berth has no Codex-specific integration code (no `AGENTS.md` deep integration, no Codex plugin manifest). A strict reviewer could call it a general dev tool. |
| Actively maintained | satisfied | Commits on the day of writing. |
| Clear one-sentence description | satisfied | Draft in §3. |
| Include `flat-square` star badge | satisfiable | Draft in §3 — will render as **1 star**. |
| Self-promotion without substance | **risk** | "your project needs real users or a clear unique value" — berth has 1 star and no users yet. |
| Not a duplicate | satisfiable | Closest existing entries are `standardagents/dmux` ("isolated tmux pane + Git worktree per task") and `mixpeek/amux` in `## Session & Workflow Management`. |
| **List merges external entries** | **NOT satisfied** | Zero merged PRs among the 100 most recently updated closed PRs (§Activity verdict). |

**Verdict: DO NOT SUBMIT — low value.** Even though the written rules are satisfiable, the maintenance pattern shows contributions are not merged; effort spent here (including the "open an issue with why it's awesome" route) has very low expected return.

## 3. Exact entry text (for completeness; do not submit)

```md
- [Mrjwj34/berth](https://github.com/Mrjwj34/berth) - Daemonless local workspaces for parallel coding agents, one Git worktree and one private port set per workspace. ![GitHub stars](https://img.shields.io/github/stars/Mrjwj34/berth?style=flat-square)
```

## 4. Exact insertion point (if it were submitted)

- **File:** `README.md`
- **Section:** `## Session & Workflow Management` (heading at line 278 of the README fetched 2026-09-11). The section is **append-order, not alphabetical**.
- **Position:** the section's current last line is

```md
- [Dicklesworthstone/coding_agent_session_search](https://github.com/Dicklesworthstone/coding_agent_session_search) - Unified TUI and CLI to index and search local coding agent session history across 11+ providers including Codex. ![GitHub stars](https://img.shields.io/github/stars/Dicklesworthstone/coding_agent_session_search?style=flat-square)
```

(line 300); insert after it, before `## Model Providers & Proxies` (line 302).
Alternative sections that exist: `## Docker & Sandboxing` (line 370), `## Cross-Agent Tools` (line 338).

## 5. PR title

No convention documented. If submitted, follow the merged-style pattern used by the maintainer's own commits, e.g. `Add berth to Session & Workflow Management`.

## 6. Submission steps

Not recommended. For completeness: either fork → edit `README.md` → PR, or open an issue with "the link and a brief description of why it's awesome". Both routes currently sit unmerged.

## 7. Predicted rejection risk

**High and quietly so.** The most likely outcome is not rejection but **indefinite non-review**: 187 open issues, a large set of unmerged contributor PRs, and no merged entry PR found in the last 100 closed PRs. Secondary risk if reviewed: "self-promotion without substance — your project needs real users", given 1 star.

## 8. Unverified / caveats

- Whether older PRs (beyond the last 100 closed, or older than the ones returned) were ever merged — the zero-merge finding is bounded by the API window I queried, though the window is large (100 PRs).
- Whether the maintainer adds entries by direct commit from issue suggestions (which would make the "open an issue" route viable). The pushes and README growth are consistent with direct commits, but I did not diff the README history to prove it.
