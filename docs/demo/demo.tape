# Demo recording for gwt, driven by VHS (https://github.com/charmbracelet/vhs).
#
# Regenerate the gif:
#   brew install vhs          # needs ttyd + ffmpeg
#   go install .              # put `gwt` on your PATH
#   vhs docs/demo/demo.tape   # writes docs/demo/demo.gif
#
# The tape spins up a throwaway repo in a temp dir so the recording is
# self-contained and never touches a real project.

Output docs/demo/demo.gif

Require gwt
Require git

Set Shell "bash"
Set FontSize 16
Set Width 1280
Set Height 820
Set Padding 22
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# --- silent setup: a scratch repo with a couple of branches + the cd wrapper ---
Hide
Type "export D=$(mktemp -d)/backend && mkdir -p $D && cd $D" Enter
Type "git init -q && git config user.email demo@example.com && git config user.name demo" Enter
Type "git commit -q --allow-empty -m 'initial commit'" Enter
Type "git branch feature-login && git branch feature-api" Enter
Type 'eval "$(gwt shell-init bash)"' Enter
Type "clear" Enter
Show

# --- new worktree: lands as a sibling, named <repo>-<branch>, and cd's in ---
Type "gwt new payments-refactor" Sleep 700ms Enter
Sleep 2s
Type "cd $D" Enter
Sleep 500ms

# --- a worktree for an existing branch ---
Type "gwt from feature-login" Sleep 500ms Enter
Sleep 1500ms
Type "cd $D" Enter
Sleep 500ms

# --- list everything ---
Type "gwt ls" Sleep 500ms Enter
Sleep 2s

# --- the dashboard: navigate, filter, quit ---
Type "gwt dashboard" Sleep 500ms Enter
Sleep 2s
Down Sleep 800ms
Down Sleep 800ms
Up Sleep 800ms
Type "/" Sleep 400ms Type "feat" Sleep 1500ms Escape Sleep 800ms
Type "q" Sleep 1s

# --- clean up a worktree and its branch ---
Type "gwt rm feature-login -d" Sleep 500ms Enter
Sleep 2s
Type "gwt ls" Sleep 500ms Enter
Sleep 2500ms
