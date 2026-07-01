package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/amuxasi/amuxasi/agent"
	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/trust"
	"github.com/amuxasi/amuxasi/workspace"
)

// UISettings holds user-customizable settings
type UISettings struct {
	Sidebar *SidebarState
}

type focusSection int

const (
	focusAgentList focusSection = iota
	focusChat
	focusSidebar
	focusQuestion
)

type trustState int

const (
	trustNone trustState = iota
	trustPrompting
	trustViewing
)

type outputTick struct{}

type statusMessage struct {
	text string
}

type model struct {
	// Core
	workspaceMgr *workspace.Manager
	agents       []*agent.Agent
	trustStore   *trust.Store

	// Nav
	focus       focusSection
	selectedIdx int
	err         error

	// UI state
	width, height int
	ready         bool
	showHelp      bool
	showLogs      bool
	statusMsg     string

	// Sidebar
	sidebar  *SidebarState
	settings *UISettings

	// Debate
	session   *DebateSession
	consensus ConsensusState

	// Chat
	chatInput   string
	chatFocused bool

	// Diagnostic
	diagnostic *AgentDiagnostic
	diagStep   int // current question index

	// Logs
	logLines      []string
	outputHistory map[string][]string

	// Detected
	detectedAgents []agent.DetectedAgent

	// Trust
	trustState   trustState
	trustScript  string
	trustContent string

	statusTimer *time.Timer
}

func NewModel(ws *workspace.Manager, trustStore *trust.Store) *model {
	m := &model{
		workspaceMgr:  ws,
		agents:        []*agent.Agent{},
		selectedIdx:   0,
		trustStore:    trustStore,
		outputHistory: make(map[string][]string),
		statusMsg:     "Ready",
		logLines:      log.Global().Lines(),
		focus:         focusAgentList,
		sidebar:       NewSidebarState(),
		settings: &UISettings{
			Sidebar: NewSidebarState(),
		},
		detectedAgents: agent.DetectAgents(),
		session:        NewDebateSession(""),
	}

	detected := m.detectedAgents
	detectedMap := make(map[string]bool)
	for _, d := range detected {
		detectedMap[d.Name] = true
	}

	// Load configured agents
	for _, name := range ws.Cfg.Launch {
		if cfg, ok := ws.Cfg.Agents[name]; ok {
			a := agent.New(name, cfg.Command, ws.Cfg.Workspace.Name, cfg.Args, cfg.Env)
			a.CheckStatus()
			m.agents = append(m.agents, a)
		}
	}

	// Add detected agents not already configured
	for _, d := range detected {
		found := false
		for _, a := range m.agents {
			if a.Name == d.Name {
				found = true
				break
			}
		}
		if !found {
			a := agent.New(d.Name, d.Command, ws.Cfg.Workspace.Name, nil, nil)
			a.CheckStatus()
			m.agents = append(m.agents, a)
		}
	}

	// Init agent contexts for debate
	UpdateAllAgentContext(m.agents, m.session)

	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond * 500, func(t time.Time) tea.Msg {
		return outputTick{}
	})
}

