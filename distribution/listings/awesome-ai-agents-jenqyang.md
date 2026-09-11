# Jenqyang/Awesome-AI-Agents

- **URL:** https://github.com/Jenqyang/Awesome-AI-Agents
- **Stars:** 1,234 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** no license file detected by GitHub (`license: null`), default branch `main`, `pushed_at` 2026-09-11T05:10:56Z, 28 open issues, PR creation policy `all`.
- **Activity verdict: ALIVE, and it merges external "Add X" PRs in batches almost daily.**
  Merged entry PRs (GitHub API, `merged_at`):
  - `#470` "Add YYLO to Multi-Agent Task Solver Projects" by `InsightFactoryAPP` — merged **2026-09-11T05:10:48Z**
  - `#477` "Add Kapso to Autonomous Agent Task Solver Projects" by `alirezamshi` — merged **2026-09-11T05:10:40Z**
  - `#484` "Add Webcmd to Tools" by `prakhar1605` — merged **2026-09-11T05:10:30Z**
  - `#478` "Add ValetFS to Tools" by `YoungjuneKwon` — merged 2026-09-09T08:33:08Z
  Four external contributions merged within the same minute on 2026-09-11 — this is the single most responsive "Agentic/AI agents" list I found that takes outside entries.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> ## Quality Bar
>
> We prioritize quality over quantity.
>
> - Standard OSS by default: entries should use a clearly open-source license. Source-available, commercial, or mixed-proprietary licensing will usually be rejected.
> - OSS substance over OSS wrapper: an open-source MCP/server/client wrapper is not enough if the core product value depends mainly on a proprietary hosted service or paid backend.
> - Real ecosystem value: the project should materially help the agent ecosystem.
> - Evidence over marketing: avoid hype-heavy or ad-like wording.
> - Maintainability signals: active repo, clear docs, usable code/artifacts.
> - No duplicates: check existing entries before submitting.
>
> Submissions may be closed if they are mostly promotional, out of scope, weakly documented, duplicate existing entries, use non-OSS licensing, or function mainly as marketing for a paid or closed service.
>
> ## Before You Submit
>
> 1. Search the README to ensure the item is not already listed.
> 2. Confirm the best section for your item.
> 3. Ensure the project has a complete README (at least install/usage context).
> 4. Prepare a neutral one-line description.
> 5. Confirm the repository has a standard open-source license clearly visible in GitHub metadata or the repo itself.
> 6. If the project connects to a paid API/service, explain what meaningful value remains usable from the OSS artifact itself without buying the hosted product.
>
> ## Preferred Contribution Path
>
> Open a PR directly when possible.
>
> - One entry per PR is preferred.
> - Keep changes minimal and focused.
> - Put the entry in the most appropriate section.
> - Keep list style consistent with existing lines.
>
> Entry format:
>
> ```md
> - [ProjectName](https://github.com/org/repo) - Neutral one-line description. ![GitHub Repo stars](https://img.shields.io/github/stars/org/repo?style=social)
> ```

PR template (`.github/pull_request_template.md`) requires, among other checks:

> - [ ] I searched `README.md` and confirmed this is not a duplicate.
> - [ ] The submission is relevant to the AI agent ecosystem.
> - [ ] The description is neutral and evidence-based (not promotional).
> - [ ] I removed unverifiable claims (e.g., "best", "first") or provided clear evidence.
> - [ ] If this is open source, license information is clear.
> - [ ] The linked project/resource has usable documentation (README/docs).

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Clearly open-source license visible in GitHub metadata | satisfied | `Mrjwj34/berth` is MIT (`license.spdx_id: MIT`, `LICENSE` in repo root). |
| Not a thin wrapper over a paid/closed service | satisfied | Everything runs locally; no hosted backend, no account, no API key. |
| Real ecosystem value | satisfied | It is infrastructure for running parallel coding agents locally; the list's `### Tools` section already holds equals such as `OrcaReplay`, `Desktop Control`, `agent-coordinator`. |
| Evidence over marketing / neutral language | satisfied | Draft description is factual; no "best"/"first". |
| Maintainability signals (active repo, clear docs) | satisfied | 22 commits in the first 24h, CI + release workflows, README (EN/zh-CN), docs site, `SKILL.md`, MIT. |
| No duplicates | satisfied | `berth` is not in the README (searched the fetched copy). |
| One entry per PR / minimal diff | satisfied | One line appended. |
| Entry format incl. star badge | satisfied (with a caveat) | The CONTRIBUTING-mandated format includes `![GitHub Repo stars](...?style=social)`. A few recent entries omit it (e.g. `- [ax](https://github.com/Necmttn/ax) - Local telemetry for AI coding agents.` at line 185), so omission is tolerated — but the badge is specified, and for berth it will render as **1 star**, which is a visible negative. Recommendation: submit **with** the badge to match CONTRIBUTING. |
| Minimum age / minimum stars | **no such rule** | The CONTRIBUTING has **no** star threshold and **no** age threshold. Unlike kyrolabs (which bans "brand new repo without demonstrated traction"), nothing here blocks a new repository. |
| Correct section | satisfied | `## Applications` → `### Tools` (the section that just took "Add Webcmd to Tools" and "Add ValetFS to Tools"). |

