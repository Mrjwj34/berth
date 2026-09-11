# JackyST0/awesome-agent-skills

- **URL:** https://github.com/JackyST0/awesome-agent-skills
- **Stars:** 635 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** CC0-1.0, default branch `main`, `pushed_at` 2026-09-09T01:53:37Z, only 4 open issues, PR creation policy `all`; files include `CONTRIBUTING.md`, `README.md`, `README_ZH.md`, `.github/PULL_REQUEST_TEMPLATE.md`, `.github/ISSUE_TEMPLATE.md`.
- **Activity verdict: ALIVE, merges third-party "Add X" PRs.**
  Merged entry PRs:
  - `#86` "Add planning-with-files to Productivity" by `OthmanAdi` — merged **2026-09-09T01:53:25Z**
  - `#89` "Add d1v DevOps skill" by `aboutmydreams` — merged **2026-09-08T09:47:45Z**
  - `#71` "Add suede-creator-skills collection" by `JasonColapietro` — merged 2026-08-18T03:15:08Z
  - `#68` "Add wiki to Productivity" by `ryanpettry` — merged 2026-08-18T01:47:26Z
  - `#69` "Add OrkasVideoStudio skills collection" — merged 2026-08-18T01:46:07Z
- **Project facts used:** `Mrjwj34/berth` — **1 star**, created 2026-09-11.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> #### General Requirements
>
> - [ ] Entry must be publicly accessible
> - [ ] Description must be clear and accurate
> - [ ] Link must be valid
> - [ ] Must update both `README.md` and `README_ZH.md` (keep bilingual READMEs in sync)
> - [ ] Please verify formatting before submitting — use the existing entry format for the target category and do not break the document structure
> - [ ] Prefer existing categories; do not create a standalone section for a single project
>
> #### By Entry Type
>
> - [ ] Community submissions should have at least **64 GitHub Stars** by default; maintainers may make explicit exceptions for special cases
> - [ ] Single Skill repositories: must contain a `SKILL.md` file and meet the community stars threshold
> - [ ] Skills collections / managers / installers: must clearly serve skill discovery, installation, sync, distribution, or management; `SKILL.md` at the repo root is not required, but the repository must meet the community stars threshold
> - [ ] Official Resources: may be exempt from the stars threshold, but must be official projects, official docs, or widely recognized ecosystem infrastructure

> #### Skill Entry Format
>
> ```markdown
> | Name | Description | Platform | Link |
> |------|-------------|----------|------|
> | my-skill | Short description of what it does (10-20 words) | All | [Link](https://github.com/...) |
> ```

> #### Platform Labels
>
> - `All` - Supports all platforms
> - `Cursor` - Cursor only
> - `Claude` - Claude only …
> - `Codex` - Codex only …

> ### PR Template
>
> ```markdown
> ## Add Entry
>
> **Name**:
> **Link**:
> **Description**:
> **Category**:
> **Type**: Single Skill / Skills Collection / Manager / Installer / Official Resource
>
> ## Checklist
>
> - [ ] Link is valid
> - [ ] Updated both README.md and README_ZH.md
> - [ ] Description is accurate
> - [ ] Placed in correct category
> - [ ] Did not create a standalone section for a single project
> - [ ] Preserves the category's existing meaningful order; do not reorder the historical list solely for alphabetization
> - [ ] Single Skill repository contains SKILL.md
> - [ ] Skills collections / managers / installers clearly serve the skills ecosystem
> - [ ] Repository meets the minimum threshold (64+ Stars for community projects)
> ```

**Format inconsistency to be aware of:** CONTRIBUTING mandates a **table** row. In the live files, `README_ZH.md` uses tables while `README.md` uses plain bullets in the same sections. Example, `## DevOps`:

- `README.md` (line 185+): `- [devops-claude-skills](https://github.com/ahmedasmar/devops-claude-skills) - DevOps workflow marketplace with Terraform/K8s.`
- `README_ZH.md` (line 239+): `| devops-claude-skills | DevOps 工作流市场，含 Terraform/K8s | Claude | [GitHub](https://github.com/ahmedasmar/devops-claude-skills) |`