// ---------- Update ----------

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		if m.trustState != trustNone {
			return m, m.handleTrustKey(msg)
		}
		if m.diagnostic != nil && !m.diagnostic.Complete {
			cmds = append(cmds, m.handleDiagKey(msg))
			return m, tea.Batch(cmds...)
		}
		cmds = append(cmds, m.handleKey(msg))

	case outputTick:
		m.pollAgents()
		m.pollDebate()
		m.logLines = log.Global().Lines()
		cmds = append(cmds, tickCmd())

	case statusMessage:
		m.statusMsg = msg.text
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Quit):
		return tea.Quit

	case key.Matches(msg, keys.Up):
		switch m.focus {
		case focusAgentList:
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}

	case key.Matches(msg, keys.Down):
		switch m.focus {
		case focusAgentList:
			if m.selectedIdx < len(m.agents)-1 {
				m.selectedIdx++
			}
		}

	case key.Matches(msg, keys.Tab):
		m.cycleFocus()

	case key.Matches(msg, keys.ShiftTab):
		m.cycleFocusReverse()

	case key.Matches(msg, keys.Launch):
		if m.selectedIdx < len(m.agents) {
			a := m.agents[m.selectedIdx]
			if !a.IsRunning() {
				m.setStatus(fmt.Sprintf("Launching %s...", a.Name))
				if err := a.Launch(); err != nil {
					m.err = err
					m.setStatus(fmt.Sprintf("Error: %s", err))
				} else {
					m.setStatus(fmt.Sprintf("%s launched • tmux: %s", a.Name, a.Session))
					log.Info("launched agent %s", a.Name)
				}
			} else {
				m.setStatus(fmt.Sprintf("%s is already running", a.Name))
			}
		}

	case key.Matches(msg, keys.Stop):
		if m.selectedIdx < len(m.agents) {
			a := m.agents[m.selectedIdx]
			if a.IsRunning() {
				m.setStatus(fmt.Sprintf("Stopping %s...", a.Name))
				if err := a.Stop(); err != nil {
					m.setStatus(fmt.Sprintf("Error: %s", err))
				} else {
					m.setStatus(fmt.Sprintf("%s stopped", a.Name))
					log.Info("stopped agent %s", a.Name)
				}
			}
		}

	case key.Matches(msg, keys.Restart):
		if m.selectedIdx < len(m.agents) {
			a := m.agents[m.selectedIdx]
			m.setStatus(fmt.Sprintf("Restarting %s...", a.Name))
			if a.IsRunning() {
				a.Stop()
			}
			time.Sleep(200 * time.Millisecond)
			if err := a.Launch(); err != nil {
				m.setStatus(fmt.Sprintf("Error: %s", err))
			} else {
				m.setStatus(fmt.Sprintf("%s restarted", a.Name))
				log.Info("restarted agent %s", a.Name)
			}
		}

	case key.Matches(msg, keys.Attach):
		if m.selectedIdx < len(m.agents) {
			a := m.agents[m.selectedIdx]
			if a.IsRunning() {
				m.setStatus(fmt.Sprintf("Attaching to %s...", a.Name))
				log.Info("attaching to agent %s", a.Name)
				cmd := exec.Command("tmux", "attach-session", "-t", a.Session)
				return tea.ExecProcess(cmd, nil)
			}
		}

	case key.Matches(msg, keys.Detach):
		m.setStatus("Detached — agents keep running")
		log.Info("user detached")
		return tea.Quit

	case key.Matches(msg, keys.ChatInput):
		m.focus = focusChat
		m.chatFocused = true
		m.setStatus("Chat input — type and press Enter")

	case key.Matches(msg, keys.Enter):
		if m.focus == focusChat && m.chatFocused && m.chatInput != "" {
			m.session.AddUserMsg(m.chatInput)
			log.Info("user: %s", m.chatInput)
			m.chatInput = ""
			m.chatFocused = false
			m.focus = focusChat
			m.triggerAgentResponses()
		}

	case key.Matches(msg, keys.DebateStart):
		if !m.session.Active {
			m.session.Active = true
			m.session.StartedAt = time.Now()
			topic := m.chatInput
			if topic == "" {
				topic = "Debate general del proyecto"
			}
			m.session.Topic = topic
			m.session.AddSystemMsg(fmt.Sprintf("Debate iniciado: %s", topic))
			m.setStatus(fmt.Sprintf("Debate started: %s", topic))
			log.Info("debate started: %s", topic)
			m.triggerAgentResponses()
		}

	case key.Matches(msg, keys.DebateStop):
		if m.session.Active {
			m.session.AddSystemMsg("Debate finalizado por el usuario")
			m.session.Active = false
			m.setStatus("Debate stopped")
			log.Info("debate stopped by user")
		}

	case key.Matches(msg, keys.AgentQuestion):
		if m.selectedIdx < len(m.agents) {
			agentName := m.agents[m.selectedIdx].Name
			if m.session != nil {
				for _, ac := range m.session.AgentCtx {
					if ac.AgentName == agentName && ac.ContextPct < 70 {
						m.diagnostic = NewAgentDiagnostic(agentName)
						m.diagStep = 0
						m.focus = focusQuestion
						m.setStatus(fmt.Sprintf("%s needs context — answer the questions", agentName))
						return nil
					}
				}
			}
			m.setStatus(fmt.Sprintf("%s has sufficient context", agentName))
		}

	case key.Matches(msg, keys.SidebarToggle):
		m.sidebar.Visible = !m.sidebar.Visible
		if m.sidebar.Visible {
			m.focus = focusSidebar
		}

	case key.Matches(msg, keys.SidebarNext):
		if m.sidebar.Visible {
			m.sidebar.NextTab()
		}

	case key.Matches(msg, keys.SidebarPrev):
		if m.sidebar.Visible {
			m.sidebar.PrevTab()
		}

	case key.Matches(msg, keys.ToggleLogs):
		m.showLogs = !m.showLogs

	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp

	case key.Matches(msg, keys.RunSetup):
		return m.handleRunScript("setup")

	case key.Matches(msg, keys.RunArchive):
		return m.handleRunScript("archive")
	}

	return nil
}

