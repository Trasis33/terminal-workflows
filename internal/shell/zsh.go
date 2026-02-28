package shell

import "text/template"

// ZshTemplate is the shell integration template for zsh.
var ZshTemplate = template.Must(template.New("zsh").Parse(`# wf shell integration for zsh
# Usage: eval "$(wf init zsh)"  or  source <(wf init zsh)
{{.Comment}}

_wf_picker() {
  local output
  output=$(wf pick)
  local ret=$?
  if [[ -n "$output" ]]; then
    LBUFFER="$output"
    RBUFFER=""
  fi
  zle reset-prompt
  return $ret
}
zle -N _wf_picker
bindkey -M emacs '{{.Key}}' _wf_picker
bindkey -M viins '{{.Key}}' _wf_picker
bindkey -M vicmd '{{.Key}}' _wf_picker

_wf_precmd() {
  local _wf_dir="${XDG_DATA_HOME:-$HOME/.local/share}/wf"
  [[ -d "$_wf_dir" ]] || mkdir -p "$_wf_dir"
  print -r -- "$history[1]" > "$_wf_dir/last_cmd"
}
autoload -Uz add-zsh-hook
add-zsh-hook precmd _wf_precmd
`))
