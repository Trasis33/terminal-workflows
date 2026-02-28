---
phase: 07-polish-terminal-compat
plan: 03
status: completed
key-files:
  created: []
  modified:
    - internal/manage/sidebar.go
    - internal/manage/browse.go
    - internal/manage/model_test.go
    - internal/picker/model.go
    - cmd/wf/list.go
self-check: passed
---

# 07-03 Summary

Implemented UX polish for manage/picker/list surfaces.

What changed:
- Sidebar auto-filter now triggers on cursor movement (`up/down/j/k`) via `sidebarFilterMsg`
- Added browse breadcrumb line for active filter context (`All Workflows`, folder, or tag)
- Added context-aware empty states for folder/tag filters
- Replaced raw manage preview rendering with viewport-backed content:
  - syntax-highlighted command preview using `highlight.Shell(...)`
  - bounded scrolling using viewport
  - scroll indicator (`Scroll N%`) when content exceeds preview height
  - `J/K` keys scroll preview without changing list selection
- Picker preview now syntax-highlights command text using `highlight.Shell(...)`
- `wf list` output now visually separates folder/name/description/tags with lipgloss styles
- Added test coverage for sidebar cursor movement emitting filter commands

Validation:
- `go build ./...`
- `go test ./internal/manage/... -v -count=1`
- `go test ./internal/picker/... -v -count=1`
- `go test ./cmd/wf/... -v -count=1`
- `go test ./internal/highlight/... -v -count=1`
- `go test ./... -count=1`
- `go run ./cmd/wf list | head -5`

Issues encountered: none.
