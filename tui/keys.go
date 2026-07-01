package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Tab           key.Binding
	ShiftTab      key.Binding
	Enter         key.Binding

	// Agent control
	Launch        key.Binding
	Stop          key.Binding
	Restart       key.Binding
	Attach        key.Binding

	// Chat / Debate
	ChatInput     key.Binding
	ChatSend      key.Binding
	DebateStart   key.Binding
	DebateStop    key.Binding
	AgentQuestion key.Binding

	// Sidebar
	SidebarToggle key.Binding
	SidebarNext   key.Binding
	SidebarPrev   key.Binding

	// Display
	ToggleLogs    key.Binding
	Help          key.Binding

	// Session
	Detach        key.Binding
	Quit          key.Binding

	// Scripts
	RunSetup      key.Binding
	RunArchive    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Tab, k.ChatInput, k.Launch, k.Stop,
		k.Attach, k.SidebarToggle, k.Help, k.Quit,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab, k.ShiftTab, k.Enter},
		{k.Launch, k.Stop, k.Restart, k.Attach},
		{k.ChatInput, k.ChatSend, k.DebateStart, k.DebateStop, k.AgentQuestion},
		{k.SidebarToggle, k.SidebarNext, k.SidebarPrev},
		{k.ToggleLogs, k.Detach, k.Help, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next section"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("⇧+tab", "prev section"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select / send"),
	),
	Launch: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "launch agent"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop agent"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart agent"),
	),
	Attach: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "attach tmux"),
	),
	ChatInput: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "chat input"),
	),
	ChatSend: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send message"),
	),
	DebateStart: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "start debate"),
	),
	DebateStop: key.NewBinding(
		key.WithKeys("X"),
		key.WithHelp("X", "stop debate"),
	),
	AgentQuestion: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "ask agent"),
	),
	SidebarToggle: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "toggle sidebar"),
	),
	SidebarNext: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next sidebar tab"),
	),
	SidebarPrev: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "prev sidebar tab"),
	),
	ToggleLogs: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("^L", "toggle logs"),
	),
	Help: key.NewBinding(
		key.WithKeys("F1"),
		key.WithHelp("F1", "help"),
	),
	Detach: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "detach (leave running)"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("^C/q", "quit"),
	),
	RunSetup: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "run setup"),
	),
	RunArchive: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "run archive"),
	),
}
