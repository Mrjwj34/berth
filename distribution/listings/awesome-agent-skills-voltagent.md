# VoltAgent/awesome-agent-skills

- **URL:** https://github.com/VoltAgent/awesome-agent-skills
- **Stars:** 34,098 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** MIT, default branch `main`, `pushed_at` 2026-09-07T08:53:43Z, 37 open issues, PR creation policy `all`, root files: `README.md`, `CONTRIBUTING.md`, `LICENSE`, `.gitignore` (no `.github` directory).
- **Activity verdict: ALIVE, merges third-party "Add skill:" PRs.**
  Merged entry PRs:
  - `#974` "Add skill: rebelytics/task-observer" by `rebelytics` — merged **2026-09-05T07:41:33Z**
  - `#994` "Add skill: tt-a1i/archify" by `tt-a1i` — merged 2026-09-06T06:35:39Z
  - `#991` "Add skill: Nanako0129/sepia" by `Nanako0129` — merged 2026-09-05T08:33:53Z
  - `#996` "Add marketing-mindset skill" by `axelfreeman` — merged 2026-09-05T08:33:17Z
  - `#988` "Add Kayforkind/reimagine-it (Content-Derived Design CLI)" by `Kayforkind` — merged 2026-09-05T07:58:21Z
  Not merged at check time: `#1044` "Add skill: zenstory-ai/oh-story-claudecode", `#1030` "Add skill: satan9394/dsh-personal-dev-workflow", `#1027`, `#1026`, `#1003`, `#999`, `#998`.
- **Project facts used:** berth ships `.agents/skills/berth/SKILL.md` (plus `references/berth-yaml.md`, `references/recovery.md`). Repo age: first commit **2026-09-11**, **1 star**.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> ## Adding a Skill
>
> ### Entry Format
>
> Add your skill to the end of the relevant category in `README.md`:
>
> ```markdown
> - **[author/skill-name](https://github.com/author/repo/path)** - Short description of what it does
> ```
>
> ### Where to Add
>
> - **Community skills**: Add to the end of the matching subcategory under "Community Skills" (Marketing, Productivity and Collaboration, Development and Testing, Context Engineering, AI and Data, n8n Automation, or Other).
> - If no existing category fits, add to "Other".
>
> ### Requirements
>
> - Public repository with a working skill
> - Has documentation (README or SKILL.md)
> - Author/org prefix included in the name
> - Description must be short, 10 words or fewer. No lengthy paragraphs.
> - Skill must have real community usage. We focus on community-adopted, proven skills. Brand new skills that were just created are not accepted. Give your skill time to mature and gain users before submitting.
>
> ### PR Title
>
> `Add skill: author/skill-name`
>
> ## Important
>
> - This repository curates links only. Each skill lives in its own repo.
> - Verify your links work before submitting.
> - We review all submissions and may decline skills that don't meet the quality bar.

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Public repository with a working skill | satisfied | `Mrjwj34/berth` is public and `.agents/skills/berth/SKILL.md` exists on `main`. |
| Has documentation (README or SKILL.md) | satisfied | Both; plus a docs site. |
| Author/org prefix in the name | satisfied | Name would be `Mrjwj34/berth`. |
| Description ≤10 words | satisfied | Draft below is 9 words. |
| Link works | satisfied | Skill path is stable on the default branch. |
| **"Skill must have real community usage … Brand new skills that were just created are not accepted."** | **NOT satisfied** | The skill was created 2026-09-11. This is the channel's explicit, subjective gate and it is fatal today. |
| Correct category | satisfied | "Community Skills" → "Development and Testing". |

**Verdict: DO NOT SUBMIT TODAY.** Single blocker: the "real community usage / not brand new" rule. There is no way to satisfy it other than time and adoption. Revisit when the skill has demonstrable users.

## 3. Exact entry text (paste-ready)

```md
- **[Mrjwj34/berth](https://github.com/Mrjwj34/berth/tree/main/.agents/skills/berth)** - Isolated git worktree, private data, reserved ports, supervised processes
```

(9 words in the description, matching the `- **[author/skill-name](url/path)** - description` format, which the list uses everywhere in this section.)

## 4. Exact insertion point

- **File:** `README.md`
- **Section:** `### Community Skills` (line 1701) → the `<details>` block whose summary is `Development and Testing` (line 1794). Entries are **appended**, not sorted.
- **Position:** **end of the `Development and Testing` block** — after its current last line:

```md
- **[scarletkc/agents](https://github.com/scarletkc/agents)** - Reusable standards and workflow skills for AI coding agents
```

(README line 1886 — note it is currently duplicated at line 1881), and **before** the closing `</details>` at line 1888. The next block's summary is `Context Engineering` (line 1891).
Precedent in the same block: `- **[obra/finishing-a-development-branch](https://github.com/obra/superpowers/blob/main/skills/finishing-a-development-branch/SKILL.md)** - Complete Git code branches` — sub-path skill links are accepted.

## 5. PR title

```
Add skill: Mrjwj34/berth
```

## 6. Submission steps

1. Fork `VoltAgent/awesome-agent-skills`, branch from `main`.
2. Append the §3 line at the end of the `Development and Testing` `<details>` block (§4). One-line diff only.
3. Open a PR titled exactly `Add skill: Mrjwj34/berth` with a short body: what the skill does, where it lives, and evidence of usage (this is the field that will be judged against the "real community usage" rule — links to users, stars, downloads or discussion threads help).
4. Their CONTRIBUTING says "We review all submissions and may decline skills that don't meet the quality bar"; no other process is documented.

## 7. Predicted rejection risk

**Near-certain rejection today.** The most likely rejection reason is verbatim in CONTRIBUTING: *"Brand new skills that were just created are not accepted."* A secondary risk is the 34k-star profile of the list: several recent submission PRs (`#1044`, `#1030`, `#1027`, `#1026`, `#1003`) sit unmerged, so even qualifying skills face a queue.

## 8. Unverified / caveats

- What counts as "real community usage" is not defined numerically anywhere in the repo (no star/download threshold given).
- The README's `Development and Testing` block appears to contain entries that belong in other subcategories (e.g. design skills); I report the block's literal end rather than an editorial judgement about where a reviewer would put berth.
- I did not verify whether the skill's `SKILL.md` validates against a schema the maintainer might apply.
