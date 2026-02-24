package picker

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/sahilm/fuzzy"
)

// viewState tracks which screen the picker is on.
type viewState int

const (
	// StateSearch is the fuzzy search and browse state.
	StateSearch viewState = iota
	// StateParamFill is the parameter fill state for parameterized workflows.
	StateParamFill
)

// clearFlashMsg is sent after a timer to clear the flash message.
type clearFlashMsg struct{}

// SearchResult pairs a fuzzy match with its originating Workflow.
type SearchResult struct {
	Workflow store.Workflow
	Match    fuzzy.Match
}

// Model is the Bubble Tea model for the workflow picker.
type Model struct {
	state     viewState
	workflows []store.Workflow // all loaded workflows, immutable after init

	// Search state
	searchInput textinput.Model
	results     []SearchResult
	cursor      int
	preview     viewport.Model

	// Param fill state
	selected          *store.Workflow
	params            []template.Param
	paramInputs       []textinput.Model
	focusedParam      int
	paramTypes        []template.ParamType // type per param (text/enum/dynamic)
	paramOptions      [][]string           // options for enum/dynamic params (nil for text)
	paramOptionCursor []int                // cursor position within each param's option list
	paramLoading      []bool               // true while dynamic command is executing
	paramFailed       []bool               // true if dynamic command failed (fallback to text)

	// Result is the final output command, read by caller after tea.Quit.
	Result string

	// Flash message (brief feedback, e.g. "Copied!")
	flashMsg string

	// Layout
	width      int
	height     int
	maxVisible int
}

// New creates a new picker Model from a list of workflows.
func New(workflows []store.Workflow) Model {
	ti := textinput.New()
	ti.Placeholder = "Search workflows... (@tag to filter)"
	ti.Focus()
	ti.PromptStyle = searchPromptStyle
	ti.Prompt = "❯ "

	vp := viewport.New(80, 6)

	m := Model{
		state:       StateSearch,
		workflows:   workflows,
		searchInput: ti,
		preview:     vp,
		maxVisible:  10,
	}

	m.performSearch("")
	return m
}

// Init returns the initial command. Bubble Tea automatically sends WindowSizeMsg.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update processes messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve lines for: search input (2), hints (1), preview border (3+), padding (2)
		m.maxVisible = msg.Height - 8
		if m.maxVisible < 1 {
			m.maxVisible = 1
		}
		m.preview.Width = msg.Width - 4 // account for border padding
		m.preview.Height = 4
		previewBorderStyle = previewBorderStyle.Width(msg.Width - 4)
		m.updatePreview()
		return m, nil

	case clearFlashMsg:
		m.flashMsg = ""
		return m, nil

	case dynamicResultMsg:
		return m.handleDynamicResult(msg)

	case tea.KeyMsg:
		switch m.state {
		case StateSearch:
			return m.updateSearch(msg)
		case StateParamFill:
			return m.updateParamFill(msg)
		}
	}

	return m, nil
}

// updateSearch handles key events in the search/browse state.
func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.Result = ""
		return m, tea.Quit

	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
			m.updatePreview()
		}
		return m, nil

	case "down", "ctrl+n":
		if m.cursor < len(m.results)-1 {
			m.cursor++
			m.updatePreview()
		}
		return m, nil

	case "enter":
		if len(m.results) == 0 {
			return m, nil
		}
		wf := m.results[m.cursor].Workflow
		params := template.ExtractParams(wf.Command)
		if len(params) == 0 {
			// Zero-param workflow: output directly
			m.Result = wf.Command
			return m, tea.Quit
		}
		// Transition to param fill
		m.selected = &wf
		initParamFill(&m)
		m.state = StateParamFill
		// Fire dynamic param commands
		cmds := initParamFillCmds(&m)
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case "ctrl+y":
		if len(m.results) == 0 {
			return m, nil
		}
		cmd := m.results[m.cursor].Workflow.Command
		if err := clipboard.WriteAll(cmd); err != nil {
			m.flashMsg = "✗ clipboard error"
		} else {
			m.flashMsg = "✓ Copied!"
		}
		return m, tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
			return clearFlashMsg{}
		})

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.performSearch(m.searchInput.Value())
		m.cursor = 0
		m.updatePreview()
		return m, cmd
	}
}

// performSearch runs fuzzy search and updates results.
func (m *Model) performSearch(raw string) {
	tagFilter, fuzzyQuery := ParseQuery(raw)
	matches := Search(fuzzyQuery, tagFilter, m.workflows)

	m.results = make([]SearchResult, len(matches))
	for i, match := range matches {
		idx := match.Index
		if idx >= 0 && idx < len(m.workflows) {
			m.results[i] = SearchResult{
				Workflow: m.workflows[idx],
				Match:    match,
			}
		}
	}

	m.updatePreview()
}

