# gwt shell integration (fish)
#
# GWT_BIN is set by shell-init. Switch verbs use GWT_PATH_OUT so stdout is not
# captured in command substitution (which breaks long-running setup scripts).
#
# Install with:  gwt shell-init fish | source
function _gwt_cd_from
    set -l pathfile (mktemp "$TMPDIR/gwt-path.XXXXXX")
    or return 1
    set -lx GWT_PATH_OUT $pathfile
    $GWT_BIN $argv
    or begin; rm -f $pathfile; return; end
    test -f $pathfile
    or return 0
    set -l content (cat $pathfile)
    rm -f $pathfile
    test -n "$content"
    or return 0
    if string match -q 'GWT_POPULATE:*' -- "$content"
        commandline -r -- (string replace 'GWT_POPULATE:' '' -- "$content")
    else
        set -l path (string trim -- "$content")
        builtin cd -- "$path"
        if test -n "$GWT_AUTO_LS"
            $GWT_BIN ls
        end
    end
end

function gwt
    switch "$argv[1]"
        case new from co checkout search pick dashboard ''
            for a in $argv
                if test "$a" = -h -o "$a" = --help
                    $GWT_BIN $argv
                    return
                end
            end
            _gwt_cd_from $argv
        case '*'
            $GWT_BIN $argv
    end
end
