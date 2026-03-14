package shell

import "text/template"

// FishTemplate is the shell integration template for fish.
var FishTemplate = template.Must(template.New("fish").Parse(`# wf shell integration for fish
# Usage: wf init fish | source
{{.Comment}}

function _wf_picker
  set -l output (wf pick | string collect)
  if test -n "$output"
    commandline -r $output
  end
  commandline -f repaint
end
bind {{.Key}} _wf_picker
bind -M insert {{.Key}} _wf_picker

function _wf_manage
  set -l result_file (mktemp)
  if test $status -ne 0
    commandline -f repaint
    return 1
  end

  wf manage --result-file "$result_file"
  set -l ret $status
  if test $ret -eq 0 -a -s "$result_file"
    set -l output (string collect < "$result_file")
    if test -n "$output"
      commandline -r $output
    end
  end
  rm -f -- "$result_file"
  commandline -f repaint
  return $ret
end
bind {{.ManageKey}} _wf_manage
bind -M insert {{.ManageKey}} _wf_manage

function _wf_postexec --on-event fish_postexec
  set -l _wf_dir (test -n "$XDG_DATA_HOME" && echo "$XDG_DATA_HOME" || echo "$HOME/.local/share")"/wf"
  mkdir -p "$_wf_dir"
  printf '%s' $argv[1] > "$_wf_dir/last_cmd"
end
`))
