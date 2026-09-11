# hesreallyhim/awesome-claude-code

- **URL:** https://github.com/hesreallyhim/awesome-claude-code
- **Stars:** 53,883 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** license `NOASSERTION`, default branch `main`, `pushed_at` 2026-09-11T18:11:25Z, **1,028 open issues**, **PR creation policy: `collaborators_only`**, `web_commit_signoff_required: true`.
- **Activity verdict: ALIVE and curated daily.**
  Merged PRs are authored by the maintainer (who converts issue-form submissions into his own PRs):
  - `#2804` "Backfilling" — merged **2026-09-10T05:20:28Z**
  - `#2797` "Tooling batch etc" — merged 2026-09-09T20:35:02Z
  - `#2785` "Add resource: Netresearch Agentic Skills" — merged **2026-09-08T22:24:14Z**
  - `#2781` "Add resource: OSS Autopilot" — merged 2026-09-08T21:44:58Z
  - `#2761` "Add resource: OrcaReplay" — merged 2026-09-07T04:44:11Z
  - `#2740` "Add resource: claude-intercom" — merged 2026-09-05T00:20:07Z
  - `#2703` "Add resource: faf-cli" — merged 2026-09-02T05:12:00Z
- **Project facts used:** `Mrjwj34/berth` — public, MIT, **first commit 2026-09-11T17:14:52Z**, **1 star**.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> ## GROUND RULES:
>
> Any resource that is recommended must either:
>
> (i) Be at least 14 days old (14 days since first commit on default branch) AND show signs of active development (I expect there to be also additional commits after the first day);
>
> OR
>
> (ii) Have at least 100 stars.
>
> In addition: **You may not recommend more than one resource at a time.**
>
> Resources that fail these criteria will be closed automatically.

> Although many awesome resources are inter-operable, we especially welcome and invite recommendations of resources that focus on the unique features and functionality of Claude Code. This is not a hard requirement but it is a guideline.

> ## How to Recommend a Resource
>
> **NOTE: ALL RECOMMENDATIONS MUST BE MADE USING THE WEB UI ISSUE FORM TEMPLATE, OR YOU RISK BEING RESTRICTED FROM INTERACTING WITH THIS REPOSITORY TEMPORARILY.**
>
> ### **[Click here to submit a new resource](https://github.com/hesreallyhim/awesome-claude-code/issues/new?template=recommend-resource.yml)**
>
> Do not open a PR. Just fill out the form. If there are any issues with the form, the bot will notify you.
>
> > [!Warning]
> > It is **not** possible to submit a resource recommendation using the `gh` CLI.
>
> Although resources themselves may be partially or entirely written by a coding agent, resource recommendations must be created by human beings.

> - **STYLE:** Resource descriptions should be written as _descriptions_ - not a sales pitch. Don't address the reader ("Don't you hate it when Claude etc.") - state what the software does. Keep it formatted to one line. Don't use any emojis.

> - I am very grateful to receive recommendations from the visitors to this list. But be aware that there is no formal submission/review process at the moment. ... Recommendations are reviewed in a best-effort way, and no guarantee is made as to whether you will receive a response.
> - Bear in mind that the point of an Awesome List is to be *selective* - I cannot recommend every single resource that is submitted.

The issue form `.github/ISSUE_TEMPLATE/recommend-resource.yml` also requires two checkboxes verbatim:

> - "I have visited this repo before with my own eyes, and I have confirmed that this resource is sufficiently distinct from any existing resource"
> - "This resource is specific to Claude Code"
> - "I promise I actually did these things and not doing so is shameful and lazy"
> - "Do not check the following box - leave it unchecked. By checking this box, I admit that I am not reading any of these statements."

and states the same gate inside the form:

> To encourage people to refrain from recommending projects that have not quite matured, the recommended resource must meet one of these conditions:
>
> (a) at least 14 days of active development since the first commit to the default branch;
> OR
> (b) at least 100 stars.

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| (i) ≥14 days old **and** active development past day one | **NOT satisfied yet — satisfied on 2026-09-25** | First commit 2026-09-11T17:14:52Z → 14 days lands on **2026-09-25**. "Additional commits after the first day" is already true (22 commits, several on 2026-09-12 local time). |
| (ii) ≥100 stars | **NOT satisfied** | berth has **1 star**. |
| One resource at a time | satisfied | Single submission planned. |
| Must be submitted via the web-UI issue form (not a PR, not `gh`) | satisfied (planned) | A human will use the browser form. |
| Recommendation created by a human being | satisfied (planned) | Must be the human maintainer, not an agent. |
| Description: one line, descriptive, no reader address, no emojis | satisfied | Draft below. |
| "This resource is specific to Claude Code" (required checkbox) | **questionable** | berth is deliberately agent-agnostic (README targets Claude Code, Codex, Cursor, Antigravity; it ships `.agents/skills/berth/SKILL.md` plus `berth hook install cursor`). The maintainer softens this in CONTRIBUTING ("not a hard requirement but it is a guideline"), but the issue form makes it a **required** checkbox. This is the honest-attestation risk of this channel. |
| Distinct from existing resources | satisfiable | No git-worktree/port-isolation tool appears in `## Infrastructure & DevOps`; nearest neighbours are `aicontainer` and `Brood Box`, which are container/VM sandboxes, not daemonless native workspaces. |

