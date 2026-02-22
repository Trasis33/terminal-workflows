package shell

// BashScript is the shell integration script for bash.
// Users source this via: eval "$(wf init bash)"
//
// Pattern verified from fzf key-bindings.bash source.
// Uses READLINE_LINE/READLINE_POINT for readline buffer manipulation.
// Requires bash >= 4.0 for bind -x with READLINE_LINE.
// Uses fd swap (3>&1 1>&2 2>&3) so the TUI renders on the terminal
// while the selected command is captured by the shell function.
const BashScript = `# wf shell integration for bash
# Usage: eval "$(wf init bash)"

_wf_picker() {
  local output
  # Swap fd: TUI on stderr (terminal), selection on stdout (captured)
  output=$(wf pick 3>&1 1>&2 2>&3)
  if [[ -n "$output" ]]; then
    READLINE_LINE="$output"
    READLINE_POINT=${#READLINE_LINE}
  fi
}
bind -m emacs-standard -x '"\C-g": _wf_picker'
bind -m vi-insert -x '"\C-g": _wf_picker'
`
