package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary  = lipgloss.Color("63")
	colorMuted    = lipgloss.Color("240")
	colorDone     = lipgloss.Color("34")
	colorHigh     = lipgloss.Color("196")
	colorCarried  = lipgloss.Color("214")
	colorSelected = lipgloss.Color("63")
	colorBorder   = lipgloss.Color("238")

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			Underline(true).
			Padding(0, 1)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)

	stylePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	stylePaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	styleTitle = lipgloss.NewStyle().Bold(true)

	styleDone    = lipgloss.NewStyle().Foreground(colorDone).Strikethrough(true)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleHigh    = lipgloss.NewStyle().Foreground(colorHigh)
	styleCarried = lipgloss.NewStyle().Foreground(colorCarried)
	styleLabel   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	styleSelected = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Bold(true)

	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)
)

func labelBadge(label string) string {
	return styleLabel.Render(label)
}
