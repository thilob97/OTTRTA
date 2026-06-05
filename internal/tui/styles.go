package tui

import "github.com/charmbracelet/lipgloss"

var (
	basePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	focusedPanelStyle = basePanelStyle.Copy().
				BorderForeground(lipgloss.Color("63"))

	blurredPanelStyle = basePanelStyle.Copy().
				BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	selectedSessionStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	stoppedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
)

func panelStyle(focused bool) lipgloss.Style {
	if focused {
		return focusedPanelStyle
	}
	return blurredPanelStyle
}
