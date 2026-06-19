# gwt shell integration (bash)
#
# A process cannot change its parent shell's working directory, so the switch
# verbs (new|from|co|checkout|search|pick and the bare dashboard) print the
# chosen worktree path to stdout. This wrapper captures that path and cd's
# there. Everything else (diagnostics, prompts, the TUI) is written to the tty
# by gwt itself, so it is left untouched.
#
# If gwt instead prints a line beginning with "GWT_POPULATE:" (used by from/co
# with no argument to suggest a command for review), the remainder is placed
# in the readline buffer when possible, otherwise printed as a ready-to-run
# line and pushed onto the history.
#
# Install with:  eval "$(gwt shell-init bash)"
gwt() {
  case "$1" in
    new|from|co|checkout|search|pick|dashboard|"")
      local out
      out="$(command gwt "$@")" || return
      if [[ -z "$out" ]]; then
        return
      fi
      if [[ "$out" == GWT_POPULATE:* ]]; then
        local cmd="${out#GWT_POPULATE:}"
        history -s -- "$cmd"
        if [[ -n "${READLINE_LINE+x}" ]]; then
          READLINE_LINE="$cmd"
          READLINE_POINT=${#cmd}
        else
          printf '%s\n' "$cmd" >&2
        fi
      else
        builtin cd -- "${out##*$'\n'}" && command gwt ls
      fi
      ;;
    *)
      command gwt "$@"
      ;;
  esac
}
