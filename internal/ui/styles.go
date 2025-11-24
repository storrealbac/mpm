package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Standard Colors
	primaryColor = lipgloss.Color("#FF9F1C") // Vibrant Orange
	successColor = lipgloss.Color("#2ECC71") // Emerald Green
	errorColor   = lipgloss.Color("#E74C3C") // Alizarin Red
	warningColor = lipgloss.Color("#F1C40F") // Sunflower Yellow
	infoColor    = lipgloss.Color("#3498DB") // Peter River Blue
	grayColor    = lipgloss.Color("#95A5A6") // Concrete Gray
	whiteColor   = lipgloss.Color("#FFFFFF") // White

	// Text Styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(whiteColor).
			Background(primaryColor).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true)

	DetailStyle = lipgloss.NewStyle().
			Foreground(grayColor)

	// Badge Styles
	SuccessBadge = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			MarginRight(1)

	ErrorBadge = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			MarginRight(1)

	WarningBadge = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true).
			MarginRight(1)

	InfoBadge = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true).
			MarginRight(1)

	// Box Style
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(grayColor).
			Padding(1, 2).
			Margin(1, 0)

	// Table Styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Padding(0, 1)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Separator Styles
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(grayColor)

	// CLI Section Style
	SectionStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// MPM Brand Style
	MPMStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
)
