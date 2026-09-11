# kyrolabs/awesome-agents

- **URL:** https://github.com/kyrolabs/awesome-agents
- **Stars:** 2,808 (GitHub REST API, fetched 2026-09-11)
- **Other repo facts:** no license file detected by GitHub, default branch `main`, `pushed_at` 2026-09-08T13:12:45Z, only 2 open issues, PR creation policy `all`.
- **Activity verdict: ALIVE, merges external PRs — but it also auto-closes new-repo submissions.**
  Merged entry PRs:
  - `#758` "Add Busabase to Memory - Knowledge Management" — merged **2026-09-08T13:12:45Z**
  - `#750` "Add Kapso to Research" — merged 2026-09-07T01:09:23Z
  Not merged (still open or closed unmerged) at check time: `#753` "Add OrcaReplay to Testing and Evaluation", `#756` "docs: add Bifrost to frameworks", `#759` "Add BrainPilot to Research", `#757` "Add Atomic Mail (email for AI agents)", `#755` "Add Council of AI (GSPC measurement MCP)".
- **Project facts used:** `Mrjwj34/berth` — created 2026-09-11, **1 star**, brand-new account-adjacent repo.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

> The Awesome Agents curates content and projects using or supporting AI Agents ecosystem. The contribution needs to be open source. The list is curated so that only the best content is included. This means that not all content will be listed. The listed content should be high-quality, demonstrate traction, be maintained, and provide clear added value.
>
> We do not list content that is:
>
> - brand new repo without demonstrated traction.
> - not in English.
> - not related to Agentic frameworks.
> - not maintained anymore.
> - not online anymore.
> - not open source.
> - not adding value to existing content.
>
> When adding a new item, please place it at the _bottom_ of the list.
>
> Submit a PR, not an issue. Any PR that does not follow the guidelines will be automatically closed.
>
> Given the rise of agent submissions, those criteria are non-negotiable, managed and applied automatically. Criteria that most often trigger closing a PR without merging it: brand new repo with no history, brand new user, or wrong place in the list.

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (checks run 2026-09-11) |
|---|---|---|
| Open source | satisfied | MIT. |
| Related to the Agentic ecosystem | satisfied | Workspaces for parallel coding agents; the README's `## Software Development` section already lists `Greywall` (agent sandbox), `h5i` (per-agent git worktree sandboxes), `AgentsMesh` (git worktree isolation), `Paperclip` (agent workspaces). |
| Maintained, online | satisfied | Active commits and releases on the day of writing. |
| High quality / adds value | unknown | Reviewer judgement; berth is not a duplicate of the worktree-orchestrator entries — it is the environment layer underneath them — but a reviewer may see overlap with `AgentsMesh`/`h5i`. |
| English | satisfied | README, docs and skill are English (plus a zh-CN README). |
| Place at the bottom of the list | satisfied (planned) | Append as the last line of `## Software Development`, not alphabetically. |
| Submit a PR, not an issue | satisfied (planned) | PR only. |
| **"brand new repo without demonstrated traction"** | **NOT satisfied** | Repo created 2026-09-11, 22 commits, **1 star**. This is one of the three criteria the maintainer names as most likely to trigger an automatic close. |

**Verdict: DO NOT SUBMIT TODAY.** The disqualifying rule is explicit and the maintainer states it is applied automatically: a brand-new repo with no traction will be closed. Revisit once berth has real adoption signals (stars/downloads/users, ~weeks-to-months of history). Note also that "not related to Agentic frameworks" is the list's own scope line — berth is a developer-environment tool for agents, not an agent framework, which is a secondary scope risk in a list whose sections are Frameworks / Testing / Software Development / Research / Conversational / Game / Memory / Automation.

## 3. Exact entry text (paste-ready)

```md
- [berth](https://github.com/Mrjwj34/berth): Daemonless local workspaces for parallel coding agents using Git worktrees, private data directories, and dynamic port allocation. ![GitHub Repo stars](https://img.shields.io/github/stars/Mrjwj34/berth?style=social)
```

Format matches the dominant style in `## Software Development`: `- [Name](url): Description ![GitHub Repo stars](https://img.shields.io/github/stars/owner/repo?style=social)` (colon separator, star badge on the same line).

## 4. Exact insertion point

- **File:** `readme.md` at the repository root (the repo's root listing uses lowercase `readme.md`; `README.md` resolves on GitHub either way).
- **Section:** `## Software Development` (heading at line 90 of the copy fetched 2026-09-11).
- **Position:** **bottom of the section** — immediately after the current last line:

```md
- [Keen Code](https://github.com/mochow13/keen-code): Open-source, context-aware terminal coding agent written in Go with multiple providers, Turn Memory for controllable cross-turn token retention, skill-driven MCP integration, subagents, Agent Skills, and hashline edits. ![GitHub Repo stars](https://img.shields.io/github/stars/mochow13/keen-code?style=social)
```

(README line 140), leaving one blank line before `## Research` (line 142).
Braces/caveat: `bottom of the **list**` is what the guidelines literally say; the practical reading used by merged PRs (`Add Busabase to Memory - Knowledge Management`, `Add Kapso to Research`) is "bottom of the target section". Bottom of `## Software Development` is therefore correct.

## 5. PR title

No title convention is documented; merged titles are `Add <Name> to <Section>`. Recommended:

```
Add berth to Software Development
```

## 6. Submission steps

1. Fork `kyrolabs/awesome-agents`, branch from `main`.
2. Append the §3 line as the last line of `## Software Development` (§4) — changing nothing else, so the diff cannot trip the "wrong place in the list" auto-close.
3. Open the PR with a short, factual description: what berth is, why it is not a duplicate of `AgentsMesh`/`h5i` (it is the local environment/port/process layer, not a task orchestrator), and the MIT license.
4. Title: `Add berth to Software Development`.

## 7. Predicted rejection risk

**Very high today.** The most likely reason is literally enumerated in the guidelines: **"brand new repo with no history"** — plus "brand new user", if the PR comes from an account with no GitHub history. Expect an automatic close with no discussion. Once berth has traction, the next most likely objection is scope ("not related to Agentic frameworks") or duplication against `AgentsMesh` / `h5i` / `Greywall`.

## 8. Unverified / caveats

- The exact traction threshold the maintainer applies ("demonstrated traction" is not quantified anywhere).
- Whether the automatic close is a bot or manual; the phrase "managed and applied automatically" is ambiguous about the mechanism, but the outcome (closed PRs) is observable in the merged/closed PR data.