**Verdict: DO NOT SUBMIT TODAY; eligible from 2026-09-25 via rule (i).** Highest-authority channel in this set (53.9k stars, curated daily), and the gate is only a date — but the required "specific to Claude Code" checkbox is a real friction point given berth's agent-agnostic positioning.

## 3. Exact entry text (paste-ready for the list, once accepted)

The maintainer writes the README line himself from the form, then appends an auto-generated badge block. The line he would write follows the section's existing format `- [Name](url) by [Author](author-url) - Description.`:

```md
- [berth](https://github.com/Mrjwj34/berth) by [Mrjwj34](https://github.com/Mrjwj34) - Provisions a Git worktree, a private data directory, dynamically reserved ports, and supervised processes per workspace so several coding agents can work in one repository without a daemon or a container. Native execution is the default; an optional reusable per-workspace Linux container backend is available.  
<img src="https://img.shields.io/github/created-at/Mrjwj34/berth?style=flat-square&labelColor=2b2b2b&color=6b6b6b" alt="created">&nbsp;&nbsp;<img src="https://img.shields.io/github/last-commit/Mrjwj34/berth?style=flat-square&labelColor=2b2b2b&color=6b6b6b" alt="last-commit">&nbsp;&nbsp;<img src="https://img.shields.io/github/license/Mrjwj34/berth?style=flat-square&labelColor=2b2b2b&color=6b6b6b" alt="license">&nbsp;&nbsp;<img src="https://img.shields.io/github/stars/Mrjwj34/berth?style=flat-square&labelColor=2b2b2b&color=6b6b6b" alt="stars">
```

**What you actually paste is the issue-form field values, not this line:**

- **Display Name:** `berth`
- **Category:** `Infrastructure & DevOps`
- **Link:** `https://github.com/Mrjwj34/berth`
- **Author Name:** `Mrjwj34`
- **Author Link:** `https://github.com/Mrjwj34`
- **Description** (1–3 sentences, 10–500 chars, no reader address, no emojis):

```
Provisions a Git worktree, a private data directory, dynamically reserved ports, and supervised processes per workspace, so several coding agents can work in one repository without a daemon and without virtualization. Native execution is the default; an optional reusable per-workspace Linux container backend is available.
```

- **Issue title** (pre-filled by the template, edit only the placeholder): `[Resource]: berth`

## 4. Exact insertion point (in the published list)

- **File:** `README.md`
- **Section:** `## Infrastructure & DevOps` (heading at line 367 of the README fetched 2026-09-11). The section **is** sorted alphabetically.
- **Position:** `berth` sorts first in the section, so it goes **immediately after the `## Infrastructure & DevOps` heading**, i.e. before the current first entry:

```md
- [cc-devops-skills](https://github.com/akin-ozer/cc-devops-skills) by [akin-ozer](https://github.com/akin-ozer) - Immensely detailed set of skills for DevOps Engineers ...
```

There is no entry before it inside the section; the preceding content is the tail of `## Creative Media` and a blank line.
Alternative category from the form's dropdown, if the maintainer prefers: `Agent Orchestration` or `Multi-Purpose`. `Infrastructure & DevOps` is the closest match and is what the form's dropdown offers.

## 5. PR title

**None — PRs are not accepted** (`pull_request_creation_policy: collaborators_only`, and CONTRIBUTING says "Do not open a PR"). The unit of submission is an **issue** whose title the template pre-fills:

```
[Resource]: berth
```

## 6. Submission steps

1. **Wait until on/after 2026-09-25** so rule (i) is met (and ideally have more than one day's worth of development visible).
2. In a browser, signed in as a human account, open:
   `https://github.com/hesreallyhim/awesome-claude-code/issues/new?template=recommend-resource.yml`
   Do **not** use `gh` (explicitly impossible/forbidden) and do **not** open a PR.
3. Fill the form with the values in §3; tick the required checkboxes truthfully. The last checkbox must be left **unchecked**.
4. Submit. A validation bot comments on the issue; a maintainer then decides. There is "no formal submission/review process" and no response guarantee.
5. If accepted, the maintainer re-writes the entry in his own PR; you may then add the badge from §5's "Badges" section of CONTRIBUTING to berth's README.

## 7. Predicted rejection risk

**Moderate.** Most likely rejection reason: **"This resource is specific to Claude Code" is not cleanly true** — berth is agent-agnostic and its README leads with Claude Code, Codex, Cursor and Antigravity together. Second most likely: the maintainer's stated selectivity plus "no guarantee is made as to whether you will receive a response" — i.e. silent non-inclusion. Third: submitting before the 14-day mark, which is closed automatically.

## 8. Unverified / caveats

- Whether the bot's automatic validation accepts a 14-day-old repo whose only activity was on day one — CONTRIBUTING says "I expect there to be also additional commits after the first day"; berth does have day-2 commits, but I could not test the validator.
- The repository requires commit sign-off (`web_commit_signoff_required: true`); irrelevant for an issue-form submission, but relevant if anything is ever turned into a PR.
- I could not determine which of the 1,028 open issues were form submissions rejected by the bot versus accepted.
