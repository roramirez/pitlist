package views

import "github.com/charmbracelet/lipgloss"

const (
	colorBorderInactive = lipgloss.Color("238")
	colorBorderActive   = lipgloss.Color("63")
)

var (
	sTitle    = lipgloss.NewStyle().Bold(true)
	sMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	sCarried  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sAccent   = lipgloss.NewStyle().Foreground(colorBorderActive)
	sSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	sDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Strikethrough(true)

	sPaneBase = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderInactive).
			Padding(0, 1)
	sPaneInactive = sPaneBase
	sPaneActive   = sPaneBase.BorderForeground(colorBorderActive)
)
