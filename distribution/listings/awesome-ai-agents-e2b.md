# e2b-dev/awesome-ai-agents

- **URL:** https://github.com/e2b-dev/awesome-ai-agents
- **Stars:** 29,955 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** license `NOASSERTION`, default branch `main`, `pushed_at` 2026-08-21T18:52:45Z, **1,017 open issues / PRs**, PR creation policy `all`.
- **Activity verdict: LOW VALUE — effectively not merging community entries.**
  Merged PRs found in the 100 most recently updated closed PRs (`pulls?state=closed&per_page=100&sort=updated`):
  - `#1419` "docs: drop legacy ref params and move docs links to docs.e2b.dev" by `michael-e2b` (maintainer) — merged **2026-08-21T18:52:43Z** — documentation maintenance, not an entry.
  - `#123` "Adding PraisonAI Low Code and Code AI Agents Framework" by `MervinPraison` — merged **2026-07-09T17:41:49Z** — the most recent *third-party entry* merge I could find.
  - `#1223` "docs(readme): add UTM tracking to e2b.dev links" — merged 2026-07-09T17:41:47Z.
  Everything else in the recent window is unmerged: `#1429` "Add Persona", `#1548` "Test PR (will close)", `#1529` "Add Symbio to the list", `#1511` "Add ENZO — self-hosted BYOK AI workspace", `#1423` "Add Xenon - Terminal AI coding agent", `#1431` "Add OpenOutreach" — all `merged_at = null`.
  Only **3 merged PRs** in the whole 100-PR window, two of them maintainer-internal. The most recent merged *add* entry is **over 2 months old** (2026-07-09) — past the 6-week mark and approaching the 6-month cut-off used by this project's brief.
- **Project facts used:** `Mrjwj34/berth` — a local developer-environment CLI with **1 star**.

## 1. Quoted eligibility rules (verbatim from `README.md`)

> ## Have anything to add?
> Create a pull request or fill in this [form](https://forms.gle/UXQFCogLYrPFvfoUA). Please keep the alphabetical order and in the correct category.
>
> For adding AI agents'-related SDKs, frameworks and tools, please visit [Awesome SDKs for AI Agents](https://github.com/e2b-dev/awesome-sdks-for-ai-agents). **This list is only for AI assistants and agents.**

There is **no `CONTRIBUTING.md`** in the repository (raw fetch of `CONTRIBUTING.md` → HTTP 404). The README above is the entire contribution policy. Entries are full multi-line blocks rather than one-liners:

```md
## [Blinky](https://github.com/seahyinghang8/blinky)
An open-source AI debugging agent for VSCode

<details>

![Banner](...)

### Category
Coding, Debugging

### Description
- Blinky is an open-source AI debugging agent ...

### Links
- [VSCode Extension](...)
- [GitHub](https://github.com/seahyinghang8/blinky)
</details>
```

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason |
|---|---|---|
| "Create a pull request" | satisfied (mechanically) | PRs from the public are allowed (policy `all`). |
| Keep alphabetical order | satisfiable | `berth` would slot between `BeeBot` and `Blinky` in the B run. |
| Correct category | **NOT satisfied** | The list's own scope line is *"This list is only for AI assistants and agents"*, and tooling is explicitly redirected to their sister list `e2b-dev/awesome-sdks-for-ai-agents`. `berth` is a local workspace manager, not an assistant or an agent. |
| "Merges external entries" (project rule) | **NOT satisfied** | Most recent third-party entry merge is **2026-07-09**; only 3 merges in the last 100 closed PRs. Title says **low value**. |

**Verdict: DO NOT SUBMIT. Out of scope and not merging.** Two independent reasons: (1) scope — the maintainers state tools belong on another list, and the entries here are products/agents, not developer environments; (2) low value — the list has not merged an outside entry in over two months and has 1,017 open issues/PRs.

## 3. Exact entry text (for completeness only — do not submit)

The list's format is a block, not a line, so a "character-for-character" entry would be:

```md
## [berth](https://github.com/Mrjwj34/berth)
Daemonless local workspaces for parallel coding agents

<details>

### Category
Coding, Development Environment

### Description
- berth provisions one linked Git worktree, a private data directory, atomically reserved ports, and supervised processes per workspace, with no background daemon.
- It runs natively by default and optionally in a reusable per-workspace Linux container.

### Links
- [GitHub](https://github.com/Mrjwj34/berth)
- [Docs](https://mrjwj34.github.io/berth/)
</details>
```

## 4. Exact insertion point (if it were submitted)

- **File:** `README.md`
- **Position:** alphabetical in the open-source-project run. It would go between:

```md
## [BeeBot](https://github.com/AutoPackAI/beebot)
```
(lines 715–729) and

```md
## [Blinky](https://github.com/seahyinghang8/blinky)
```
(line 735), i.e. immediately before `## [Blinky]`.

## 5. PR title

No convention documented. Merged third-party example: `Adding PraisonAI Low Code and Code AI Agents Framework`. If it were submitted, `Add berth` would match house style.

## 6. Submission steps

Not recommended. For completeness: fork, add the §3 block in §4's position, open a PR (or use their Google Form `https://forms.gle/UXQFCogLYrPFvfoUA`). Note the repo requires **commit sign-off** (`web_commit_signoff_required: true`), so commits need `Signed-off-by:`.

## 7. Predicted rejection risk

**Certain / no-op.** Either the PR goes into a 1,000+ item backlog and is never reviewed, or a maintainer points at the scope line and the sister "SDKs, frameworks and tools" list. Spending effort here has near-zero expected value.

## 8. Unverified / caveats

- I inspected the 100 most recently *updated* closed PRs, not the complete PR history. The conclusion "low value" rests on (a) no third-party entry merge since 2026-07-09, (b) `pushed_at` 2026-08-21, and (c) 1,017 open issues/PRs. It is conceivable that older PRs are merged slowly; I did not page further back.
- Star count of the sister repo `e2b-dev/awesome-sdks-for-ai-agents` was not measured; if the maintainers redirect there, that list is unassessed by me.
