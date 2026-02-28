---
phase: 07-polish-terminal-compat
plan: 02
status: completed
key-files:
  created:
    - internal/shell/keybinding.go
    - internal/shell/keybinding_test.go
  modified:
    - internal/shell/zsh.go
    - internal/shell/bash.go
    - internal/shell/fish.go
    - internal/shell/powershell.go
    - cmd/wf/init_shell.go
self-check: passed
---

# 07-02 Summary

Implemented configurable shell keybindings and Warp auto-detection.

What changed:
- Added `internal/shell/keybinding.go`:
  - `Keybinding` type, `ParseKey`, `Validate`, shell-specific format methods
  - `DefaultKey` (`Ctrl+G`) and `WarpDefaultKey` (`Ctrl+O`)
  - blocked dangerous keys (`Ctrl+C`, `Ctrl+D`, `Ctrl+Z`, `Ctrl+S`, `Ctrl+Q`)
  - `TemplateData` and `DetectWarp()`
- Added `internal/shell/keybinding_test.go` covering parse/validate/format behavior
- Converted shell integration scripts to templates:
  - `ZshTemplate`, `BashTemplate`, `FishTemplate`, `PowerShellTemplate`
  - added `{{.Key}}` placeholder in bindings and `{{.Comment}}` block in headers
- Updated `wf init` command:
  - supports `--key ctrl+<letter>` / `--key alt+<letter>`
  - validates blocked keys and returns descriptive errors
  - auto-selects Warp fallback key and logs to stderr
  - renders script templates with shell-specific key encodings

Validation:
- `go build ./internal/shell/...`
- `go test ./internal/shell/... -v -count=1`
- `go build ./...`
- `go test ./cmd/wf/... -v -count=1 -run TestCLI`
- manual spot checks for default, custom, and blocked key output

Issues encountered: none.
