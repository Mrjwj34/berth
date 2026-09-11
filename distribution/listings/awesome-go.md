# avelino/awesome-go

- **URL:** https://github.com/avelino/awesome-go
- **Stars:** 183,827 (GitHub REST API `repos/avelino/awesome-go`, fetched 2026-09-11)
- **Other repo facts:** MIT, default branch `main`, `pushed_at` 2026-09-10T07:37:37Z, 233 open issues, PR creation policy `all`, no sign-off requirement.
- **Activity verdict: ALIVE, merges external "add X" PRs constantly.**
  Most recent merged entry PRs (GitHub API `pulls?state=closed&sort=updated`):
  - `#6680` "Add floatdrop/di to the Dependency Injection section." — merged **2026-09-10T07:19:52Z**
  - `#6679` "Add gnata to the Query Language section." — merged 2026-09-09T16:53:00Z
  - `#6677` "Add mist to the Security section." — merged 2026-09-08T21:27:30Z
  - Commit feed for `README.md` shows 20 entry adds/removes between 2026-09-02 and 2026-09-10.
- **Project facts used:** `Mrjwj34/berth` — public, MIT, created 2026-09-11T09:59:46Z, **1 star**, 22 commits, first commit 2026-09-11T17:14:52Z, one tag `v0.1.0`.

## 1. Quoted eligibility rules (verbatim from `CONTRIBUTING.md`)

From the **Quick checklist**:

> - [ ] One PR adds, removes, or changes **only one item**.
> - [ ] The item is in the **correct category** and in **alphabetical order**.
> - [ ] The link text is the **exact project/package name**.
> - [ ] The description is **concise, non-promotional, and ends with a period**.
> - [ ] The repository has: at least **5 months of history**, an **open source license**, a `go.mod`, and at least one **SemVer release** (`vX.Y.Z`).
> - [ ] Documentation in English: **README** and **pkg.go.dev doc comments** for public APIs.
> - [ ] Tests meet the coverage guideline (**≥80%** for non-data packages, **≥90%** for data packages) when applicable.
> - [ ] Include links in the PR body to **pkg.go.dev**, **Go Report Card**, and a **coverage report**.

From **Quality standards**:

> - have at least 5 months of history since the first commit.
> - have an **open source license**, [see list of allowed licenses](https://opensource.org/licenses/alphabetical);
> - ...
> - if the library/program is testable, then coverage should be >= 80% for non-data-related packages and >=90% for data-related packages. (**Note**: the tests will be reviewed too. We will check your coverage manually if your package's coverage is just a benchmark result);
> - have at least one official version-numbered release that allows go.mod files to list the file by version number of the form vX.X.X.
>
> Categories must have at least 3 items.

From **What is checked automatically** (blocking checks):

> | **Repo accessible** | Repository URL responds and is not archived |
> | **go.mod present** | `go.mod` exists at the repository root |
> | **SemVer release** | At least one tag matching `vX.Y.Z` exists |
> | **pkg.go.dev reachable** | The provided pkg.go.dev link loads |
> | **Go Report Card grade** | Grade is A-, A, or A+ |
> | **PR body links present** | Forge link, pkg.go.dev, and Go Report Card are provided |
> | **Single item per PR** | Only one package added or removed per PR |
> | **Link consistency** | URL added to README matches the forge link in the PR body |
> | **Description format** | Entry ends with a period |
> | **Alphabetical order** | Entry is in the correct alphabetical position |
> | **No duplicate links** | URL is not already in the list |
> | **Entry format** | Matches `- [name](url) - Description.` pattern |
> | **Category minimum** | Category has at least 3 items |

From **Entry formatting rules**:

> ```md
> - [project-name](https://github.com/org/project) - Short, clear description.
> ```

From **Preparing for review**:

> - A link to the project's pkg.go.dev page
> - A link to the project's Go Report Card report
> - A link to a code coverage report

From **How to add an item to the list**:

> Open a pull request against the README.md document that adds the repository to the list.
>
> - The pull request should add one and only one item to the list.
> - The added item should be in alphabetical order within its category.

## 2. Rule-by-rule eligibility judgement

| Rule | Verdict | Reason (all checks run 2026-09-11) |
|---|---|---|
| One item per PR | satisfied | Plan is a single README line. |
| Correct category + alphabetical | satisfied | `## Software Packages` → `### DevOps Tools`; `berth` sorts between `Balerter` and `Blast`. |
| Link text = exact project name | satisfied | Link text will be `berth`; repo is `Mrjwj34/berth`. |
| Concise, non-promotional, ends with a period | satisfied | Draft below has no superlatives and ends with `.` |
| Open source license | satisfied | MIT (`LICENSE` present, GitHub reports `spdx_id: MIT`). |
| `go.mod` at repo root | satisfied | `module github.com/Mrjwj34/berth`, `go 1.25`. |
| At least one SemVer release `vX.Y.Z` | satisfied | Tag `v0.1.0` exists. |
| **≥5 months of history since first commit** | **NOT satisfied** | First commit is **2026-09-11T17:14:52Z**. Five months lands on/after **2027-02-11**. This is a non-blocking CI warning but an explicit quality standard, and reviewers enforce it. |
| pkg.go.dev reachable (blocking CI) | **NOT satisfied** | `https://pkg.go.dev/github.com/Mrjwj34/berth` returned **HTTP 404** when fetched 2026-09-11. |
| pkg.go.dev doc comments for public APIs | unknown / likely moot | The module has **no non-internal library packages** — every package except `cmd/berth` (`package main`) lives under `internal/`. pkg.go.dev would only ever show a command page. |
| Coverage ≥80% in a coverage report | **unverified** | I did not run the test suite; there is no coverage badge or coverage service link in `README.md` (grepped `badge|shields.io|codecov|goreportcard|pkg.go.dev` — only CI, release, Go-version and license badges are present). |
| Go Report Card grade A- or better (blocking CI) | **unverified** | `https://goreportcard.com/report/github.com/Mrjwj34/berth` returns HTTP 200 (page exists) but the grade is rendered client-side and could not be read from the HTML. The repo has zero goreportcard badges. |
| PR body links present | **NOT satisfied yet** | README currently carries no pkg.go.dev, Go Report Card or coverage badge to link to. |
| CI/CD configured | satisfied | `.github/workflows/ci.yml` and `release.yml` exist. |
| Category has ≥3 items | satisfied | `### DevOps Tools` has dozens of entries (e.g. `abbreviate`, `alaz`, `aptly`, …). |
| Maintainer discretionary "generally useful" | satisfied | The section already lists directly comparable runtimes (`colima` "Container runtimes on macOS (and Linux) with minimal setup.", `Den` "Self-hosted sandbox runtime for AI agents. Open-source E2B alternative."). |

**Verdict: DO NOT SUBMIT TODAY.** Two independent hard gates fail: the 5-month history rule and the blocking pkg.go.dev check. Re-attempt **on/after 2027-02-11**, and only after (a) a pkg.go.dev page exists for `github.com/Mrjwj34/berth/cmd/berth`, (b) a Go Report Card grade of A- or better has been confirmed, and (c) a coverage report link exists. Not ready: **do not submit** before those dates/artifacts exist.

## 3. Exact entry text (paste-ready)

```md
- [berth](https://github.com/Mrjwj34/berth) - Daemonless workspaces for parallel coding agents using Git worktrees, private data directories, and dynamic port allocation.
```

## 4. Exact insertion point

- **File:** `README.md`
- **Section:** `## Software Packages` → `### DevOps Tools` (heading at line 3574 of the README fetched 2026-09-11)
- **Position:** alphabetical, between `Balerter` and `Blast` — i.e. insert immediately **after** this line:

```md
- [Balerter](https://github.com/balerter/balerter) - A self-hosted script-based alerting manager.
```

and immediately **before** this line:

```md
- [Blast](https://github.com/dave/blast) - A simple tool for API load testing and batch jobs.
```

Note the file uses ASCII-case-insensitive alphabetical order within the section (`abbreviate`, `alaz`, `aptly`, `aurora`, `aws-doctor`, `awsenv`, `Balerter`, `Blast`, `bombardier`, …), so `berth` belongs after `Balerter` and before `Blast`.
Rejected alternative: `## Version Control` (line 3235) is described as "_Libraries for version control._" and holds libraries and Git clients; `berth` is an environment manager, not a VCS library, so that section would fail the "correct category" check.

## 5. PR title

There is **no documented title convention** in `CONTRIBUTING.md` or the PR template. Maintainer/merged PRs in practice use the pattern `Add <name> to <section>`, e.g. `#6679` "Add gnata to the Query Language section.", `#6677` "Add mist to the Security section.". Recommended:

```
Add berth to DevOps Tools
```

## 6. Submission steps

1. Fork `avelino/awesome-go` (do **not** do this from the berth workspace; the task forbids write-scoped git operations here).
2. Branch off `main`, edit `README.md` only — insert the single line from §3 at the position in §4.
3. Fill in `.github/PULL_REQUEST_TEMPLATE.md` completely, which requires:
   - Forge link: `https://github.com/Mrjwj34/berth`
   - pkg.go.dev: `https://pkg.go.dev/github.com/Mrjwj34/berth/cmd/berth`
   - goreportcard.com: `https://goreportcard.com/report/github.com/Mrjwj34/berth`
   - Coverage service link: *(must exist first)*
   - Delete one of the two "packages around my addition" lines.
4. Open the PR. CI runs the blocking checks; maintainers must approve (`Every PR MUST be reviewed by at least one maintainer before it can get merged.`), and `They will wait 15 days for your interaction, after that the PR will be closed.`
5. Do **not** add a badge to berth's README claiming the listing until the PR is merged.

## 7. Predicted rejection risk

**Very high today; moderate once the gates pass.** Most likely rejection reason: **the repository is one day old and cannot meet the 5-month history quality standard**, and secondarily the **blocking pkg.go.dev check (HTTP 404)** plus a missing coverage-report link. A third realistic objection is the "documentation: pkg.go.dev doc comments for public APIs" standard — berth exposes no public packages, so a reviewer may judge the project a poor fit for a *library-and-software* catalogue even though the `DevOps Tools` section already hosts command-line runtimes.

## 8. Unverified / caveats

- Go Report Card **grade** — could not be read from the client-rendered page; only that the report URL returns HTTP 200.
- Test coverage percentage — not measured; no coverage artifact exists in the repo.
- Star count will drift: **1 star at 2026-09-11**. awesome-go has no star minimum, so this does not block.
