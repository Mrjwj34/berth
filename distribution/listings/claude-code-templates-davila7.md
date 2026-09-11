# davila7/claude-code-templates (aitmpl.com directory)

- **URL:** https://github.com/davila7/claude-code-templates (browseable directory at `https://aitmpl.com`, docs at `https://docs.aitmpl.com`)
- **Stars:** 30,588 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** MIT, default branch `main`, `pushed_at` 2026-09-11T14:11:58Z, 241 open issues, PR creation policy `all`; ships `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`, and a `.github/CODEOWNERS`.
- **Activity verdict: ALIVE, merges third-party component PRs (including skills) on the day of submission.**
  Merged PRs:
  - `#880` "Add eval-genius skill" by `alexgr-agent` (external) — merged **2026-09-11T12:52:40Z**
  - `#882` "improve: enhance mlops-engineer" — merged 2026-09-11T13:59:58Z
  - `#878` "improve: enhance business-analyst" — merged 2026-09-10T13:59:44Z
  - `#853` "Add Upstash Redis and Ratelimit skills" — **not merged** at check time
  - `#876`, `#875`, `#874` — merged 2026-09-08/09
  `#880` added exactly one file: `cli-tool/components/skills/development/eval-genius/SKILL.md` — the same shape of contribution berth would make.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> ## 🧩 Contributing Components
>
> The easiest way to contribute is by adding individual components like agents, commands, MCPs, settings, or hooks.
>
> ### 🤖 Adding Agents
>
> 1. **Create Agent File**
>    ```bash
>    # Navigate to appropriate category
>    cd cli-tool/components/agents/[category]/
>    # Create your agent file
>    touch your-agent-name.md
>    ```
> 2. **Agent File Structure** … 3. **Available Categories** … 4. **Creating New Categories** …

> ### ⚙️ Adding Settings … ### 🪝 Adding Hooks …

> ## 📦 Contributing Templates
>
> ### Template Quality Standards
>
> - **Comprehensive Configuration** - Include all necessary Claude Code setup
> - **Clear Documentation** - Well-documented CLAUDE.md with examples
> - **Practical Commands** - Useful slash commands for the domain
> - **Proper MCPs** - Relevant external integrations
> - **Testing** - Test template with real projects

> ## 🤝 Contribution Process
>
> ### 1. Fork and Clone … ### 2. Create Feature Branch (`git checkout -b feature/your-contribution`) … ### 3. Make Changes … ### 4. Test Changes … ### 5. Submit Pull Request
> - Clear description of changes
> - Screenshots for UI changes
> - Testing instructions
> - Reference related issues

> ## 🎯 What We're Looking For
>
> ### High Priority Components
> - **Security Agents** … **Performance Commands** … **Cloud MCPs** … **Framework Agents** …

> ## 📄 License
>
> By contributing to this project, you agree that your contributions will be licensed under the MIT License.

**Important gap:** `CONTRIBUTING.md` documents `agents/`, `commands/`, `mcps/`, `settings/`, `hooks/` and `templates/` — it does **not** document `skills/`. However the repository now contains `cli-tool/components/skills/<category>/<name>/SKILL.md` (228 directories under `skills/development/` alone) and the maintainer merged an external skill PR (`#880`) on 2026-09-11, so the skill path is live and accepted in practice even though it is undocumented. **There is no stated star minimum, no age minimum and no self-promotion ban.**

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Fork + feature branch + PR | satisfied (planned) | Standard GitHub flow, documented in CONTRIBUTING §Contribution Process. |
| Clear description / testing instructions in the PR | satisfied | Instructions can cite `npx claude-code-templates@latest … --dry-run` and the SKILL.md content. |
| Component must live in `cli-tool/components/…` | satisfied | berth's skill is contributed as `cli-tool/components/skills/development/berth/SKILL.md`. |
| MIT-licensing of the contribution | satisfied | berth is MIT; CONTRIBUTING states contributions are licensed MIT. |
| Documentation and a usable artifact | satisfied | berth has `SKILL.md` with YAML frontmatter, README (EN/zh-CN), docs site. |
| Star / age minimum | **no such rule** | None stated anywhere in CONTRIBUTING. |
| Skills path documented in CONTRIBUTING | **NOT satisfied (documentation gap)** | The word "skills" never appears as a contribution type; the path is used in the repo but undocumented. Risk of a reviewer asking for a different location. |
| Priority alignment ("What We're Looking For") | partially satisfied | The high-priority lists are security/performance/cloud/framework components. A dev-environment skill is not on the priority list, but the directory accepts a broad range (backend-architect, bash-pro, git, operations, utilities…). |

**Verdict: ELIGIBLE NOW.** No numeric gate; the maintainer merges external skill PRs (last one 2026-09-11). This is the best *eligible-today* skill-directory channel in this set by reach (30.6k stars plus the `aitmpl.com` browsing UI).

## 3. Exact entry text / exact artifact to add