// updatePreview sets the preview pane content to the selected workflow's command.
func (m *Model) updatePreview() {
	if len(m.results) > 0 && m.cursor < len(m.results) {
		m.preview.SetContent(m.results[m.cursor].Workflow.Command)
	} else {
		m.preview.SetContent("")
	}
}

// handleDynamicResult processes a completed dynamic parameter command.
func (m Model) handleDynamicResult(msg dynamicResultMsg) (tea.Model, tea.Cmd) {
	idx := msg.paramIndex
	if idx < 0 || idx >= len(m.paramLoading) {
		return m, nil
	}

	m.paramLoading[idx] = false

	if msg.err != nil || len(msg.options) == 0 {
		// Failed: fall back to free-text input
		m.paramFailed[idx] = true
		m.paramInputs[idx].Placeholder = m.params[idx].Name
		m.paramInputs[idx].SetValue("")
		return m, nil
	}

	// Success: populate option list
	m.paramOptions[idx] = msg.options
	m.paramOptionCursor[idx] = 0
	m.paramInputs[idx].SetValue(msg.options[0])
	m.paramInputs[idx].Placeholder = ""
	return m, nil
}

// View renders the full picker UI.
func (m Model) View() string {
	switch m.state {
	case StateSearch:
		return m.viewSearch()
	case StateParamFill:
		return m.viewParamFill()
	}
	return ""
}

// viewSearch renders the search/browse state.
func (m Model) viewSearch() string {
	var sections []string

	// Search input
	sections = append(sections, m.searchInput.View())
	sections = append(sections, "")

	// Result list
	visible := m.maxVisible
	if visible > len(m.results) {
		visible = len(m.results)
	}

	// Calculate scroll offset to keep cursor visible
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.results) {
		end = len(m.results)
		start = end - visible
		if start < 0 {
			start = 0
		}
	}

	if len(m.results) == 0 {
		sections = append(sections, dimStyle.Render("  No workflows found"))
	} else {
		for i := start; i < end; i++ {
			isSelected := i == m.cursor
			row := m.renderResultRow(m.results[i], isSelected)
			sections = append(sections, row)
		}
	}

	sections = append(sections, "")

	// Preview pane
	if len(m.results) > 0 {
		previewContent := m.preview.View()
		sections = append(sections, previewBorderStyle.Render(previewContent))
	}

	// Hints
	if m.flashMsg != "" {
		sections = append(sections, hintStyle.Render("  "+m.flashMsg))
	} else {
		sections = append(sections, hintStyle.Render("  ↑↓ navigate  enter select  ctrl+y copy  esc quit"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderResultRow renders a single result row with fuzzy match highlighting.
func (m Model) renderResultRow(sr SearchResult, isSelected bool) string {
	wf := sr.Workflow

	// Build name with highlighted match characters
	name := highlightName(wf.Name, sr.Match.MatchedIndexes, isSelected)

	// Truncate description
	desc := truncateStr(wf.Description, 40)

	// Tags
	var tags string
	if len(wf.Tags) > 0 {
		tags = tagStyle.Render("[" + strings.Join(wf.Tags, " ") + "]")
	}

	// Compose row
	var prefix string
	if isSelected {
		prefix = cursorStyle.Render("❯ ")
	} else {
		prefix = "  "
	}

	parts := []string{prefix, name}
	if desc != "" {
		parts = append(parts, " ", dimStyle.Render(desc))
	}
	if tags != "" {
		parts = append(parts, " ", tags)
	}

	row := strings.Join(parts, "")

	if isSelected {
		return selectedStyle.Render(row)
	}
	return row
}

// highlightName renders a workflow name with matched characters highlighted.
func highlightName(name string, matchedIndexes []int, isSelected bool) string {
	if len(matchedIndexes) == 0 {
		if isSelected {
			return selectedStyle.Render(name)
		}
		return normalStyle.Render(name)
	}

	// Build set of matched positions
	matched := make(map[int]bool, len(matchedIndexes))
	for _, idx := range matchedIndexes {
		matched[idx] = true
	}

	var b strings.Builder
	for i, ch := range name {
		s := string(ch)
		if matched[i] {
			b.WriteString(highlightStyle.Render(s))
		} else if isSelected {
			b.WriteString(selectedStyle.Render(s))
		} else {
			b.WriteString(normalStyle.Render(s))
		}
	}
	return b.String()
}

// truncateStr truncates a string to maxLen, adding "…" if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return s[:maxLen-1] + "…"
}