func (m *model) handleDiagKey(msg tea.KeyMsg) tea.Cmd {
	if m.diagnostic == nil || m.diagStep >= len(m.diagnostic.Questions) {
		return nil
	}

	q := &m.diagnostic.Questions[m.diagStep]
	num := -1
	for i := 0; i < len(q.Options); i++ {
		if msg.String() == fmt.Sprintf("%d", i+1) {
			num = i
			break
		}
	}

	if num >= 0 && num < len(q.Options) {
		q.Answer = q.Options[num]
		m.diagnostic.Answers++
		m.diagStep++

		if m.diagStep >= len(m.diagnostic.Questions) {
			m.diagnostic.Complete = true
			m.diagnostic.generateReport()
			m.setStatus("Diagnostic complete")
			// Update agent context
			for i, ac := range m.session.AgentCtx {
				if ac.AgentName == m.diagnostic.AgentName {
					m.session.AgentCtx[i].ContextPct = 85
					m.session.AgentCtx[i].Vote = VoteAgree
					break
				}
			}
			m.focus = focusAgentList
		}
		return nil
	}

	return nil
}

func (m *model) cycleFocus() {
	switch m.focus {
	case focusAgentList:
		m.focus = focusChat
	case focusChat:
		if m.sidebar.Visible {
			m.focus = focusSidebar
		} else {
			m.focus = focusAgentList
		}
	case focusSidebar:
		m.focus = focusAgentList
	}
}

func (m *model) cycleFocusReverse() {
	switch m.focus {
	case focusAgentList:
		if m.sidebar.Visible {
			m.focus = focusSidebar
		} else {
			m.focus = focusChat
		}
	case focusChat:
		m.focus = focusAgentList
	case focusSidebar:
		m.focus = focusChat
	}
}

func (m *model) triggerAgentResponses() {
	if !m.session.Active {
		return
	}
	// Simulate agent responses (in production, these would call real APIs)
	for _, ac := range m.session.AgentCtx {
		if ac.Vote != VoteWait {
			msg := fmt.Sprintf("Analizando el tema desde mi rol como %s...", string(ac.Role))
			m.session.AddAgentMsg(ac.AgentName, ac.Role, msg)
			m.session.UpdateVote(ac.AgentName, VoteAgree, ac.ContextPct, "Analizando...")
		}
	}
	m.consensus = m.session.CalcConsensus()
}

func (m *model) pollDebate() {
	if m.session == nil || !m.session.Active {
		return
	}
	UpdateAllAgentContext(m.agents, m.session)

	// Simulate context drift / updates
	for i := range m.session.AgentCtx {
		ac := &m.session.AgentCtx[i]
		if ac.Vote == VoteAgree {
			ac.ContextPct = min(100, ac.ContextPct+1)
		}
	}
	m.consensus = m.session.CalcConsensus()
}

// ---------- View ----------

