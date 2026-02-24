package shell

// BashScript is the shell integration script for bash.
// Users source this via: eval "$(wf init bash)"
//
// Pattern verified from fzf key-bindings.bash source.
// Uses READLINE_LINE/READLINE_POINT for readline buffer manipulation.
// Requires bash >= 4.0 for bind -x with READLINE_LINE.
// wf pick renders TUI directly to /dev/tty, so no fd swap is needed —
// stdout carries only the selected command.
const BashScript = `# wf shell integration for bash
# Usage: eval "$(wf init bash)"

_wf_picker() {
  local output
  output=$(wf pick)
  if [[ -n "$output" ]]; then
    READLINE_LINE="$output"
    READLINE_POINT=${#READLINE_LINE}
  fi
}
bind -m emacs-standard -x '"\C-g": _wf_picker'
bind -m vi-insert -x '"\C-g": _wf_picker'

_wf_precmd() {
  local _wf_dir="${XDG_DATA_HOME:-$HOME/.local/share}/wf"
  [[ -d "$_wf_dir" ]] || mkdir -p "$_wf_dir"
  local _last
  _last=$(HISTTIMEFORMAT='' history 1 | sed 's/^[ ]*[0-9]*[ ]*//')
  [[ -n "$_last" ]] && printf '%s' "$_last" > "$_wf_dir/last_cmd"
}
PROMPT_COMMAND="_wf_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
`