This channel has **no README line**. The "entry" is a file:

- **Path:** `cli-tool/components/skills/development/berth/SKILL.md`
- **Content:** berth's own `.agents/skills/berth/SKILL.md`, verbatim, with the YAML frontmatter the directory expects. For reference, the merged `eval-genius` skill (PR `#880`) starts with:

```md
---
name: eval-genius
description: >-
  Decide whether an AI/LLM/agent/retrieval system needs an eval, where it fits in the
  dev process, which one to run, and how to read the result; then design, gate, judge,
  and defend it. Not ordinary unit tests.
---

# Eval Genius
```

so berth's file must open with:

```md
---
name: berth
description: >-
  Manage independent local agent workspaces: one linked Git worktree per workspace, a
  private data directory, atomically reserved ports, and supervised processes — for
  parallel coding agents in a single repository.
---
```

then the existing body of `.agents/skills/berth/SKILL.md`.

## 4. Exact insertion point

- **Directory:** `cli-tool/components/skills/development/` (228 subdirectories at check time; directory order is alphabetical and is the list's own ordering).
- **Position:** between the directories `bash-pro` and `best-practices`, i.e. immediately **after** `cli-tool/components/skills/development/bash-pro/` and immediately **before** `cli-tool/components/skills/development/best-practices/`.
- Alternative category directories that exist and could be argued: `git/`, `operations/`, `utilities/`. `development` is the best match for a dev-environment tool and is where `eval-genius`, `bash-pro`, `backend-architect` etc. live.

## 5. PR title

No title convention is documented. Recent merged titles use the plain imperative form (`Add eval-genius skill`, `Add Upstash Redis and Ratelimit skills`). Recommended:

```
Add berth skill
```

## 6. Submission steps

1. Fork `davila7/claude-code-templates`, branch from `main` (CONTRIBUTING suggests `feature/your-contribution`).
2. Add the single directory `cli-tool/components/skills/development/berth/` containing `SKILL.md` (§3). Do not modify anything else.
3. Optionally sanity-check from `cli-tool/`: `npm install`, then exercise the components CLI in `--dry-run` mode as CONTRIBUTING describes (`npx claude-code-templates@latest --agent … --dry-run`; the equivalent for skills is not documented — say so in the PR rather than inventing a flag).
4. Open the PR with a clear description, the license note (MIT), and any testing instructions.
5. Title: `Add berth skill`.

## 7. Predicted rejection risk

**Low-to-moderate.** Most likely rejection reason: the **skills path is not documented in CONTRIBUTING**, so a reviewer may ask for a different directory or template format. Second: `#853` ("Add Upstash Redis and Ratelimit skills") sitting unmerged shows skill PRs are not merged unconditionally. Third: the "What We're Looking For" priorities do not mention developer-environment tooling. None of these is a written rule.

## 8. Unverified / caveats

- Whether `aitmpl.com` (the browsing UI) re-indexes skills automatically after merge — CONTRIBUTING only promises "All contributors are recognized in our … Release Notes".
- Whether a `SKILL.md` frontmatter schema is validated by CI; I saw no CI config for it.
- I did not check whether berth's existing `SKILL.md` mentions only berth's own CLI in a way that conflicts with this directory's Claude-Code-centric framing.

## Submitted

- **PR:** https://github.com/davila7/claude-code-templates/pull/884 — opened 2026-09-11, title `Add berth skill`, branch `Mrjwj34:add-berth-skill` created at `main` @ `c93d4ee1e1a2ffabbc5084ccf4715d10683f4b82` (unchanged at verification time).
- **Diff:** +280 / −0 across exactly 3 added files, nothing else:
  - `cli-tool/components/skills/development/berth/SKILL.md`
  - `cli-tool/components/skills/development/berth/references/berth-yaml.md`
  - `cli-tool/components/skills/development/berth/references/recovery.md`
- Each `references/*.md` file was included so every pointer in `SKILL.md` resolves on disk (multi-file skills already exist here, e.g. `api-design-principles/references/`). This is a superset of §6 step 2, which said to add `SKILL.md` only.
- **One accuracy change made to berth's skill before contributing.** The `## Harnesses` paragraph opened with "This skill installs once, to `.agents/skills/berth/`, which Cursor, Codex, pi and Antigravity all read." That sentence is true only of berth's own installer, and inside a Claude-Code-distributed directory it would have been misleading, so it now reads: "berth is driven from the shell, so any harness that can run commands can use it. berth's own installer writes this same file to `.agents/skills/berth/`, which Cursor, Codex, pi and Antigravity read." Nothing in the contributed files claims Claude Code discovers `.agents/skills` — verified by grep: zero occurrences of "claude" in all three files. The two reference files were contributed verbatim; they contain no install-path or harness claims.
- **Status at 2026-09-11T18:55Z:** open, `mergeable: true`, 1 commit, 3 changed files. PR body is a two-line summary plus the repo link; this repository documents no PR-template requirement.