func (m *model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.showHelp {
		return m.helpView()
	}
	if m.trustState != trustNone {
		return m.trustView()
	}

	// Status bar
	statusBar := m.statusBarView()

	// Main content area (agent list + chat)
	mainContent := m.mainView()

	// Bottom consensus bar
	consensusBar := m.consensusBarView()

	// Help bar
	helpBar := m.helpBarView()

	// Put it all together
	height := m.height - 2

	// If logs are visible, add log panel at the bottom
	if m.showLogs {
		logPanel := m.logPanelView()
		logHeight := min(12, height/3)
		mainH := height - logHeight - 1
		body := lipgloss.JoinVertical(lipgloss.Top,
			m.resizeContent(mainContent, mainH),
			logPanel,
		)
		return appStyle.Render(lipgloss.JoinVertical(lipgloss.Top,
			statusBar, body, consensusBar, helpBar,
		))
	}

	body := m.resizeContent(mainContent, height-1)
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Top,
		statusBar, body, consensusBar, helpBar,
	))
}

func (m *model) resizeContent(content string, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		return strings.Join(lines[len(lines)-height:], "\n")
	}
	return content
}

func (m *model) statusBarView() string {
	running := 0
	for _, a := range m.agents {
		if a.IsRunning() {
			running++
		}
	}

	left := fmt.Sprintf(" AMUXASI  ◆  %s", m.workspaceMgr.Cfg.Workspace.Name)

	// Focus indicator
	var focusStr string
	switch m.focus {
	case focusAgentList:
		focusStr = "[Agents]"
	case focusChat:
		focusStr = "[Chat]"
	case focusSidebar:
		focusStr = "[Sidebar]"
	case focusQuestion:
		focusStr = "[?]"
	}

	middle := fmt.Sprintf(" %s  Agents: %d/%d", focusStr, running, len(m.agents))
	right := fmt.Sprintf(" %s", m.statusMsg)

	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right) - 6
	if spacing < 1 {
		spacing = 1
	}
	pad := strings.Repeat(" ", spacing)

	return statusBarStyle.Render(left + middle + pad + right)
}

func (m *model) mainView() string {
	if m.diagnostic != nil && !m.diagnostic.Complete {
		return m.questionPanelView()
	}

	// Left column: agent list + sidebar tabs
	leftColumn := m.leftColumnView()

	// Right column: chat/debate
	chatPanel := m.chatPanelView()

	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, chatPanel)
}

func (m *model) leftColumnView() string {
	// Determine available height
	availHeight := m.height - 5 // minus status, consensus, help, borders
	agentListH := availHeight * 2 / 3
	sidebarH := availHeight - agentListH

	agentList := m.agentListView()
	agentList = agentListStyle.
		Width(m.leftColWidth()).
		Height(agentListH).
		Render(agentList)

	var sidebarContent string
	if m.sidebar.Visible {
		tabHeaders := m.sidebar.RenderTabs(m.leftColWidth() - 4)
		tabContent := m.sidebar.RenderTabContent(m.session, m.consensus, m.settings)
		sidebarContent = sidebarStyle.
			Width(m.leftColWidth()).
			Height(sidebarH).
			Render(tabHeaders + "\n" + tabContent)
	} else {
		sidebarContent = sidebarStyle.
			Width(m.leftColWidth()).
			Height(sidebarH).
			Render(subtleStyle.Render(" Presiona 'b' para\n la sidebar"))
	}

	return lipgloss.JoinVertical(lipgloss.Top, agentList, sidebarContent)
}

func (m *model) leftColWidth() int {
	w := 32
	if m.width < 90 {
		w = 24
	}
	if m.width > 140 {
		w = 38
	}
	return w
}