CONTRIBUTING says "use the existing entry format for the target category", and the two files disagree, so both forms are provided below.

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Publicly accessible | satisfied | Public GitHub repo with MIT license. |
| Clear, accurate description | satisfied | Drafts below. |
| Valid link | satisfied | Both repo root and skill sub-path resolve. |
| Update both `README.md` and `README_ZH.md` | satisfied (planned) | berth already ships `README.zh-CN.md`, so an accurate Chinese description is easy. |
| Existing category, no new section | satisfied | `## DevOps` in both files (or `## 开发工具` / "Development Tools"). |
| Category order preserved, no global re-sort | satisfied | Append at the end of the category, matching what merged PRs did. |
| Entry type fits | satisfied | "Skills collections / managers / installers" — berth ships a skill and manages workspaces for skills; `SKILL.md` is present at `.agents/skills/berth/SKILL.md`. |
| Description 10–20 words | satisfied | Drafts below. |
| **"Community submissions should have at least 64 GitHub Stars"** | **NOT satisfied** | berth has **1 star**. The threshold section explicitly applies to *both* "Single Skill repositories" and "Skills collections / managers / installers". The only escape is "maintainers may make explicit exceptions for special cases" — not something to plan around. |

**Verdict: DO NOT SUBMIT TODAY — blocked on the 64-star threshold.** Everything else is satisfied and the list is responsive (merges within days). Come back once berth has ≥64 stars.

## 3. Exact entry text (paste-ready)

**For `README.md`** (bullet form, matching neighbours; append at the end of `## DevOps`):

```md
- [berth](https://github.com/Mrjwj34/berth) - Daemonless local workspaces for parallel coding agents with per-workspace Git worktrees, private data, reserved ports, and supervised processes.
```

**For `README_ZH.md`** (table form, matching neighbours; append at the end of `## DevOps`):

```md
| berth | 面向并行编码 Agent 的无守护进程本地工作区，每个工作区独立 Git worktree、私有数据、预留端口与受管进程 | All | [GitHub](https://github.com/Mrjwj34/berth) |
```

## 4. Exact insertion point

- **Files:** `README.md` and `README_ZH.md` (both must be updated in the same PR).
- **Section:** `## DevOps` in both files. (`README.md` heading line 185, `README_ZH.md` heading line 239 in the copies fetched 2026-09-11. `## 开发工具` / `## Development Tools` at `README.md` line 139 is the plausible alternative if a reviewer prefers it.)
- **Position:** last row/bullet of the category, i.e.
  - `README.md` — after the current last line of the section:
    ```md
    - [d1v](https://github.com/d1vai/d1v-cli/blob/main/skills/d1v/SKILL.md) - Deploy web projects with verified previews and explicit-confirmation production releases.
    ```
    (line 192), before the blank line and `## Data Processing` (line 194).
  - `README_ZH.md` — after the current last row of the section:
    ```md
    | d1v | 部署 Web 项目，提供可验证预览和需明确确认的生产发布 | Claude/Codex | [GitHub](https://github.com/d1vai/d1v-cli/blob/main/skills/d1v/SKILL.md) |
    ```
    (line 248), before the blank line and `## 数据处理` (line 250).

## 5. PR title

No title convention is documented; merged titles are `Add <Name> to <Section>`. Recommended:

```
Add berth to DevOps
```

## 6. Submission steps

1. Fork `JackyST0/awesome-agent-skills`, branch from `main`.
2. Append the §3 lines to `README.md` and `README_ZH.md` in the same commit — the bilingual requirement is a hard checklist item.
3. Open the PR using the repo's PR template (`## Add Entry` with Name / Link / Description / Category / Type plus the checklist). Note the type as **Skills Collection / Manager / Installer** and that the repository currently has fewer than 64 stars, if asking for an exception.
4. Title: `Add berth to DevOps`.

## 7. Predicted rejection risk

**High today** — the single likely rejection reason is the **64-star minimum**, which CONTRIBUTING states applies to community submissions of every type. There is no age rule, so once the star threshold is met the remaining risks are minor: forgetting to update `README_ZH.md`, or a reviewer preferring the table format from CONTRIBUTING where the file uses bullets.

## 8. Unverified / caveats

- Whether the maintainer would count berth as a "Skills Collection / Manager / Installer" or as something outside the skills ecosystem's scope; the category rules don't mention developer-environment tooling explicitly.
- The stated exception mechanism ("maintainers may make explicit exceptions for special cases") has no documented criteria; I did not find a case where it was applied.
- Whether the discrepancy between the bullet format in `README.md` and the table format in `README_ZH.md` is intentional or drift — I report both forms rather than guessing.
