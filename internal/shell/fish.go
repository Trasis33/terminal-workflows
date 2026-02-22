package shell

// FishScript is the shell integration script for fish.
// Users source this via: wf init fish | source
//
// Pattern verified from atuin.fish source.
// Uses commandline -r to replace the line buffer.
// Uses fd swap (3>&1 1>&2 2>&3) so the TUI renders on the terminal
// while the selected command is captured by the shell function.
const FishScript = `# wf shell integration for fish
# Usage: wf init fish | source

function _wf_picker
  set -l output (wf pick 3>&1 1>&2 2>&3 | string collect)
  if test -n "$output"
    commandline -r $output
  end
  commandline -f repaint
end
bind \cg _wf_picker
bind -M insert \cg _wf_picker
`