func (m *model) agentListView() string {
	if len(m.agents) == 0 {
		var b strings.Builder
		b.WriteString(agentListTitleStyle.Render(" Agentes"))
		b.WriteString("\n\n")
		b.WriteString(noDetectedStyle.Render(" No hay agentes"))
		b.WriteString("\n\n")
		if len(m.detectedAgents) > 0 {
			b.WriteString(detectedStyle.Render(" Detectados:"))
			b.WriteString("\n")
			for _, d := range m.detectedAgents {
				b.WriteString(fmt.Sprintf("  ✓ %s\n", d.Name))
			}
		}
		if m.focus == focusAgentList {
			b.WriteString("\n" + subtleStyle.Render(" l: lanzar"))
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString(agentListTitleStyle.Render(" Agentes"))
	b.WriteString("\n\n")

	for i, a := range m.agents {
		prefix := " "
		if i == m.selectedIdx && m.focus == focusAgentList {
			prefix = "▸"
		}

		// Find context in debate
		ctxPct := 50
		role := RoleSynthesizer
		if m.session != nil {
			for _, ac := range m.session.AgentCtx {
				if ac.AgentName == a.Name {
					ctxPct = ac.ContextPct
					role = ac.Role
					break
				}
			}
		}

		var statusChar string
		var statusColor lipgloss.Style
		switch a.CheckStatus() {
		case agent.StatusRunning:
			statusChar = "●"
			statusColor = agentRunningStyle
		case agent.StatusStopped:
			statusChar = "○"
			statusColor = agentStoppedStyle
		case agent.StatusError:
			statusChar = "✕"
			statusColor = agentErrorStyle
		}

		roleTag := fmt.Sprintf("[%s]", RoleShortName(role))
		ctxBar := renderContextBar(ctxPct)

		line := fmt.Sprintf("%s %s %s %s %s",
			prefix,
			statusColor.Render(statusChar),
			roleStyleFromRole(role).Render(roleTag),
			a.Name,
			ctxBar,
		)
		if i == m.selectedIdx && m.focus == focusAgentList {
			line = agentSelectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m *model) chatPanelView() string {
	availWidth := m.width - m.leftColWidth() - 6
	if availWidth < 30 {
		availWidth = 30
	}
	availHeight := m.height - 5

	// Count diagnostic panel height if active
	questionH := 0
	if m.diagnostic != nil && !m.diagnostic.Complete {
		questionH = 8
	}

	chatH := availHeight - questionH

	var content string
	if m.session != nil && len(m.session.Messages) > 0 {
		maxLines := chatH - 4
		if maxLines < 5 {
			maxLines = 5
		}
		content = m.session.RenderChat(maxLines)
	} else {
		content = " " + titleStyle.Render("Chat / Debate")
		content += "\n\n"
		content += subtleStyle.Render(" Presiona 'i' para escribir un mensaje")
		content += "\n"
		content += subtleStyle.Render(" Presiona 'D' para iniciar un debate")
		content += "\n\n"
		if m.session != nil && m.session.Topic != "" {
			content += fmt.Sprintf(" Último tema: %s\n", m.session.Topic)
		}
	}

	// Chat input area
	if m.chatFocused {
		inputLine := fmt.Sprintf(" > %s█", m.chatInput)
		content += "\n\n" + chatUserMsg.Render(inputLine)
	}

	chatPanel := chatStyle.
		Width(availWidth).
		Height(availHeight).
		Render(content)

	return chatPanel
}

func (m *model) questionPanelView() string {
	if m.diagnostic == nil {
		return ""
	}

	return RenderQuestionPanel(m.diagnostic)
}

func (m *model) consensusBarView() string {
	cs := m.consensus
	// Render thermometer
	meter := RenderConsensusMeter(cs)
	// Render donut
	donut := RenderDonut(cs)

	content := fmt.Sprintf(" %s  |  %s", meter, donut)
	return consensusBarStyle.
		Width(m.width - 2).
		Render(content)
}

func (m *model) helpBarView() string {
	return helpStyle.Render(
		" Tab:Next  i:Chat  l:Launch  s:Stop  a:Attach  b:Sidebar  ?:Help  q:Quit",
	)
}

func (m *model) logPanelView() string {
	lines := m.logLines
	if len(lines) > 30 {
		lines = lines[len(lines)-30:]
	}
	content := strings.Join(lines, "\n")
	if content == "" {
		content = " (no log entries)"
	}
	return logPanelStyle.
		Width(m.width - 4).
		Render(" Logs\n" + content)
}

func (m *model) helpView() string {
	content := fmt.Sprintf(`%s

  Navigation:
    Tab            Cycle sections (agents → chat → sidebar)
    ↑/k  ↓/j      Navigate agent list

  Agent Control:
    l              Launch selected agent
    s              Stop selected agent
    r              Restart selected agent
    a              Attach to tmux session
    ?              Ask agent diagnostic questions

  Chat / Debate:
    i              Enter chat input
    Enter          Send message
    D              Start debate
    X              Stop debate

  Sidebar:
    b              Toggle sidebar
    ]/[            Next/prev tab

  Display:
    Ctrl+L         Toggle log panel
    F1             Toggle this help

  Session:
    d              Detach (agents keep running)
    q / Ctrl+C     Quit

  Scripts:
    S              Run setup script
    A              Run archive script

  Workspace:
    amuxasi init        Create config
    amuxasi add-worktree  Create git worktree
`,
		titleStyle.Render("Amuxasi v0.2 — Multi-Agent Dashboard"),
	)

	return trustPromptStyle.
		Width(m.width - 6).
		Height(m.height - 4).
		Render(content)
}

func (m *model) trustView() string {
	scriptName := m.trustScript
	if idx := strings.LastIndex(scriptName, "/"); idx >= 0 {
		scriptName = scriptName[idx+1:]
	}

	var content string
	if m.trustState == trustViewing {
		content = m.trustContent
		if len(content) > 2000 {
			content = content[:2000] + "\n... (truncated)"
		}
		content = fmt.Sprintf(" Full script content:\n\n%s\n\n Press enter to return", content)
	} else {
		lines := strings.Split(m.trustContent, "\n")
		if len(lines) > 20 {
			lines = lines[:20]
		}
		preview := strings.Join(lines, "\n")
		if len(m.trustContent) > 1000 {
			preview += "\n..."
		}
		content = fmt.Sprintf(
			" Script: %s\n\n%s\n\n (y) Trust and run   (v) View full   (n/N) Reject",
			trustTitleStyle.Render(scriptName),
			trustContentStyle.Render(preview),
		)
	}

	return trustPromptStyle.
		Width(m.width - 6).
		Render(content)
}

// ---------- Helpers ----------

func (m *model) pollAgents() {
	for _, a := range m.agents {
		output := a.RefreshOutput()
		if output != "" {
			history := m.outputHistory[a.Name]
			if len(history) == 0 || history[len(history)-1] != output {
				m.outputHistory[a.Name] = append(history, output)
				if len(m.outputHistory[a.Name]) > 500 {
					m.outputHistory[a.Name] = m.outputHistory[a.Name][250:]
				}
			}
		}
		a.CheckStatus()
	}
}

func (m *model) setStatus(text string) {
	m.statusMsg = text
	if m.statusTimer != nil {
		m.statusTimer.Stop()
	}
	m.statusTimer = time.AfterFunc(5*time.Second, func() {
		m.statusMsg = "Ready"
	})
	log.Debug("status: %s", text)
}

func (m *model) handleRunScript(scriptName string) tea.Cmd {
	scriptPath, err := m.workspaceMgr.ScriptPath(scriptName)
	if err != nil {
		m.setStatus(fmt.Sprintf("No %s script configured", scriptName))
		return nil
	}
	if !m.trustStore.IsApproved(scriptPath) {
		data, err := os.ReadFile(scriptPath)
		if err != nil {
			m.setStatus(fmt.Sprintf("Error reading script: %s", err))
			return nil
		}
		m.trustState = trustPrompting
		m.trustScript = scriptPath
		m.trustContent = string(data)
		return nil
	}
	m.setStatus(fmt.Sprintf("Running %s script...", scriptName))
	log.Info("running %s script: %s", scriptName, scriptPath)
	if err := m.workspaceMgr.RunScript(scriptName); err != nil {
		m.setStatus(fmt.Sprintf("Script error: %s", err))
		log.Error("script %s failed: %s", scriptName, err)
	} else {
		m.setStatus(fmt.Sprintf("%s script completed", scriptName))
	}
	return nil
}

func (m *model) handleTrustKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		if err := m.trustStore.Approve(m.trustScript); err != nil {
			m.setStatus(fmt.Sprintf("Error saving approval: %s", err))
		} else {
			m.setStatus("Script approved and running")
			log.Info("script approved: %s", m.trustScript)
		}
		m.trustState = trustNone
		go m.workspaceMgr.RunScript(filepathBase(m.trustScript))
	case "v", "V":
		m.trustState = trustViewing
	case "n", "N", "esc":
		m.setStatus("Script rejected")
		m.trustState = trustNone
	case "enter":
		if m.trustState == trustViewing {
			m.trustState = trustPrompting
		}
	}
	return nil
}

func filepathBase(p string) string {
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		return p[idx+1:]
	}
	return p
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
