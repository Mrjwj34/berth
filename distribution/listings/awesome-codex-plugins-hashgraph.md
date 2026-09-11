# hashgraph-online/awesome-codex-plugins

- **URL:** https://github.com/hashgraph-online/awesome-codex-plugins
- **Stars:** 990 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** Apache-2.0, default branch `main`, `pushed_at` 2026-09-11T17:00:28Z, 19 open issues, PR creation policy `all`.
- **Activity verdict: ALIVE — the most responsive channel found; it merged four external entry PRs on the day of checking.**
  - `#412` "Add Magents" — merged **2026-09-11T14:25:42Z**
  - `#411` "Add Fantasy Football Manager" — merged **2026-09-11T14:18:56Z**
  - `#370` "Add Stop That Shit" — merged **2026-09-11T14:14:47Z**
  - `#405` "Add QRStuff plugin to Tools & Integrations" — merged **2026-09-11T14:05:54Z**
  - `#373` "Add PMC project-memory plugin" — merged 2026-08-31T12:06:55Z
- **Project facts used (checked in the berth repo 2026-09-11):** `.codex-plugin/` — **absent**; `assets/` — absent; `SECURITY.md` — **absent**; `.github/workflows/` — `ci.yml`, `release.yml` (no HOL scanner workflow); MIT license present. Repo age: first commit 2026-09-11, **1 star**.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> ## How Submissions Work
>
> You add a single line to `README.md`. That's it. A maintainer-verified generator mirrors your plugin bundle from your source repo and regenerates the catalog files (`plugins.json`, `marketplace.json`).
>
> > **Important: Read this entire guide before opening a PR. Submissions missing required items will be asked to fix them.**

> ### Step 1: Set up scanner CI in your plugin repo (required)
>
> Your plugin repo must have the **HOL AI Plugin Scanner** running in CI before you submit. This is not optional. We verify this during review.
>
> ... `uses: hashgraph-online/ai-plugin-scanner-action@v1` with `min_score: 80`, `fail_on_severity: high` …
>
> ### Step 2: Run the scanner locally and check your score
>
> You need a score of **80/130** or higher with **no critical or high severity findings**. Save the output to include in your PR description.
>
> ### Step 3: Verify your plugin repo has the required files
>
> Your plugin repo must contain:
> - `.codex-plugin/plugin.json` (valid manifest)
> - `SECURITY.md` (vulnerability disclosure policy)
> - `LICENSE` (MIT or Apache-2.0 recommended)
> - `README.md` (clear description)
> - No hardcoded secrets, no dangerous MCP commands
> - SHA-pinned GitHub Actions (if using Actions)
> - Dependency lockfiles (`package-lock.json` or equivalent)

> ### Step 4: Add your entry to README.md and open a PR
>
> 1. **Fork** this repository
> 2. **Add your entry** to the appropriate section in `README.md` (alphabetical order by display name)
> 3. **Submit a PR** with:
>    - Your scanner score (or link to the passing CI run on your plugin repo)
>    - The public GitHub URL of your plugin repo
>
> **Do not copy plugin files, `plugins/` directories, `plugins.json`, or `marketplace.json` into your PR.**

> ## README Entry Format
>
> ```markdown
> - [Plugin Name](https://github.com/<owner>/<repo>) - One-line description of what it does.
> ```
>
> Rules:
> - One plugin per line
> - Alphabetical order within each category
> - Description must be a single sentence
> - Link must point to the GitHub repository root

> ## Plugin Repo Requirements
>
> ```
> your-plugin-repo/
>   .codex-plugin/
>     plugin.json        # Required - plugin manifest
>   assets/
>     icon.svg           # Required - plugin icon (SVG preferred, PNG acceptable)
> ```
>
> **Required fields:** `name`, `version`, `description`, `repository`, `license`, `interface.composerIcon`

> ## Additional Requirements
>
> - Plugin must have a **public GitHub repository**
> - Must be **functional** with a valid `.codex-plugin/plugin.json` manifest
> - Must include an **icon** as described above
> - **Must pass the HOL Plugin Scanner** (score ≥ 80, no critical/high findings)
> - **Must have scanner running in CI** (GitHub Action or equivalent)
> - **One plugin per PR**

