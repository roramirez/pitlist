package views

import "github.com/charmbracelet/lipgloss"

var (
	sTitle    = lipgloss.NewStyle().Bold(true)
	sMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	sCarried  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sAccent   = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	sSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	sDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Strikethrough(true)
)
