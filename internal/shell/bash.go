package shell

import "text/template"

// BashTemplate is the shell integration template for bash.
var BashTemplate = template.Must(template.New("bash").Parse(`# wf shell integration for bash
# Usage: eval "$(wf init bash)"
{{.Comment}}

_wf_picker() {
  local output
  output=$(wf pick)
  if [[ -n "$output" ]]; then
    READLINE_LINE="$output"
    READLINE_POINT=${#READLINE_LINE}
  fi
}
bind -m emacs-standard -x '"{{.Key}}": _wf_picker'
bind -m vi-insert -x '"{{.Key}}": _wf_picker'

_wf_precmd() {
  local _wf_dir="${XDG_DATA_HOME:-$HOME/.local/share}/wf"
  [[ -d "$_wf_dir" ]] || mkdir -p "$_wf_dir"
  local _last
  _last=$(HISTTIMEFORMAT='' history 1 | sed 's/^[ ]*[0-9]*[ ]*//')
  [[ -n "$_last" ]] && printf '%s' "$_last" > "$_wf_dir/last_cmd"
}
PROMPT_COMMAND="_wf_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
`))
