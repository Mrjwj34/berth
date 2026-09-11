# steven2358/awesome-generative-ai (Discoveries list)

- **URL:** https://github.com/steven2358/awesome-generative-ai — target file: `DISCOVERIES.md`
- **Stars:** 12,624 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** CC0-1.0, default branch `main`, `pushed_at` 2026-09-09T00:01:42Z, 654 open issues, PR creation policy `all`.
- **Activity verdict: ALIVE, merges external "Add X" PRs every few days.**
  - `#488` "Add BoTTube to Video section" — merged **2026-09-09T00:01:42Z**
  - `#485` "Add Cortex - AI Memory Extension" — merged **2026-09-06T22:08:06Z**
  - `#482` "Add Cited By AI® CPS® Framework - AI citation optimisation framework" — merged 2026-09-06T21:57:14Z (and it is now in `DISCOVERIES.md`)
  - Not merged at check time: `#1262` "Add Persona to Agents", `#487` "Add Prefactor – AI agent runtime control plane", `#483` "Add MealThinker to Productivity section", `#481` "Add AI Security Testing section with Tessera".
- **Project facts used:** `Mrjwj34/berth` — **1 star**, created 2026-09-11.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> *[Update 5 Sept. 2026: I review each contributed project by hand. Order: FIFO on the PR queue. If you want to jump the queue, get creative, surprise me, think outside of the box, make my day.]*
>
> ## Formatting
>
> - Use the following format: `[ProjectName](Link) - Description.`
> - Open-source projects should include the tag #opensource at the end. If the source lives at another URL, the tag should link to it.
> - Add new entries to the bottom of their respective category.
> - Keep descriptions concise, clear, and straightforward, and end them with a period.
> - New categories or improvements to the existing ones are also welcome.
> - Ensure your text editor is set to remove trailing whitespace.
>
> ## Quality standards
>
> All projects should follow these quality standards:
>
> - Widely used and useful to the community.
> - Actively maintained (even if that simply means addressing open issues).
> - Well-documented.
>
> ## Inclusion criteria for the Main List
>
> A project may be included in the [Main List](…) if it fulfills at least one of the following criteria:
>
> 1. **High general interest and significant followers**: The project should generate considerable interest and have a substantial number of followers (**at least 1,000**)…
> 2. **Personally interesting to the maintainer**…
>
> ⚠️ **If your project does not fulfill any of these criteria, it will be added to the Discoveries list.** If your project fails to meet the quality standards, the pull request will be respectfully rejected.
>
> ## Inclusion in the Discoveries List
>
> The [Discoveries list](DISCOVERIES.md) is a special showcase for the community, celebrating a wide range of fascinating Generative AI projects. It provides a platform to feature projects that **may not meet the inclusion criteria for the main list** but are still valuable, unique, or innovative contributions to the field.
>
> The Discoveries list is an inclusive and vibrant collection of projects that demonstrate the diversity and creativity within the Generative AI community. If you have a project, service, or resource related to Generative AI that you would like to share, feel free to submit it through a [pull request](https://github.com/steven2358/awesome-generative-ai/pulls).

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Main list: ≥1,000 followers | **NOT satisfied** | berth has 1 star → cannot go on the main list. |
| Main list: personally interesting to the maintainer | unknown | Cannot be predicted. |
| **Discoveries list: any generative-AI-adjacent project** | **satisfied** | The Discoveries list is explicitly the destination for projects that fail the main-list bar. It is the correct target for berth. |
| Format `[ProjectName](Link) - Description.` | satisfied | Draft in §3. |
| `#opensource` tag | satisfied | Draft includes it as a bare tag, matching entries whose primary link is already GitHub (e.g. `- [Atlas UI 3](https://github.com/sandialabs/atlas-ui-3) - … #opensource`). |
| Add to the bottom of the category | satisfied | Destination is the end of `## Coding` → `### Developer tools`. |
| Description concise, ends with a period, no trailing whitespace | satisfied | Draft in §3. |
| Widely used and useful | **NOT satisfied (weakened by design)** | "Widely used" is not true for a 1-star project; the Discoveries list exists precisely because that criterion is relaxed, but the maintainer states it is applied by hand and "projects … that [fail] to meet the quality standards" are rejected. This is the main risk. |
| Actively maintained | satisfied | Daily commits. |
| Well documented | satisfied | README (EN/zh-CN), docs site, skill. |

**Verdict: ELIGIBLE NOW, targeting `DISCOVERIES.md`.** This is the only large general-AI list found where the low-star path is an explicit, documented, maintained destination rather than a rejection. Note the queue: the maintainer reviews by hand, FIFO, and several submissions have sat unmerged.

## 3. Exact entry text (paste-ready)

Target file is **`DISCOVERIES.md`** (not `README.md`):

```md
- [berth](https://github.com/Mrjwj34/berth) - Daemonless local workspaces for parallel coding agents: a Git worktree, a private data directory, reserved ports, and supervised processes per workspace. #opensource
```

## 4. Exact insertion point

- **File:** `DISCOVERIES.md`
- **Section:** `## Coding` (line 110) → `### Developer tools` (line 127). Entries are **appended at the bottom of the subcategory** (they are not alphabetical).
- **Position:** after the section's current last line:

```md
- [CPS Framework](https://github.com/citedbyai/cps-framework) - AI-citation-readiness scoring for web content, with a free MCP checker and paid full audits.
```

(line 155 of the DISCOVERIES.md fetched 2026-09-11), before the blank line and `### Playgrounds` (line 157).
Plausible alternative sections in the same file: `## Agents` → `### Autonomous agents` (line 165), where `Maestro` — "Run multiple AI coding agents in parallel with a spec-driven workflow" — already sits. `### Developer tools` is the closer match for a workspace/CLI tool.

## 5. PR title

No title convention documented; merged titles are `Add <Name> to <Section>` (e.g. "Add BoTTube to Video section", "Add Cortex - AI Memory Extension"). Recommended:

```
Add berth to Developer tools (Discoveries)
```

## 6. Submission steps

1. Fork `steven2358/awesome-generative-ai`, branch from `main`.
2. Add the §3 line at the end of `### Developer tools` **inside `DISCOVERIES.md`** — do not touch `README.md` unless deliberately asking for the main list (in which case the CONTRIBUTING says the maintainer will move it to Discoveries anyway).
3. Open a PR. The maintainer reviews "by hand. Order: FIFO on the PR queue" — a short, factual PR body helps; CONTRIBUTING explicitly invites "get creative, surprise me".
4. Expect a hand review; the maintainer may add it to the main list under criterion 2 if it interests him.

## 7. Predicted rejection risk

**Moderate.** Most likely rejection reason: the **"Widely used and useful to the community"** quality standard, applied by hand — a 1-star project on the day it is created can plausibly be judged premature even for Discoveries. Second: FIFO queue latency (the maintainer states he reviews by hand and no response is guaranteed). Third: `## Coding` → `### Developer tools` contains both open-source and commercial entries, so placement is not the issue.

## 8. Unverified / caveats

- Whether the maintainer treats "let it mature first" as an implicit rule — the Discoveries criteria are qualitative, not numeric, beyond the main-list 1,000-follower bar.
- The main list's own README was not diffed against `DISCOVERIES.md` to check for entries that were later promoted; I cannot predict promotion.

## Submitted

- **PR:** https://github.com/steven2358/awesome-generative-ai/pull/1360 — opened 2026-09-11, title `Add berth to Developer tools`, branch `Mrjwj34:add-berth` created at `main` @ `349dbec71dd13f82dd6697fc283564cf22acfb1e` (unchanged at verification time).
- **Diff:** `DISCOVERIES.md` +1 / −0 — exactly the intended entry line, appended after `CPS Framework` at the end of `### Developer tools`, before `### Playgrounds`. Verified from the PR patch: one file changed, one line added, nothing else. `README.md` was deliberately not touched, so no request was made for the 1,000-follower main list.
- The file was re-fetched immediately before editing and the insertion point re-derived dynamically: the `### Developer tools` heading was still at line 127, the next heading at 157, last entry at 155 — matching §4. The committed entry is character-identical to §3, including the `#opensource` tag.
- **Status at 2026-09-11T18:55Z:** open, `mergeable: true`, 1 commit. PR body is a two-line summary plus the repo link.
