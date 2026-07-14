# gwt shell integration (bash)
#
# Switch verbs write the chosen path via GWT_PATH_OUT so stdout is not captured
# in command substitution (which breaks long-running setup scripts).
#
# _GWT_BIN is set by shell-init to the binary that generated this script.
#
# Install with:  eval "$(/path/to/gwt shell-init bash)"
_gwt_cd_from() {
  local pathfile content
  pathfile="$(mktemp "${TMPDIR:-/tmp}/gwt-path.XXXXXX")" || return 1
  GWT_PATH_OUT="$pathfile" "$_GWT_BIN" "$@" || { rm -f "$pathfile"; return; }
  [[ -f "$pathfile" ]] || return 0
  content="$(<"$pathfile")"
  rm -f "$pathfile"
  [[ -z "$content" ]] && return 0
  if [[ "$content" == GWT_POPULATE:* ]]; then
    local cmd="${content#GWT_POPULATE:}"
    history -s -- "$cmd"
    if [[ -n "${READLINE_LINE+x}" ]]; then
      READLINE_LINE="$cmd"
      READLINE_POINT=${#cmd}
    else
      printf '%s\n' "$cmd" >&2
    fi
  else
    builtin cd -- "${content%%$'\n'*}"
    if [[ -n ${GWT_AUTO_LS:-} ]]; then
      "$_GWT_BIN" ls
    fi
  fi
}

gwt() {
  case "$1" in
    new|from|co|checkout|search|pick|dashboard|"")
      local a
      for a in "$@"; do
        if [[ "$a" == -h || "$a" == --help ]]; then
          "$_GWT_BIN" "$@"
          return
        fi
      done
      _gwt_cd_from "$@"
      ;;
    *)
      "$_GWT_BIN" "$@"
      ;;
  esac
}
