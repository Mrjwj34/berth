# berth — distribution listings research

Submission-ready material for listing `berth` (https://github.com/Mrjwj34/berth) on high-signal
directories and marketplaces.

**All repository facts in this folder were fetched on 2026-09-11 (UTC).** Star counts, merge dates and
README line numbers drift; re-check before pasting.

## The fact that shapes everything

| Project fact | Value | Why it matters |
|---|---|---|
| GitHub stars | **1** | Kills every list with a star floor (awesome-cli-apps >20, JackyST0 ≥64, awesome-claude-code ≥100, awesome-generative-ai main list ≥1,000). |
| First commit | **2026-09-11T17:14:52Z** | Kills every list with an age floor (awesome-go 5 months, awesome-cli-apps 3 months, awesome-claude-code 14 days). |
| Commits / releases | 22 commits, one tag `v0.1.0` | SemVer tag exists; history does not. |
| License | MIT | Satisfies every list that requires an OSS license. |
| CI | `.github/workflows/ci.yml`, `release.yml` present | Satisfies "CI configured" checks. |
| pkg.go.dev | `https://pkg.go.dev/github.com/Mrjwj34/berth` → **HTTP 404** | Blocking CI check on awesome-go. |
| Public Go API | none — only `cmd/berth` (`package main`) plus `internal/*` | awesome-go's "pkg.go.dev doc comments for public APIs" standard is awkward even later. |
| Missing for plugin marketplaces | no `.codex-plugin/plugin.json`, no icon, no `SECURITY.md` | Blocks `hashgraph-online/awesome-codex-plugins`. |

## Ranked summary (by expected passive inbound value)

| # | Channel | ROI | Eligible? | Effort | Blocking requirement |
|---|---|---|---|---|---|
| 1 | [davila7/claude-code-templates](claude-code-templates-davila7.md) (30,588★) | high | **yes, now** | low (1 `SKILL.md`) | none — merges external skill PRs (last: 2026-09-11) |
| 2 | [Jenqyang/Awesome-AI-Agents](awesome-ai-agents-jenqyang.md) (1,234★) | medium | **yes, now** | low (1 line) | none — no age or star rule; merges external entries daily |
| 3 | [steven2358/awesome-generative-ai → DISCOVERIES.md](awesome-generative-ai-steven2358.md) (12,624★) | medium | **yes, now** (Discoveries only) | low (1 line) | none for Discoveries; main list needs ≥1,000 followers |
| 4 | [hesreallyhim/awesome-claude-code](awesome-claude-code.md) (53,883★) | high | **not yet — 2026-09-25** | low (issue form) | 14-day-since-first-commit rule; "specific to Claude Code" is a required checkbox |
| 5 | [VoltAgent/awesome-agent-skills](awesome-agent-skills-voltagent.md) (34,098★) | medium | **no** | low (1 line) | "Skill must have real community usage … brand new skills … are not accepted" |
| 6 | [kyrolabs/awesome-agents](awesome-agents-kyrolabs.md) (2,808★) | medium | **no** | low (1 line) | "brand new repo without demonstrated traction" — applied automatically |
| 7 | [JackyST0/awesome-agent-skills](awesome-agent-skills-jackyst0.md) (635★) | low | **no** | low (2 lines) | ≥64 GitHub Stars for community submissions |
| 8 | [agarrharr/awesome-cli-apps](awesome-cli-apps.md) (20,369★) | high | **no** | medium (human-written PR) | bot-enforced: >20 stars **and** repo >3 months (on/after 2026-12-11); AI-generated PRs unwelcome |
| 9 | [avelino/awesome-go](awesome-go.md) (183,827★) | very high | **no** | high | ≥5 months of history (on/after 2027-02-11) **plus** pkg.go.dev reachable (currently 404), Go Report Card A- or better, and a coverage-report link |
| 10 | [hashgraph-online/awesome-codex-plugins](awesome-codex-plugins-hashgraph.md) (990★) | low-medium | **no as-is** | high (repo prep) | needs `.codex-plugin/plugin.json`, `assets/icon.svg`, `SECURITY.md`, and the HOL scanner in berth's CI scoring ≥80/130 |

## Recommended order of operations

1. **Today (2026-09-11–12) — three submissions, all eligible with no gate:**
   1. `davila7/claude-code-templates` — add `cli-tool/components/skills/development/berth/SKILL.md` (one file, merged same-day for the last comparable PR).
   2. `Jenqyang/Awesome-AI-Agents` — append one line to `### Tools`; no age or star rule exists there.
   3. `steven2358/awesome-generative-ai` — one line in `DISCOVERIES.md` → `### Developer tools`. Skip the main list (1,000-follower bar).
2. **2026-09-25 — `hesreallyhim/awesome-claude-code`.** Use the browser issue form (`.../issues/new?template=recommend-resource.yml`); never a PR, never `gh`. Decide honestly about the required "This resource is specific to Claude Code" checkbox — berth is agent-agnostic.
3. **After the first stars/usage appear, re-test the traction-gated lists** in this order: `kyrolabs/awesome-agents` (2.8k★, no numeric bar, just "demonstrated traction"), then `VoltAgent/awesome-agent-skills` (34k★, "real community usage"), then `JackyST0/awesome-agent-skills` at 64★.
4. **2026-12-11 or later — `agarrharr/awesome-cli-apps`**, once berth is >3 months old *and* >20 stars; the PR must be written by a human (the repo ships an `AGENTS.md` telling agents never to open PRs, and a bot closes PRs that quote it).
5. **2027-02-11 or later — `avelino/awesome-go`**, and only after producing the three artifacts its CI demands: a reachable pkg.go.dev page, a Go Report Card grade of A- or better, and a coverage-report link in berth's README.
6. **Only if berth decides to ship a Codex plugin bundle** — `hashgraph-online/awesome-codex-plugins`. The manifest, icon, `SECURITY.md` and scanner-CI work happen inside the berth repo, not here.

Do not run steps 2–6 early "to get in the queue": every one of those channels has an automated or explicitly stated gate and an early submission burns the project's first impression with that maintainer.

## Channels I recommend NOT submitting to (with reasons)

| Channel | Stars | Reason not to submit |
|---|---|---|
| [e2b-dev/awesome-ai-agents](awesome-ai-agents-e2b.md) | 29,955 | **Out of scope and not merging.** README says "This list is only for AI assistants and agents" and redirects tools/SDKs to a sister list; most recent third-party entry merged **2026-07-09**; only 3 merges in the last 100 closed PRs; 1,017 open issues. |
| [RoggeOhta/awesome-codex-cli](awesome-codex-cli-roggeohta.md) | 515 | **No merged entry PR found at all** in the 100 most recently updated closed PRs; 187 open issues; entries appear to be added by direct commit. |
| `composio-community/awesome-codex-skills` | 16,383 | **Stalled.** Last merged entry PR `#160` "Add Taisly Agent Kit" on **2026-07-26**, and its last push is the same day; 211 open issues. |
| `Kikobeats/awesome-cli` | 352 | **Dead for entries.** Most recent merged PR is **2021-12-11**; last push 2026-01-24. |
| `k4m4/terminals-are-sexy` | 13,114 | **Dormant.** Last push **2024-07-26**; no merged PRs among the last 100 closed. Also scope is terminals/plugins, not dev environments. |
| `devtoolsd/awesome-devtools` | 675 | **Dormant.** Last push **2025-10-12** (~11 months). The competing moimikey/athivvat variants are smaller or stale. |
| `libukai/awesome-agent-skills` | 5,085 | **No external submission path demonstrated.** Active, but every recent merge (`#148`, `#146`, `#145`, `#119`, `#118`) is authored by the maintainer `libukai`; it reads as a curated Chinese-language guide. |
| `521xueweihan/HelloGitHub` | 176,046 | Big Chinese audience and berth ships a zh-CN README, but the repo's most recent merged PRs are from **2021–2023**; content is compiled editorially into monthly issues and recommended through `hellogithub.com/periodical`, not through PRs. Verified path: none found. |
| `sanjeed5/awesome-cursor-rules-mdc` | 3,572 | **Out of scope.** Catalogues `.mdc` Cursor rule files only; berth ships a Cursor worktree *hook adapter*, not rules. Last push 2026-05-19. |

## Method

- Repository metadata, star counts, merge dates and merged-PR titles came from the GitHub REST API
  (`repos/{owner}/{repo}` and `repos/{owner}/{repo}/pulls?state=closed&sort=updated&direction=desc`)
  via an authenticated `gh` session, read-only.
- Rules were quoted verbatim from each list's own `CONTRIBUTING.md` / `contributing.md` / README and,
  where present, from its PR template, issue form (`recommend-resource.yml`) and `auto-close-prs` script.
- Entry text and neighbours were taken from the list READMEs downloaded on 2026-09-11 with
  `raw.githubusercontent.com`, then located by heading and line number; each channel file names the
  exact neighbours.
- This research pass ran no write-scoped git command, forked nothing and opened no pull request or
  issue; it produced material for a human to submit. A later, separately authorised pass executed the
  three eligible-now submissions — see **Submitted** below.

## Could not be verified (listed explicitly, per channel files)

1. **berth's Go Report Card grade** — `goreportcard.com/report/github.com/Mrjwj34/berth` returns HTTP 200 but the grade is rendered client-side; the repo carries no badge. awesome-go requires A- or better.
2. **berth's test coverage** — not measured; no coverage service link or badge exists in the repo.
3. **Whether pkg.go.dev will index `github.com/Mrjwj34/berth/cmd/berth`** — the root module page 404s today; a `cmd`-only module may never produce the library page awesome-go's docs standard assumes.
4. **Whether the awesome-claude-code validation bot accepts a repo whose activity is concentrated in its first two days** — CONTRIBUTING's "additional commits after the first day" is met, but the validator was not executed.
5. **Whether berth would score ≥80/130 on the HOL plugin scanner** — cannot be run without a `.codex-plugin/plugin.json`.
6. **How the dormant lists actually admit entries** — e2b's Google Form, HelloGitHub's periodical submission, and possible direct-commit curation on `RoggeOhta/awesome-codex-cli` were identified but not exercised, so "low value" rests on the observable merged-PR record plus push dates.
7. **Whether maintainers would consider berth in-scope** where scope is qualitative: kyrolabs ("related to Agentic frameworks"), VoltAgent/JackyST0 (skill-ecosystem relevance), awesome-go (a catalogue of Go software for a CLI with no public library API).
8. **Whether the star-based gates will be met on the dates given** — star counts are point-in-time (1 star on 2026-09-11) and the age-based dates assume the repo stays public and active.

## Submitted

Three PRs were opened on 2026-09-11 under the `Mrjwj34` account, one per channel, for exactly the three channels marked **eligible now**. Each was prepared from a fork branch pinned to the target's current default-branch head, and each diff re-verified through the API afterwards.

| Channel | PR | Date | Diff | Status |
|---|---|---|---|---|
| [davila7/claude-code-templates](claude-code-templates-davila7.md) | https://github.com/davila7/claude-code-templates/pull/884 | 2026-09-11 | 3 added files, +280 / −0: `cli-tool/components/skills/development/berth/SKILL.md`, `references/berth-yaml.md`, `references/recovery.md` | open, mergeable |
| [Jenqyang/Awesome-AI-Agents](awesome-ai-agents-jenqyang.md) | https://github.com/Jenqyang/Awesome-AI-Agents/pull/485 | 2026-09-11 | `README.md` +1 / −0: one entry line appended to `### Tools` | open |
| [steven2358/awesome-generative-ai](awesome-generative-ai-steven2358.md) | https://github.com/steven2358/awesome-generative-ai/pull/1360 | 2026-09-11 | `DISCOVERIES.md` +1 / −0: one entry line appended to `### Developer tools` | open, mergeable |

No channel was skipped, and no channel outside these three was touched. Channels 4–10 in the ranked table remain unsubmitted because none is eligible yet (date gates from 2026-09-25 onwards, star gates, or missing repository artifacts).

### Accuracy note on the contributed skill

The skill file was changed in exactly one place before submission. Its `## Harnesses` paragraph began:

> This skill installs once, to `.agents/skills/berth/`, which Cursor, Codex, pi and Antigravity all read.

That is accurate for berth's own installer and for those four harnesses, but inside a Claude-Code-distributed directory it would have implied something untrue. The contributed file therefore reads:

> berth is driven from the shell, so any harness that can run commands can use it. berth's own installer writes this same file to `.agents/skills/berth/`, which Cursor, Codex, pi and Antigravity read.

No claim is made anywhere — in the skill, in the reference files, or in any PR body — that Claude Code auto-discovers `.agents/skills`. Verified by grep: zero occurrences of "claude" across all three contributed files.

### Post-merge follow-ups

- If PR #884 is merged, note that the commit is authored by an account with repository-creation-date 2026-09-11; the maintainer may still ask for a different category directory (`git/`, `operations/` and `utilities/` are the plausible alternatives to `development/`).
- If PR #485 is merged, the rendered line will carry a `1 star` badge, because the CONTRIBUTING-mandated format embeds a shields.io star badge.
- If PR #1360 is merged, it lands on `DISCOVERIES.md` only; the main list still requires 1,000 followers.