> ## Categories
>
> - **Development & Workflow** - Tools for coding, planning, and development workflows
> - **Tools & Integrations** - External service integrations and utilities

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (verified in berth's repo, 2026-09-11) |
|---|---|---|
| Public GitHub repository | satisfied | `Mrjwj34/berth` is public. |
| `LICENSE` present | satisfied | MIT. (Apache-2.0 would also be accepted; MIT is "recommended".) |
| `README.md` clear description | satisfied | Detailed README, EN + zh-CN. |
| No hardcoded secrets / no dangerous MCP commands in a manifest | unknown | Not verified by running their scanner; berth has no manifest to scan yet. |
| `.codex-plugin/plugin.json` valid manifest | **NOT satisfied** | No `.codex-plugin/` directory exists. |
| `assets/icon.svg` (512×512 recommended, <50KB) | **NOT satisfied** | No `assets/` directory; the README logo lives on an external CDN (`cdn.jwjbox.dev/berth.png`). |
| `SECURITY.md` | **NOT satisfied** | Not present at repo root. |
| HOL AI Plugin Scanner running in the plugin repo's CI | **NOT satisfied** | `.github/workflows/` contains only `ci.yml` and `release.yml`. |
| Scanner score ≥80/130, no critical/high findings | **NOT satisfied / unverified** | Cannot be run until the manifest, icon and `SECURITY.md` exist. |
| Dependency lockfiles | n/a | Go project: `go.sum` present, which is the equivalent lockfile; the rule names `package-lock.json` or `requirements-lock.txt`. |
| SHA-pinned GitHub Actions | **unverified** | I did not audit `.github/workflows/*.yml` for `uses:` pinning; this is a scored criterion, not an absolute gate. |
| One plugin per PR, alphabetical, single-sentence description, root link | satisfiable | See §3/§4. |

**Verdict: DO NOT SUBMIT AS-IS — requires repository preparation first.** The channel is genuinely responsive (four external merges on 2026-09-11) and has no star or age minimum, but it is a *Codex plugin* marketplace and the gate is a set of artifacts berth does not have: a `.codex-plugin/plugin.json` manifest, an icon, a `SECURITY.md`, and the HOL scanner running in berth's own CI with a score ≥80/130. That work happens in the berth repository, not in this listings folder, and is therefore out of scope here — the maintainer must decide whether berth wants to ship a Codex plugin bundle at all.

## 3. Exact entry text (paste-ready, once the artifacts exist)

```md
- [berth](https://github.com/Mrjwj34/berth) - Provisions an isolated Git worktree, private data directory, reserved ports, and supervised processes per workspace for parallel coding agents.
```

## 4. Exact insertion point

- **File:** `README.md`
- **Section:** `## Community Plugins` (heading at line 140 of the README fetched 2026-09-11). Entries are **alphabetical by display name**.
- **Position:** between `BABOK Analyst` and `BGS Modding Superpowers` — after:

```md
- [BABOK Analyst](https://github.com/GSkuza/BABOK_ANALYST) - BABOK v3 business analysis agent with 16 MCP tools, a 9-stage pipeline, and human-in-the-loop approval gates.
```

(line 167) and before:

```md
- [BGS Modding Superpowers](https://github.com/BB-84C/bgs-modding-superpowers) - Agentic Bethesda Game Studio modpack curation toolkit ...
```

(line 168). Case-insensitive ordering puts `berth` (b-e) after `BABOK` (b-a) and before `BGS` (b-g).
Alternative category: `## Community Plugins` is the only place third-party additions appear (the other sections are `## Official Plugins`, `## Plugin Development`, `## Guides & Articles`, `## Related Projects`). Category choice between "Development & Workflow" and "Tools & Integrations" is expressed in the PR text, not by a separate README heading.

## 5. PR title

No convention documented; merged titles are plain (`Add Magents`, `Add QRStuff plugin to Tools & Integrations`). Recommended:

```
Add berth
```

## 6. Submission steps

1. **Prerequisite work in the berth repo** (not created by this task): add `.codex-plugin/plugin.json` (with `name`, `version`, `description`, `repository`, `license`, `interface.composerIcon`), `assets/icon.svg`, `SECURITY.md`, and a `.github/workflows/hol-plugin-scanner.yml` using `hashgraph-online/ai-plugin-scanner-action@v1`; wait for it to pass on `main`.
2. Run the scanner locally and capture the score: `pipx install --force "plugin-scanner==3.0.160"` then `plugin-scanner scan . --format text`; require **≥80/130** with no critical/high findings.
3. Fork `hashgraph-online/awesome-codex-plugins`, branch from `main`, add the §3 line at the position in §4.
4. Open the PR including **the scanner score or a link to the passing CI run**, and the public repo URL. Do not commit `plugins/`, `plugins.json` or `marketplace.json`.
5. CI re-validates alphabetical order, fetches the source repo, validates `plugin.json`/icon presence, re-runs the scanner at the documented thresholds, and checks that all README links resolve.

## 7. Predicted rejection risk

**Certain rejection today**, and the reason is unambiguous and mechanical: the CI gate "checks the source repository" and requires `.codex-plugin/plugin.json`, an icon, `SECURITY.md`, and a scanner workflow in berth's CI with a score ≥80 — none of which exist. Once those exist, the residual risk is the scanner threshold plus the "no dangerous commands" rule (berth's docs mention process-compose and container commands, so a scanner reading of `curl | sh` or `sudo` patterns in documentation is a plausible flag).

## 8. Unverified / caveats

- The scanner score berth would obtain — cannot be measured without the prerequisite artifacts.
- Whether the scanner requires `package-lock.json`/`requirements-lock.txt` literally, or accepts Go's `go.sum`; the requirement text names only JavaScript/Python files.
- Whether a Go CLI that is not primarily an MCP/plugin bundle is considered in-scope as a "Codex plugin" even with a valid manifest; several catalogued plugins are local CLIs, which suggests yes, but I could not verify a rule.
