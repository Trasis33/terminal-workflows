package shell

// FishScript is the shell integration script for fish.
// Users source this via: wf init fish | source
//
// Pattern verified from atuin.fish source.
// Uses commandline -r to replace the line buffer.
// wf pick renders TUI directly to /dev/tty, so no fd swap is needed —
// stdout carries only the selected command.
const FishScript = `# wf shell integration for fish
# Usage: wf init fish | source

function _wf_picker
  set -l output (wf pick | string collect)
  if test -n "$output"
    commandline -r $output
  end
  commandline -f repaint
end
bind \cg _wf_picker
bind -M insert \cg _wf_picker
`
