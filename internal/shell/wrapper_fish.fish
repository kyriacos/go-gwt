# gwt shell integration (fish)
#
# A process cannot change its parent shell's working directory, so the switch
# verbs (new|from|co|checkout|search|pick and the bare dashboard) print the
# chosen worktree path to stdout. This wrapper captures that path and cd's
# there. Everything else (diagnostics, prompts, the TUI) is written to the tty
# by gwt itself, so it is left untouched.
#
# If gwt instead prints a line beginning with "GWT_POPULATE:" (used by from/co
# with no argument to suggest a command for review), the remainder is inserted
# into the command line with `commandline` so you can edit it before running.
#
# Install with:  gwt shell-init fish | source
function gwt
    switch "$argv[1]"
        case new from co checkout search pick dashboard ''
            set -l out (command gwt $argv)
            or return
            if test -z "$out"
                return
            end
            set -l last $out[-1]
            if string match -q 'GWT_POPULATE:*' -- "$last"
                commandline -r -- (string replace 'GWT_POPULATE:' '' -- "$last")
            else
                builtin cd -- "$last"
                if test -n "$GWT_AUTO_LS"
                    command gwt ls
                end
            end
        case '*'
            command gwt $argv
    end
end
