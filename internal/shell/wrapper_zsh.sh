# gwt shell integration (zsh)
#
# A process cannot change its parent shell's working directory, so the switch
# verbs (new|from|co|checkout|search|pick and the bare dashboard) write the
# chosen worktree path to a temp file (GWT_PATH_OUT). This avoids capturing
# stdout in command substitution, which breaks Cursor setup (uv/pnpm) progress.
#
# _GWT_BIN is set by shell-init to the binary that generated this script.
#
# Install with:  eval "$(/path/to/gwt shell-init zsh)"
_gwt_cd_from() {
  local pathfile content
  pathfile="$(mktemp "${TMPDIR:-/tmp}/gwt-path.XXXXXX")" || return 1
  GWT_PATH_OUT="$pathfile" "$_GWT_BIN" "$@" || { rm -f "$pathfile"; return; }
  [[ -f "$pathfile" ]] || return 0
  content="$(<"$pathfile")"
  rm -f "$pathfile"
  [[ -z "$content" ]] && return 0
  if [[ "$content" == GWT_POPULATE:* ]]; then
    print -z -- "${content#GWT_POPULATE:}"
  else
    builtin cd -- "${content%%$'\n'*}"
    if [[ -n ${GWT_AUTO_LS:-} ]]; then
      "$_GWT_BIN" ls
    fi
  fi
}

gwt() {
  case "$1" in
    new|from|co|checkout|search|pick|dashboard)
      local a
      for a in "$@"; do
        if [[ "$a" == -h || "$a" == --help ]]; then
          "$_GWT_BIN" "$@"
          return
        fi
      done
      _gwt_cd_from "$@"
      ;;
    "")
      _gwt_cd_from
      ;;
    *)
      "$_GWT_BIN" "$@"
      ;;
  esac
}
