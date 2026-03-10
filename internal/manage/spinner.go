package manage

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const spinnerInterval = 100 * time.Millisecond

var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

type spinnerScope string

const (
	spinnerScopeGlobal spinnerScope = "global"
	spinnerScopeForm   spinnerScope = "form"
)

type spinnerTickMsg struct {
	scope spinnerScope
}

func spinnerTickCmd(scope spinnerScope) tea.Cmd {
	return tea.Tick(spinnerInterval, func(time.Time) tea.Msg {
		return spinnerTickMsg{scope: scope}
	})
}

func spinnerFrame(index int) string {
	if len(spinnerFrames) == 0 {
		return ""
	}
	if index < 0 {
		index = 0
	}
	return spinnerFrames[index%len(spinnerFrames)]
}

func nextSpinnerFrame(index int) int {
	if len(spinnerFrames) == 0 {
		return 0
	}
	return (index + 1) % len(spinnerFrames)
}
