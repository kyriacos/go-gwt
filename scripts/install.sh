#!/usr/bin/env bash
# Install gwt to $(go env GOPATH)/bin (or GOBIN) and print shell-init refresh steps.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"

(cd "$root" && go install ./cmd/gwt)

gobin="$(go env GOBIN)"
gopath="$(go env GOPATH)"
if [[ -n "$gobin" ]]; then
	bin="$gobin/gwt"
else
	bin="$gopath/bin/gwt"
fi

if [[ ! -x "$bin" ]]; then
	echo "install failed: $bin not found" >&2
	exit 1
fi

echo "Installed: $bin"
"$bin" version
echo

stale="$HOME/.local/bin/gwt"
if [[ -x "$stale" ]] && ! cmp -s "$stale" "$bin" 2>/dev/null; then
	echo "WARNING: $stale differs from $bin and is often first on PATH." >&2
	echo "  That breaks shell integration (setup hangs after y). Move it aside:" >&2
	echo "    mv $stale ${stale}.bak" >&2
	echo >&2
fi

echo "Next: refresh shell integration (required after every install)."
echo "Use the installed binary path — do not run bare 'gwt shell-init' if another gwt is on PATH."
echo
echo "  eval \"\$($bin shell-init zsh)\""
echo
echo "Or run: $bin doctor"
