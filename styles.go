package main

import "github.com/charmbracelet/lipgloss"

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true)
	
	// Fancy help bar style
	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#313244")).
			Padding(0, 1).
			MarginTop(1)
	
	// Individual key styles for the help bar
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Bold(true)
	
	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4"))
	
	helpSeparatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086"))
)
