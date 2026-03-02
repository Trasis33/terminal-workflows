package manage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
)

const dialogExecute dialogType = 11
const executePreviewMinRows = 4

type executePhase int

const (
	phaseParamFill executePhase = iota
	phaseActionMenu
)

type executeAction int

const (
	actionCopy executeAction = iota
	actionPasteToPrompt
	actionCancel
)

type executeDialogDynamicMsg struct {
	paramIndex int
	options    []string
	err        error
}

type ExecuteDialogModel struct {
	workflow store.Workflow
	phase    executePhase

	params            []template.Param
	paramInputs       []textinput.Model
	paramTypes        []template.ParamType
	paramOptions      [][]string
	paramOptionCursor []int
	paramLoading      []bool
	paramFailed       []bool
	focusedParam      int

	renderedCommand string
	actionCursor    int
	actions         []string

	width int
	theme Theme
}

func NewExecuteDialog(wf store.Workflow, width int, theme Theme) ExecuteDialogModel {
	params := template.ExtractParams(wf.Command)
	for i, p := range params {
		if p.Default != "" {
			continue
		}
		for _, arg := range wf.Args {
			if arg.Name == p.Name && arg.Default != "" {
				params[i].Default = arg.Default
				break
			}
		}
	}

	d := ExecuteDialogModel{
		workflow: wf,
		params:   params,
		actions: []string{
			"Copy to clipboard",
			"Paste to prompt",
			"Cancel",
		},
		width: width,
		theme: theme,
	}

	if len(params) == 0 {
		d.phase = phaseActionMenu
		d.renderedCommand = template.Render(wf.Command, nil)
		return d
	}

	d.phase = phaseParamFill
	d.paramInputs = make([]textinput.Model, len(params))
	d.paramTypes = make([]template.ParamType, len(params))
	d.paramOptions = make([][]string, len(params))
	d.paramOptionCursor = make([]int, len(params))
	d.paramLoading = make([]bool, len(params))
	d.paramFailed = make([]bool, len(params))

	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	for i, p := range params {
		ti := textinput.New()
		ti.Placeholder = p.Name
		ti.CharLimit = 256
		d.paramTypes[i] = p.Type

		switch p.Type {
		case template.ParamEnum:
			d.paramOptions[i] = p.Options
			defIdx := 0
			if p.Default != "" {
				for j, opt := range p.Options {
					if opt == p.Default {
						defIdx = j
						break
					}
				}
			}
			d.paramOptionCursor[i] = defIdx
			if len(p.Options) > 0 {
				ti.SetValue(p.Options[defIdx])
			}
		case template.ParamDynamic:
			d.paramLoading[i] = true
			ti.Placeholder = "Loading..."
		default:
			if p.Default != "" {
				ti.SetValue(p.Default)
				ti.TextStyle = defaultStyle
				ti.CursorEnd()
			}
		}

		if i == 0 {
			ti.Focus()
		}
		d.paramInputs[i] = ti
	}

	return d
}

func (d ExecuteDialogModel) Init() tea.Cmd {
	if d.phase == phaseParamFill && len(d.paramInputs) > 0 {
		return textinput.Blink
	}
	return nil
}

func (d ExecuteDialogModel) InitCmds() []tea.Cmd {
	var cmds []tea.Cmd
	for i, p := range d.params {
		if p.Type != template.ParamDynamic {
			continue
		}
		idx := i
		dynCmd := p.DynamicCmd
		cmds = append(cmds, func() tea.Msg {
			return executeDialogDynamic(idx, dynCmd)
		})
	}
	return cmds
}

func executeDialogDynamic(paramIndex int, command string) executeDialogDynamicMsg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return executeDialogDynamicMsg{paramIndex: paramIndex, err: fmt.Errorf("dynamic command failed: %w", err)}
	}

	var options []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			options = append(options, line)
		}
	}
	if len(options) == 0 {
		return executeDialogDynamicMsg{paramIndex: paramIndex, err: fmt.Errorf("dynamic command returned no output")}
	}
	return executeDialogDynamicMsg{paramIndex: paramIndex, options: options}
}

func (d ExecuteDialogModel) Update(msg tea.Msg) (ExecuteDialogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case executeDialogDynamicMsg:
		return d.handleDynamicResult(msg)
	case tea.KeyMsg:
		if d.phase == phaseActionMenu {
			return d.updateActionMenu(msg)
		}
		return d.updateParamFill(msg)
	}
	return d, nil
}

