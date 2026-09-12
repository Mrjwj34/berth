# Releasing

Every release must be reproducible from this document alone. It exists because
the tap, the bucket and the winget manifest all pin a sha256 of an artifact that
only exists after CI has run.

## Before tagging

`main` must be green: the `ci` workflow runs `gofmt`, `go vet`, the unit tests,
the race tests, `staticcheck`, the build, and both acceptance tests (native and
container). A release cut from a red `main` ships a binary nobody has executed.

Behaviour changes require documentation changes in the same commit — the
readmes, `docs/features.md`, the site pages and the agent skill all describe the
CLI surface.

## 1. Tag

```sh
git switch main && git pull
git tag -a v0.3.0 -m "berth v0.3.0

<one paragraph: what changed for a user>"
git push origin v0.3.0
```

The annotated message becomes part of the release page, and release pages are
indexed and linked from the package manifests, so write it for a stranger.

## 2. Watch the release workflow

```sh
gh run watch "$(gh run list --workflow=release --limit=1 --json databaseId --jq '.[0].databaseId')"
```

`release.yml` builds six targets, writes `checksums.txt`, creates the GitHub
release with generated notes, and then appends the install block. That last step
is `continue-on-error: true`, so verify it actually landed:

```sh
gh release view v0.3.0 --json body --jq .body | tail -20
gh release view v0.3.0 --json assets --jq '.assets[].name'
```

Six archives plus `checksums.txt` are expected. A missing target means the
platform build broke and the manifests must not be rendered yet.

## 3. Render the package manifests

```sh
python distribution/render.py --tag v0.3.0
```

The script refuses to write anything unless all six assets are present with
well-formed sha256 digests, so a partial release cannot produce a formula that
installs half a tool.

## 4. Verify before publishing

Never copy a hash by hand. Confirm the manifests match the published archives:

```sh
tmp="$(mktemp -d)"
gh release download v0.3.0 --dir "$tmp"
for f in "$tmp"/berth_*; do sha256sum "$f"; done | sort
git diff --stat distribution/
```

The values in `distribution/homebrew/berth.rb`, `distribution/scoop/berth.json`
and `distribution/winget/*.installer.yaml` must equal the hashes of the
downloaded files. Only then continue.

## 5. Publish to the tap and the bucket

**This step is automated.** Both repositories run `.github/workflows/update.yml`,
which reads the latest release's `checksums.txt`, renders their manifest again and
commits only when the published hashes differ. They run daily and on demand:

```sh
gh workflow run update --repo Mrjwj34/homebrew-tap
gh workflow run update --repo Mrjwj34/scoop-bucket
```

Nothing needs to happen after a release; the recipes below are for the case where
the automation is broken and a manifest has to be corrected by hand.

### Manual fallback

Both repositories are written through the contents API so no clone is needed.
Get the current blob sha first, then send the new file:

```sh
sha=$(gh api repos/Mrjwj34/homebrew-tap/contents/Formula/berth.rb --jq .sha)
content=$(base64 -w0 distribution/homebrew/berth.rb)   # -w0 is GNU base64
gh api -X PUT repos/Mrjwj34/homebrew-tap/contents/Formula/berth.rb \
  -f message="berth 0.3.0" -f content="$content" -f sha="$sha"

sha=$(gh api repos/Mrjwj34/scoop-bucket/contents/bucket/berth.json --jq .sha)
content=$(base64 -w0 distribution/scoop/berth.json)
gh api -X PUT repos/Mrjwj34/scoop-bucket/contents/bucket/berth.json \
  -f message="berth 0.3.0" -f content="$content" -f sha="$sha"
```

On Windows, `base64 -w0` is unavailable; read the file, normalise to LF and
encode it, or push with a temporary clone instead. Publish LF content: the
scripts write LF explicitly so `brew` and `scoop` read the same bytes on every
platform.

## 6. winget

**This step is manual**: one pull request per version. `wingetcreate update
Mrjwj34.berth --version <version> --urls <url> --submit` does it in one command
when run by the account that owns the package, and the API calls in this runbook's
history do the same thing by hand. The first submission additionally needs a
signed Contributor License Agreement; later versions do not.

Superseded submissions are closed rather than left queued: a version that was
never merged and has been replaced by a newer release should not stay open
(`Mrjwj34.berth` 0.2.0 was closed when 0.3.0 was submitted).

### Manual fallback

The package exists as `Mrjwj34.berth` in `microsoft/winget-pkgs`. Bump the three
manifests under `manifests/m/Mrjwj34/berth/<version>/` in a fork, with the pull
request titled `New version/Update: Mrjwj34.berth version <version>`.

Before submitting, validate locally — this is free and catches the mistakes the
review pipeline would:

```sh
winget validate --manifest distribution/winget
```

The manual route described below needs a signed Contributor License Agreement; later version
bumps do not.

## 7. After publishing

```sh
tmp="$(mktemp -d)" && GOPATH="$tmp" go install github.com/Mrjwj34/berth/cmd/berth@latest
"$tmp/bin/berth" version
```

`proxy.golang.org` and `pkg.go.dev` pick the tag up on their own; `go install`
from a clean `GOPATH` is the only thing worth confirming, because it is the
first line of the readme.

## Periodically

- **Dependencies**: Dependabot opens weekly grouped pull requests for Go modules
  and Actions. Merging them is cheap; letting them pile up is not.
- **Listings**: the eligibility floors in `distribution/listings/README.md` have
  dates. Re-check them when they pass rather than re-asking the same questions.
- **Agent matrix**: `docs/agent-matrix.md` names paths owned by other projects.
  When a harness changes its skill location, the matrix is what tells us, so
  re-verify it at each minor release.

## Never

- Never hand-edit a sha256 in a manifest: render it or do not publish it.
- Never delete a published release or tag; the module proxy has already cached
  it, and the manifests of older versions point at its archives.
- Never tag from a dirty tree or from a branch that is not `main`.
