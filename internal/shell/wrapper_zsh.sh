# gwt shell integration (zsh)
#
# A process cannot change its parent shell's working directory, so the switch
# verbs (new|from|co|checkout|search|pick and the bare dashboard) print the
# chosen worktree path to stdout. This wrapper captures that path and cd's
# there. Everything else (diagnostics, prompts, the TUI) is written to the tty
# by gwt itself, so it is left untouched.
#
# If gwt instead prints a line beginning with "GWT_POPULATE:" (used by from/co
# with no argument to suggest a command for review), the remainder is pushed
# into the line editor via `print -z` so you can edit it before running.
#
# Install with:  eval "$(gwt shell-init zsh)"
gwt() {
  case "$1" in
    new|from|co|checkout|search|pick|dashboard)
      local a
      for a in "$@"; do
        if [[ "$a" == -h || "$a" == --help ]]; then
          command gwt "$@"
          return
        fi
      done
      local out
      out="$(command gwt "$@")" || return
      if [[ -z "$out" ]]; then
        return
      fi
      if [[ "$out" == GWT_POPULATE:* ]]; then
        print -z -- "${out#GWT_POPULATE:}"
      else
        builtin cd -- "${out##*$'\n'}"
        if [[ -n ${GWT_AUTO_LS:-} ]]; then
          command gwt ls
        fi
      fi
      ;;
    "")
      # Bare invocation: dashboard. It prints the selected path on stdout.
      local out
      out="$(command gwt)" || return
      if [[ -n "$out" ]]; then
        builtin cd -- "${out##*$'\n'}"
        if [[ -n ${GWT_AUTO_LS:-} ]]; then
          command gwt ls
        fi
      fi
      ;;
    *)
      command gwt "$@"
      ;;
  esac
}
