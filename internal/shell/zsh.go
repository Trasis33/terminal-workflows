package shell

// ZshScript is the shell integration script for zsh.
// Users source this via: eval "$(wf init zsh)" or source <(wf init zsh)
//
// Pattern verified from fzf key-bindings.zsh and atuin.zsh sources.
// Uses LBUFFER/RBUFFER for ZLE buffer manipulation.
// Uses fd swap (3>&1 1>&2 2>&3) so the TUI renders on the terminal
// while the selected command is captured by the shell function.
const ZshScript = `# wf shell integration for zsh
# Usage: eval "$(wf init zsh)"  or  source <(wf init zsh)

_wf_picker() {
  local output
  # Swap fd: TUI on stderr (terminal), selection on stdout (captured)
  output=$(wf pick 3>&1 1>&2 2>&3)
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
`
