package shell

import "text/template"

// PowerShellTemplate is the shell integration template for PowerShell 7+.
var PowerShellTemplate = template.Must(template.New("powershell").Parse(`# wf shell integration for PowerShell 7+
# Usage: wf init powershell | Invoke-Expression
# Or add to $PROFILE: wf init powershell | Invoke-Expression
{{.Comment}}

Set-PSReadLineKeyHandler -Chord '{{.Key}}' -ScriptBlock {
    [Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
    $output = wf pick 2>$null
    if ($output) {
        [Microsoft.PowerShell.PSConsoleReadLine]::Insert($output)
    }
}
`))
