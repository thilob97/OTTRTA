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

	sessionCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 1)

	selectedSessionCardStyle = sessionCardStyle.Copy().
					BorderForeground(lipgloss.Color("63")).
					Background(lipgloss.Color("235"))

	attentionSessionCardStyle = sessionCardStyle.Copy().
					BorderForeground(lipgloss.Color("208"))

	selectedAttentionSessionCardStyle = selectedSessionCardStyle.Copy().
						BorderForeground(lipgloss.Color("208"))

	attentionBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("208"))

	// Terminal panel: same background as session panel
	baseTermStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	focusedTermStyle = baseTermStyle.Copy().
				BorderForeground(lipgloss.Color("63"))

	blurredTermStyle = baseTermStyle.Copy().
				BorderForeground(lipgloss.Color("240"))

	// Attach mode: full-screen terminal
	attachPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Background(lipgloss.Color("0"))
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	selectedSessionStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	stoppedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	failedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func panelStyle(focused bool) lipgloss.Style {
	if focused {
		return focusedPanelStyle
	}
	return blurredPanelStyle
}

func termPanelStyle(focused bool) lipgloss.Style {
	if focused {
		return focusedTermStyle
	}
	return blurredTermStyle
}
