# Demo recording for gwt, driven by VHS (https://github.com/charmbracelet/vhs).
#
# Regenerate the gif:
#   brew install vhs          # or: go install github.com/charmbracelet/vhs@latest
#   go install .              # put `gwt` on your PATH
#   vhs docs/demo/demo.tape   # writes docs/demo/demo.gif
#
# The tape spins up a throwaway repo in a temp dir so the recording is
# self-contained and never touches a real project.

Output docs/demo/demo.gif

Require gwt
Require git

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

# --- silent setup: a scratch repo with a couple of branches ---
Hide
Type "cd $(mktemp -d)/backend && git init -q . 2>/dev/null || (mkdir -p $(mktemp -d)/backend && cd $_)" Enter
Type "git init -q && git config user.email demo@example.com && git config user.name demo" Enter
Type "git commit -q --allow-empty -m 'initial commit'" Enter
Type "git branch feature-login && git branch feature-api" Enter
Type "clear" Enter
Show

# --- new worktree: lands as a sibling, named <repo>-<branch> ---
Type "gwt new payments-refactor" Sleep 800ms Enter
Sleep 2s

# --- list worktrees ---
Type "gwt ls" Sleep 600ms Enter
Sleep 2s

# --- create one for an existing branch ---
Type "gwt from feature-login" Sleep 600ms Enter
Sleep 2s

# --- the dashboard: browse, preview, quit ---
Type "gwt dashboard" Sleep 600ms Enter
Sleep 2s
Down Sleep 1s
Down Sleep 1s
Type "/" Sleep 500ms Type "feat" Sleep 1500ms Escape Sleep 1s
Type "q" Sleep 1s

# --- clean up a worktree and its branch ---
Type "gwt rm feature-login -d" Sleep 600ms Enter
Sleep 2s

Type "gwt ls" Sleep 600ms Enter
Sleep 2500ms