**Verdict: ELIGIBLE NOW — this is the best "AI agents" channel.** No numeric gate applies, the list merges external entries within minutes, and the diff is one line.

## 3. Exact entry text (paste-ready)

```md
- [berth](https://github.com/Mrjwj34/berth) - Daemonless local workspaces for parallel coding agents: one Git worktree per workspace, a private data directory, atomically reserved ports, and supervised processes. ![GitHub Repo stars](https://img.shields.io/github/stars/Mrjwj34/berth?style=social)
```

If the reviewer prefers no badge (matching line 185 of the README), drop the trailing `![GitHub Repo stars](...)` image and keep the rest identical.

## 4. Exact insertion point

- **File:** `README.md`
- **Section:** `## Applications` → `### Tools` (heading at line 146 of the README fetched 2026-09-11). The section is **append-order, not alphabetical**.
- **Position:** append at the **end of `### Tools`** — immediately **after** the section's current last line:

```md
- [Webcmd](https://github.com/agentrhq/webcmd) - Self-learning browser infrastructure for AI agents: learns a site's navigation once, then compiles it into deterministic per-site CLI commands. TypeScript, Apache-2.0. ![GitHub Repo stars](https://img.shields.io/github/stars/agentrhq/webcmd?style=social)
```

(the file's line 199), leaving one blank line before the `## Frameworks` heading at line 201. There is no line after the insertion point inside this section.

## 5. PR title

No title format is documented. Recent merged titles are plain sentences: `Add Webcmd to Tools`, `Add ValetFS to Tools`, `Add YYLO to Multi-Agent Task Solver Projects`. Recommended:

```
Add berth to Tools
```

## 6. Submission steps

1. Fork `Jenqyang/Awesome-AI-Agents`, branch from `main`.
2. Append the §3 line to the end of `### Tools` (§4). Nothing else in the diff.
3. Fill in `.github/pull_request_template.md`:
   - **Summary:** "Adds `berth`, a daemonless local workspace manager for parallel coding agents, to the Tools section."
   - **Type of change:** check `Add new resource entry`.
   - **Target section:** check `Applications / Tools`.
   - Tick every box in **Quality checks (required)** — they are all true for berth (non-duplicate, relevant, neutral, verifiable license, usable docs, no paid backend).
   - **Proposed entry line:** paste the §3 line inside the template's ```md fence.
4. Title: `Add berth to Tools`. Open the PR.

## 7. Predicted rejection risk

**Low.** Most likely rejection reason if any: the reviewers' "Maintainability signals: active repo" check — a repository created the same day may be read as unproven, and the star badge on the line advertises `1`. Secondary risk: a reviewer may see "coding agents" tooling as overflowing the `### Tools` section (which is already long and memory-heavy). Neither is a written rule, so this is the channel to submit first.

## 8. Unverified / caveats

- Whether the same-day merge cadence continues (a batch merge on 2026-09-11 could be a maintainer catching up).
- Whether the maintainer minds the 1-star badge in the rendered line — the CONTRIBUTING mandates the badge but does not discuss low-star entries.

## Submitted

- **PR:** https://github.com/Jenqyang/Awesome-AI-Agents/pull/485 — opened 2026-09-11, title `Add berth to Tools`, branch `Mrjwj34:add-berth` created at `main` @ `7bc5e8134e5ed20ef60757f9502d7058f70821da` (unchanged at verification time).
- **Diff:** `README.md` +1 / −0 — exactly the intended entry line, inserted immediately after the `Webcmd` entry at the end of `### Tools`, before the blank line and `## Frameworks`. Verified from the PR patch: one file changed, one line added, nothing else.
- The file was re-fetched immediately before editing and the insertion point re-derived dynamically (last non-empty line of the section): `### Tools` was still at line 146, the next heading at 201, and the last entry at 199 — i.e. the position in §4 had not shifted. The committed entry is character-identical to §3, including the CONTRIBUTING-mandated star badge.
- **Status at 2026-09-11T18:53Z:** open. The PR body follows the repository's own `pull_request_template.md` (Summary / Type of change / Target section / Quality checks / Proposed entry line), kept terse and non-promotional.