func (d ExecuteDialogModel) handleDynamicResult(msg executeDialogDynamicMsg) (ExecuteDialogModel, tea.Cmd) {
	i := msg.paramIndex
	if i < 0 || i >= len(d.params) {
		return d, nil
	}

	d.paramLoading[i] = false
	if msg.err != nil || len(msg.options) == 0 {
		d.paramFailed[i] = true
		d.paramInputs[i].Placeholder = d.params[i].Name
		return d, nil
	}

	d.paramOptions[i] = msg.options
	d.paramOptionCursor[i] = 0
	d.paramTypes[i] = template.ParamEnum
	d.paramInputs[i].SetValue(msg.options[0])
	return d, nil
}

func (d ExecuteDialogModel) updateParamFill(msg tea.KeyMsg) (ExecuteDialogModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: dialogExecute, confirmed: false}
		}
	case "tab":
		d.paramInputs[d.focusedParam].Blur()
		d.focusedParam = (d.focusedParam + 1) % len(d.paramInputs)
		d.paramInputs[d.focusedParam].Focus()
		return d, nil
	case "shift+tab":
		d.paramInputs[d.focusedParam].Blur()
		d.focusedParam = (d.focusedParam - 1 + len(d.paramInputs)) % len(d.paramInputs)
		d.paramInputs[d.focusedParam].Focus()
		return d, nil
	case "up":
		if d.isListParam(d.focusedParam) {
			opts := d.paramOptions[d.focusedParam]
			if len(opts) > 0 {
				cur := d.paramOptionCursor[d.focusedParam]
				cur = (cur - 1 + len(opts)) % len(opts)
				d.paramOptionCursor[d.focusedParam] = cur
				d.paramInputs[d.focusedParam].SetValue(opts[cur])
			}
			return d, nil
		}
	case "down":
		if d.isListParam(d.focusedParam) {
			opts := d.paramOptions[d.focusedParam]
			if len(opts) > 0 {
				cur := d.paramOptionCursor[d.focusedParam]
				cur = (cur + 1) % len(opts)
				d.paramOptionCursor[d.focusedParam] = cur
				d.paramInputs[d.focusedParam].SetValue(opts[cur])
			}
			return d, nil
		}
	case "enter":
		if d.focusedParam == len(d.paramInputs)-1 || d.allParamsFilled() {
			d.phase = phaseActionMenu
			d.actionCursor = 0
			d.renderedCommand = d.liveRender()
			return d, nil
		}
		d.paramInputs[d.focusedParam].Blur()
		d.focusedParam++
		d.paramInputs[d.focusedParam].Focus()
		return d, nil
	}

	if d.isListParam(d.focusedParam) {
		return d, nil
	}

	var cmd tea.Cmd
	d.paramInputs[d.focusedParam], cmd = d.paramInputs[d.focusedParam].Update(msg)
	d.updateFocusedTextStyle()
	return d, cmd
}

func (d ExecuteDialogModel) updateActionMenu(msg tea.KeyMsg) (ExecuteDialogModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: dialogExecute, confirmed: false}
		}
	case "up", "k":
		if d.actionCursor > 0 {
			d.actionCursor--
		}
		return d, nil
	case "down", "j":
		if d.actionCursor < len(d.actions)-1 {
			d.actionCursor++
		}
		return d, nil
	case "enter":
		switch executeAction(d.actionCursor) {
		case actionCopy:
			return d, func() tea.Msg {
				return dialogResultMsg{
					dtype:     dialogExecute,
					confirmed: true,
					data: map[string]string{
						"action":  "copy",
						"command": d.renderedCommand,
					},
				}
			}
		case actionPasteToPrompt:
			return d, func() tea.Msg {
				return dialogResultMsg{
					dtype:     dialogExecute,
					confirmed: true,
					data: map[string]string{
						"action":  "paste",
						"command": d.renderedCommand,
					},
				}
			}
		default:
			return d, func() tea.Msg {
				return dialogResultMsg{dtype: dialogExecute, confirmed: false}
			}
		}
	}
	return d, nil
}

func (d ExecuteDialogModel) isListParam(i int) bool {
	if i < 0 || i >= len(d.paramTypes) {
		return false
	}
	if d.paramTypes[i] == template.ParamEnum {
		return true
	}
	return d.paramTypes[i] == template.ParamDynamic && !d.paramLoading[i] && !d.paramFailed[i] && len(d.paramOptions[i]) > 0
}

func (d ExecuteDialogModel) allParamsFilled() bool {
	for _, input := range d.paramInputs {
		if input.Value() == "" {
			return false
		}
	}
	return true
}

