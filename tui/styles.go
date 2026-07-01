package tui

import "github.com/charmbracelet/lipgloss"

// Retro terminal palette — green on black, amber accents
var (
	green   = lipgloss.Color("#00FF41")
	dgreen  = lipgloss.Color("#005A00")
	amber   = lipgloss.Color("#FFB000")
	white   = lipgloss.Color("#CCCCCC")
	gray    = lipgloss.Color("#555555")
	dgray   = lipgloss.Color("#333333")
	black   = lipgloss.Color("#000000")
	red     = lipgloss.Color("#FF3333")
	cyan    = lipgloss.Color("#00FFAA")
	purple  = lipgloss.Color("#AA66FF")
	orange  = lipgloss.Color("#FF8800")
)

// Status bar
var (
	statusBarStyle = lipgloss.NewStyle().
			Background(green).
			Foreground(black).
			Padding(0, 1).
			Bold(true)

	statusBarInfoStyle = lipgloss.NewStyle().
				Foreground(black).
				Background(green)
)

// Borders and panes
var (
	agentListStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(green).
			Padding(0, 1).
			Foreground(green).
			Background(black)

	agentListTitleStyle = lipgloss.NewStyle().
				Foreground(green).
				Bold(true).
				Underline(true)

	outputStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(green).
			Padding(0, 1).
			Foreground(white).
			Background(black)

	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(amber).
			Padding(0, 1).
			Foreground(amber).
			Background(black)

	sidebarTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(gray).
			Background(black)

	sidebarTabActiveStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(black).
				Background(amber).
				Bold(true)
)

// Agent status indicators
var (
	agentRunningStyle = lipgloss.NewStyle().
				Foreground(green)

	agentStoppedStyle = lipgloss.NewStyle().
				Foreground(gray)

	agentErrorStyle = lipgloss.NewStyle().
			Foreground(red)

	agentSelectedStyle = lipgloss.NewStyle().
				Foreground(amber).
				Bold(true)
)

// Chat / Debate styles
var (
	chatStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(green).
			Padding(0, 1).
			Foreground(white).
			Background(black)

	chatUsername = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan)

	chatUserMsg = lipgloss.NewStyle().
			Foreground(amber)

	chatAgentEstratega = lipgloss.NewStyle().
				Bold(true).
				Foreground(purple)

	chatAgentCritico = lipgloss.NewStyle().
				Bold(true).
				Foreground(red)

	chatAgentAcelerador = lipgloss.NewStyle().
				Bold(true).
				Foreground(green)

	chatAgentDisenador = lipgloss.NewStyle().
				Bold(true).
				Foreground(cyan)

	chatAgentVigia = lipgloss.NewStyle().
			Bold(true).
			Foreground(orange)

	chatAgentSinte = lipgloss.NewStyle().
			Bold(true).
			Foreground(amber)
)

// Consensus meter (thermometer + donut)
var (
	consensusBarStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(green).
				Padding(0, 1).
				Foreground(green).
				Background(black).
				Height(3)

	consensusFillStyle = lipgloss.NewStyle().
				Foreground(green).
				Background(black)

	consensusEmptyStyle = lipgloss.NewStyle().
				Foreground(gray).
				Background(black)

	donutStyle = lipgloss.NewStyle().
			Foreground(amber).
			Background(black)
)

// Sidebar panel content
var (
	statsValue = lipgloss.NewStyle().
			Foreground(white).
			Background(black)

	statsLabel = lipgloss.NewStyle().
			Foreground(gray).
			Background(black)

	topicActive = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	topicPaused = lipgloss.NewStyle().
			Foreground(gray)
)

// Help and status
var (
	helpStyle = lipgloss.NewStyle().
			Foreground(gray).
			Background(black).
			Height(1).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Underline(true)

	subtleStyle = lipgloss.NewStyle().
			Foreground(gray)

	detectedStyle = lipgloss.NewStyle().
			Foreground(green)

	noDetectedStyle = lipgloss.NewStyle().
			Foreground(red)
)

// Trust prompt
var (
	trustPromptStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(amber).
				Padding(1, 2).
				Foreground(white).
				Background(black)

	trustTitleStyle = lipgloss.NewStyle().
				Foreground(amber).
				Bold(true)

	trustContentStyle = lipgloss.NewStyle().
				Foreground(white)
)

// App container
var (
	appStyle = lipgloss.NewStyle().
			Padding(0, 0).
			Background(black)
)

// Log panel
var (
	logPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(amber).
			Padding(0, 1).
			Foreground(gray).
			Background(black).
			MaxHeight(12)
)

// Question panel (5 questions from agent)
var (
	questionPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(amber).
				Padding(1, 2).
				Foreground(amber).
				Background(black)

	questionTextStyle = lipgloss.NewStyle().
				Foreground(white).
				Bold(true)

	optionStyle = lipgloss.NewStyle().
			Foreground(green)

	selectedOptionStyle = lipgloss.NewStyle().
				Foreground(amber).
				Bold(true)
)

// Mini-report from agent after context recovery
var (
	reportStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(cyan).
			Padding(1, 2).
			Foreground(cyan).
			Background(black)
)
