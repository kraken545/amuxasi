package tui

import (
	"fmt"
	"strings"
)

// Sidebar tabs
type SidebarTab int

const (
	TabStats SidebarTab = iota
	TabTopics
	TabConfig
	TabAgents
	TabKeys
)

func (t SidebarTab) String() string {
	switch t {
	case TabStats:
		return "Stats"
	case TabTopics:
		return "Topics"
	case TabConfig:
		return "Config"
	case TabAgents:
		return "Agents"
	case TabKeys:
		return "Keys"
	}
	return "?"
}

func (t SidebarTab) Icon() string {
	switch t {
	case TabStats:
		return "📊"
	case TabTopics:
		return "📋"
	case TabConfig:
		return "⚙"
	case TabAgents:
		return "🤖"
	case TabKeys:
		return "🔑"
	}
	return "?"
}

var allTabs = []SidebarTab{TabStats, TabTopics, TabConfig, TabAgents, TabKeys}

// SidebarState manages the sidebar content
type SidebarState struct {
	Visible    bool
	ActiveTab  SidebarTab
	TabNames   []string
}

func NewSidebarState() *SidebarState {
	return &SidebarState{
		Visible:   false,
		ActiveTab: TabStats,
	}
}

func (s *SidebarState) NextTab() {
	for i, t := range allTabs {
		if t == s.ActiveTab {
			s.ActiveTab = allTabs[(i+1)%len(allTabs)]
			return
		}
	}
	s.ActiveTab = TabStats
}

func (s *SidebarState) PrevTab() {
	for i, t := range allTabs {
		if t == s.ActiveTab {
			s.ActiveTab = allTabs[(i-1+len(allTabs))%len(allTabs)]
			return
		}
	}
	s.ActiveTab = TabStats
}

// RenderTabs renders the tab headers
func (s *SidebarState) RenderTabs(width int) string {
	var b strings.Builder
	for _, t := range allTabs {
		tabStr := fmt.Sprintf(" %s %s ", t.Icon(), t.String())
		if t == s.ActiveTab {
			b.WriteString(sidebarTabActiveStyle.Render(tabStr))
		} else {
			b.WriteString(sidebarTabStyle.Render(tabStr))
		}
	}
	return b.String()
}

// RenderTabContent renders the active tab's content
func (s *SidebarState) RenderTabContent(session *DebateSession, cs ConsensusState, settings *UISettings) string {
	switch s.ActiveTab {
	case TabStats:
		return s.renderStatsTab(session, cs)
	case TabTopics:
		return s.renderTopicsTab(session)
	case TabConfig:
		return s.renderConfigTab(settings)
	case TabAgents:
		return s.renderAgentsTab(session)
	case TabKeys:
		return s.renderKeysTab()
	}
	return ""
}

func (s *SidebarState) renderStatsTab(session *DebateSession, cs ConsensusState) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" 📊 Estadísticas"))
	b.WriteString("\n\n")

	if session != nil {
		b.WriteString(statsLabel.Render(" Debate:"))
		if session.Active {
			b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(" Activo")))
		} else {
			b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(" Inactivo")))
		}

		b.WriteString(fmt.Sprintf("\n%s", statsLabel.Render(" Tema:")))
		if session.Topic != "" {
			b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(" "+truncate(session.Topic, 25))))
		} else {
			b.WriteString(fmt.Sprintf("\n%s", subtleStyle.Render(" Sin tema")))
		}

		b.WriteString(fmt.Sprintf("\n\n%s", statsLabel.Render(" Mensajes:")))
		b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(fmt.Sprintf(" %d", len(session.Messages)))))

		b.WriteString(fmt.Sprintf("\n\n%s", statsLabel.Render(" Agentes:")))
		b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(fmt.Sprintf(" %d", len(session.AgentCtx)))))
	}

	b.WriteString(fmt.Sprintf("\n\n%s", statsLabel.Render(" Consenso:")))
	b.WriteString(fmt.Sprintf("\n%s %d%%", consensusFillStyle.Render("●"), cs.AgreeCount*100/max(1, cs.TotalAgents)))

	b.WriteString(fmt.Sprintf("\n\n%s", statsLabel.Render(" Contexto promedio:")))
	b.WriteString(fmt.Sprintf("\n%s %d%%", consensusFillStyle.Render("●"), cs.AvgContextPct))

	return b.String()
}

func (s *SidebarState) renderTopicsTab(session *DebateSession) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" 📋 Temas"))
	b.WriteString("\n\n")

	if session == nil || session.Topic == "" {
		b.WriteString(subtleStyle.Render(" Sin temas activos"))
		b.WriteString("\n\n")
		b.WriteString(subtleStyle.Render(" Presiona 'D' para"))
		b.WriteString("\n")
		b.WriteString(subtleStyle.Render(" iniciar un debate"))
		return b.String()
	}

	b.WriteString(topicActive.Render(" ▸ Activo:"))
	b.WriteString(fmt.Sprintf("\n   %s", statsValue.Render(session.Topic)))
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render(" [Enter] para archivar"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(" [X] para cerrar debate"))

	return b.String()
}

func (s *SidebarState) renderConfigTab(settings *UISettings) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" ⚙ Config"))
	b.WriteString("\n\n")

	b.WriteString(statsLabel.Render(" Tema visual:"))
	b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(" Retro Terminal")))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("\n%s", statsLabel.Render(" Layout:")))
	b.WriteString(fmt.Sprintf("\n%s", statsValue.Render(" Opción A")))

	b.WriteString(fmt.Sprintf("\n\n%s", statsLabel.Render(" Sidebar:")))
	if settings.Sidebar.Visible {
		b.WriteString(fmt.Sprintf("\n%s", agentRunningStyle.Render(" Visible")))
	} else {
		b.WriteString(fmt.Sprintf("\n%s", subtleStyle.Render(" Oculta")))
	}

	b.WriteString(fmt.Sprintf("\n\n%s", subtleStyle.Render(" Edita amuxasi.toml")))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(" para más opciones"))

	return b.String()
}

func (s *SidebarState) renderAgentsTab(session *DebateSession) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" 🤖 Roles"))
	b.WriteString("\n\n")

	if session == nil || len(session.AgentCtx) == 0 {
		b.WriteString(subtleStyle.Render(" Sin agentes en debate"))
		return b.String()
	}

	for _, ac := range session.AgentCtx {
		roleStr := fmt.Sprintf("[%s]", RoleShortName(ac.Role))
		b.WriteString(fmt.Sprintf(" %s %s → %s\n",
			roleStyleFromRole(ac.Role).Render(roleStr),
			ac.AgentName,
			statsLabel.Render(string(ac.Role)),
		))
		b.WriteString(fmt.Sprintf("   %s %d%%\n",
			subtleStyle.Render("Contexto:"),
			ac.ContextPct,
		))
	}

	return b.String()
}

func (s *SidebarState) renderKeysTab() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" 🔑 API Keys"))
	b.WriteString("\n\n")

	b.WriteString(statsLabel.Render(" Usa variables de entorno:"))
	b.WriteString("\n")
	b.WriteString(statsValue.Render(" ANTHROPIC_API_KEY"))
	b.WriteString("\n")
	b.WriteString(statsValue.Render(" OPENAI_API_KEY"))
	b.WriteString("\n")
	b.WriteString(statsValue.Render(" GEMINI_API_KEY"))
	b.WriteString("\n")
	b.WriteString(statsValue.Render(" OPENROUTER_API_KEY"))
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render(" Define en ~/.bashrc"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(" o presiona Ctrl+K"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(" para formulario."))

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
