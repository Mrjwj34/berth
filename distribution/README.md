# Distribution

Everything needed to get `berth` into package managers and directories, so the
project can be found without anyone promoting it.

## Layout

| Path | What it is |
| --- | --- |
| `render.py` | Generates the Homebrew, Scoop and winget manifests from a release's `checksums.txt`. |
| `homebrew/berth.rb` | Formula for the `Mrjwj34/tap` tap. |
| `scoop/berth.json` | Manifest for the `Mrjwj34/scoop-bucket` bucket. |
| `winget/` | The three manifests submitted to `microsoft/winget-pkgs`. |
| `listings/` | Ready-to-submit entries for awesome lists and agent marketplaces, with each list's eligibility rules quoted. |

Generated files are committed so the published manifests are always reviewable
in the repository history.

## Releasing to package managers

The release workflow publishes `berth_<version>_<os>_<arch>.tar.gz|zip` plus
`checksums.txt`. Render the manifests from that release — never copy a hash by
hand:

```sh
# after the v* tag finished in Actions
python distribution/render.py --tag v0.2.0
```

The script verifies that all six platform assets exist and that every digest is
a well-formed sha256, then rewrites `homebrew/`, `scoop/` and `winget/`.

### Homebrew tap

```sh
gh repo create Mrjwj34/homebrew-tap --public --description "Homebrew formulae for Mrjwj34 projects"
git -C /tmp clone https://github.com/Mrjwj34/homebrew-tap.git
mkdir -p /tmp/homebrew-tap/Formula
cp distribution/homebrew/berth.rb /tmp/homebrew-tap/Formula/berth.rb
git -C /tmp/homebrew-tap add Formula/berth.rb
git -C /tmp/homebrew-tap commit -m "berth 0.2.0"
git -C /tmp/homebrew-tap push
```

Users then run `brew install Mrjwj34/tap/berth`. Formulae in a tap are not
required to pass `brew audit --strict`, but the formula keeps the layout core
Homebrew expects, so it can be promoted to `homebrew/core` later without a
rewrite.

### Scoop bucket

```sh
gh repo create Mrjwj34/scoop-bucket --public --description "Scoop bucket for Mrjwj34 projects"
# commit scoop/berth.json to bucket/berth.json in that repository
```

Users then run:

```sh
scoop bucket add berth https://github.com/Mrjwj34/scoop-bucket
scoop install berth
```

`checkver`/`autoupdate` are already wired to the GitHub releases, so new
versions can be picked up by `scoop checkver`.

### winget

Copy `winget/*.yaml` to
`manifests/m/Mrjwj34/berth/<version>/` in a fork of `microsoft/winget-pkgs` and
open a pull request titled `New package: Mrjwj34.berth version <version>`. The
package uses a portable zip installer, so it needs no installer switches. The
first submission is reviewed by hand; later versions are automated.

### Go module proxy

Nothing to do: `proxy.golang.org` and `pkg.go.dev` index the repository from the
public tag alone, which is what makes `go install github.com/Mrjwj34/berth/cmd/berth@latest`
work.

## Verifying a render

After rendering, confirm the manifests match the actual release rather than the
script's expectations:

```sh
# the published archives must hash to the values in the formula
gh release download v0.2.0 --pattern 'berth_0.2.0_linux_amd64.tar.gz' --dir /tmp/check
sha256sum /tmp/check/berth_0.2.0_linux_amd64.tar.gz
grep -A1 'linux_amd64' distribution/homebrew/berth.rb
```
