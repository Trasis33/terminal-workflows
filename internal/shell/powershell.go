package shell

// PowerShellScript is the shell integration script for PowerShell 7+.
// Users source this via: wf init powershell | Invoke-Expression
// Or add to $PROFILE: wf init powershell | Invoke-Expression
//
// Pattern verified from Atuin's atuin.ps1 source.
// Uses PSReadLine APIs for command line buffer manipulation.
// RevertLine() clears current input before inserting selected command.
// 2>$null suppresses stderr (TUI renders to CONOUT$ on Windows).
// No sidecar/last_cmd tracking for PowerShell in v1 — precmd hooks
// are shell-specific to zsh/bash/fish.
const PowerShellScript = `# wf shell integration for PowerShell 7+
# Usage: wf init powershell | Invoke-Expression
# Or add to $PROFILE: wf init powershell | Invoke-Expression

Set-PSReadLineKeyHandler -Chord 'Ctrl+g' -ScriptBlock {
    [Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
    $output = wf pick 2>$null
    if ($output) {
        [Microsoft.PowerShell.PSConsoleReadLine]::Insert($output)
    }
}
`
