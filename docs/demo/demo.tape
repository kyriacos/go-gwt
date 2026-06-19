# Demo recording for gwt, driven by VHS (https://github.com/charmbracelet/vhs).
#
# Regenerate the gif:
#   brew install vhs          # needs ttyd + ffmpeg
#   go install .              # installs `gwt` (the demo below uses it as `gogwt`)
#   bash docs/demo/demo-setup.sh   # builds a sample repo in /tmp/gwt-demo
#   vhs docs/demo/demo.tape        # writes docs/demo/demo.gif
#
# The sample repo has worktrees in every state (active / local-only / gone) so
# the listing and dashboard show off the color-coded states.

Output docs/demo/demo.gif

Require gogwt
Require git

Set Shell "bash"
Set FontSize 15
Set Width 1400
Set Height 840
Set Padding 22
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type "cd /tmp/gwt-demo/go-gwt && clear" Enter
Show

# --- ls: worktrees color-coded by state (+ legend when stale) ---
Type "gogwt ls" Sleep 700ms Enter
Sleep 3s
Type "clear" Enter

# --- the dashboard: live status, preview, navigate, filter ---
Type "gogwt" Sleep 600ms Enter
Sleep 2s
Down Sleep 800ms
Down Sleep 800ms
Down Sleep 1s
Type "/" Sleep 400ms Type "auth" Sleep 1600ms Escape Sleep 900ms
Up Sleep 800ms
Enter
Sleep 1.5s
Type "clear" Enter

# --- branch picker: `from` with no argument ---
Type "gogwt from" Sleep 600ms Enter
Sleep 1800ms
Down Sleep 800ms
Down Sleep 1s
Type "q"
Sleep 800ms
Type "clear" Enter

# --- clean: multi-select, stale worktrees pre-marked ---
Type "gogwt clean" Sleep 600ms Enter
Sleep 2s
Down Sleep 800ms
Space Sleep 1s
Up Sleep 600ms
Type "q"
Sleep 1500ms
