package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	PercentStyle = lipgloss.NewStyle().
			Width(8).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("170"))

	ResetTimeStyle = lipgloss.NewStyle().
			Width(20).
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("241")).
			MarginLeft(2)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	GroupHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				MarginTop(1)

	ClaudeSubheaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	GeminiSubheaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)

	OtherSubheaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Bold(true)
)
