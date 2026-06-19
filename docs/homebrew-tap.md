# Setting up the Homebrew tap

`gwt` ships a Homebrew formula on every tagged release via
[goreleaser](https://goreleaser.com). goreleaser does not publish to
Homebrew core; it pushes a formula into a tap repository you own. This is a
one-time setup.

## What a "tap" is

A tap is just a public GitHub repo named `homebrew-<name>`. Homebrew maps
`brew install <user>/<name>/<formula>` to the repo `github.com/<user>/homebrew-<name>`
and looks for `Formula/<formula>.rb` inside it. For this project:

| Thing | Value |
|---|---|
| Tap repo | `kyriacos/homebrew-tap` |
| Install command | `brew install kyriacos/tap/gwt` |
| Formula path goreleaser writes | `Formula/gwt.rb` |

The `.goreleaser.yaml` in this repo already points its `brews` block at
`kyriacos/homebrew-tap`.

## One-time setup

### 1. Create the tap repository

```sh
gh repo create kyriacos/homebrew-tap --public \
  --description "Homebrew formulae for kyriacos's tools"
```

It can start empty — goreleaser creates `Formula/gwt.rb` on the first release.

### 2. Create a token that can push to the tap

The default `GITHUB_TOKEN` in Actions is scoped to the repo running the
workflow, so it cannot push to a *different* repo (the tap). Create a
fine-grained Personal Access Token instead:

1. GitHub → Settings → Developer settings → Fine-grained tokens → Generate.
2. Repository access: only `kyriacos/homebrew-tap`.
3. Permissions: Repository → **Contents: Read and write**.
4. Generate and copy the token.

### 3. Add the token as a secret on this repo

```sh
gh secret set HOMEBREW_TAP_TOKEN --repo kyriacos/go-gwt
# paste the token when prompted
```

The release workflow (`.github/workflows/release.yml`) passes this secret to
goreleaser as the credential for the tap push.

## Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```

The tag triggers `.github/workflows/release.yml`, which runs goreleaser to:

- build macOS and Linux binaries (amd64 + arm64),
- attach archives and checksums to the GitHub Release,
- write/update `Formula/gwt.rb` in `kyriacos/homebrew-tap`.

After that completes:

```sh
brew install kyriacos/tap/gwt
```

## Verifying locally before tagging

```sh
goreleaser release --snapshot --clean   # builds everything, publishes nothing
```

Inspect `dist/` for the archives and the rendered formula.
