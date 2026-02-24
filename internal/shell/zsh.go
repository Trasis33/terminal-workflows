package shell

// ZshScript is the shell integration script for zsh.
// Users source this via: eval "$(wf init zsh)" or source <(wf init zsh)
//
// Pattern verified from fzf key-bindings.zsh and atuin.zsh sources.
// Uses LBUFFER/RBUFFER for ZLE buffer manipulation.
// wf pick renders TUI directly to /dev/tty, so no fd swap is needed —
// stdout carries only the selected command.
const ZshScript = `# wf shell integration for zsh
# Usage: eval "$(wf init zsh)"  or  source <(wf init zsh)

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
bindkey -M emacs '^G' _wf_picker
bindkey -M viins '^G' _wf_picker
bindkey -M vicmd '^G' _wf_picker

_wf_precmd() {
  local _wf_dir="${XDG_DATA_HOME:-$HOME/.local/share}/wf"
  [[ -d "$_wf_dir" ]] || mkdir -p "$_wf_dir"
  print -r -- "$history[1]" > "$_wf_dir/last_cmd"
}
autoload -Uz add-zsh-hook
add-zsh-hook precmd _wf_precmd
`
