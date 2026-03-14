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

Set-PSReadLineKeyHandler -Chord '{{.ManageKey}}' -ScriptBlock {
    [Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
    $resultFile = [System.IO.Path]::GetTempFileName()
    try {
        wf manage --result-file $resultFile 2>$null
        if ($LASTEXITCODE -eq 0 -and (Test-Path -LiteralPath $resultFile) -and (Get-Item -LiteralPath $resultFile).Length -gt 0) {
            $output = Get-Content -Raw -LiteralPath $resultFile
            if ($output) {
                [Microsoft.PowerShell.PSConsoleReadLine]::Insert($output)
            }
        }
    } finally {
        Remove-Item -Force $resultFile -ErrorAction SilentlyContinue
    }
}
`))