func (d *ExecuteDialogModel) updateFocusedTextStyle() {
	if d.focusedParam < 0 || d.focusedParam >= len(d.paramInputs) {
		return
	}
	if d.paramTypes[d.focusedParam] != template.ParamText {
		return
	}
	def := d.params[d.focusedParam].Default
	if def != "" && d.paramInputs[d.focusedParam].Value() == def {
		d.paramInputs[d.focusedParam].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
		return
	}
	d.paramInputs[d.focusedParam].TextStyle = lipgloss.NewStyle()
}

func (d ExecuteDialogModel) liveRender() string {
	values := make(map[string]string)
	for i, p := range d.params {
		v := d.paramInputs[i].Value()
		if v != "" {
			values[p.Name] = v
		}
	}
	return template.Render(d.workflow.Command, values)
}

func (d ExecuteDialogModel) View() string {
	s := d.theme.Styles()
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(d.theme.Colors.Primary)).
		Padding(1, 2).
		Width(d.width)

	title := s.DialogTitle.Render("Execute: " + d.workflow.Name)
	var body string
	if d.phase == phaseActionMenu {
		body = d.viewActionMenu()
	} else {
		body = d.viewParamFill()
	}
	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", body))
}

func (d ExecuteDialogModel) viewParamFill() string {
	s := d.theme.Styles()
	preview := d.renderPreview()

	var rows []string
	rows = append(rows, preview, "")

	for i, p := range d.params {
		isFocused := i == d.focusedParam
		prefix := "  "
		labelStyle := s.Dim
		if isFocused {
			prefix = s.Highlight.Render("❯ ")
			labelStyle = s.Highlight
		}
		label := labelStyle.Render(p.Name + ": ")

		switch {
		case d.paramTypes[i] == template.ParamDynamic && d.paramLoading[i]:
			rows = append(rows, prefix+label+s.Dim.Render("Loading..."))
		case d.paramTypes[i] == template.ParamDynamic && d.paramFailed[i]:
			rows = append(rows, prefix+label+d.paramInputs[i].View()+s.Dim.Render(" (command failed, type manually)"))
		case d.isListParam(i):
			opts := d.paramOptions[i]
			cur := d.paramOptionCursor[i]
			value := ""
			if cur >= 0 && cur < len(opts) {
				value = opts[cur]
			}
			rows = append(rows, prefix+label+s.Highlight.Render(value))
			if isFocused {
				maxShow := 5
				start := 0
				if len(opts) > maxShow {
					start = cur - maxShow/2
					if start < 0 {
						start = 0
					}
					if start+maxShow > len(opts) {
						start = len(opts) - maxShow
					}
				}
				end := start + maxShow
				if end > len(opts) {
					end = len(opts)
				}
				for j := start; j < end; j++ {
					optPrefix := "      "
					optStyle := s.Dim
					if j == cur {
						optPrefix = "    " + s.Highlight.Render("❯ ")
						optStyle = s.Highlight
					}
					rows = append(rows, optPrefix+optStyle.Render(opts[j]))
				}
				if len(opts) > maxShow {
					rows = append(rows, "    "+s.Dim.Render(fmt.Sprintf("(%d/%d)", cur+1, len(opts))))
				}
			}
		default:
			desc := ""
			if p.Default != "" && d.paramInputs[i].Value() == p.Default {
				desc = s.Dim.Render(" (default)")
			}
			rows = append(rows, prefix+label+d.paramInputs[i].View()+desc)
		}
	}

	rows = append(rows, "", s.Dim.Render("[tab] next  [shift+tab] prev  [up/down] select  [enter] submit  [esc] cancel"))
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (d ExecuteDialogModel) renderPreview() string {
	s := d.theme.Styles()
	label := s.Dim.Render("Command preview")
	command := d.liveRender()
	command = strings.ReplaceAll(command, "\n", " ")
	commandWidth := d.width - 10
	if commandWidth < 20 {
		commandWidth = 20
	}
	command = truncateWithEllipsis(command, commandWidth)
	line := "  " + s.Dim.Render(command)

	rows := []string{label, line}
	for len(rows) < executePreviewMinRows {
		rows = append(rows, "")
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func truncateWithEllipsis(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(r[:maxLen-1]) + "…"
}

func (d ExecuteDialogModel) viewActionMenu() string {
	s := d.theme.Styles()
	rows := []string{
		s.Highlight.Render(d.renderedCommand),
		"",
	}

	for i, action := range d.actions {
		if i == d.actionCursor {
			rows = append(rows, s.Selected.Render("❯ "+action))
		} else {
			rows = append(rows, "  "+action)
		}
	}
	rows = append(rows, "", s.Dim.Render("[up/down] select  [enter] confirm  [esc] cancel"))
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
