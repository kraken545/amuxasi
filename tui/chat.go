package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/amuxasi/amuxasi/agent"
)

// AgentRole defines the debate role for an agent
type AgentRole string

const (
	RoleStrategist   AgentRole = "estratega"
	RoleCritic       AgentRole = "critico"
	RoleAccelerator  AgentRole = "acelerador"
	RoleDesigner     AgentRole = "disenador"
	RoleWatcher      AgentRole = "vigia"
	RoleSynthesizer  AgentRole = "sintetizador"
)

func RoleStyle(role AgentRole) lipgloss.Style {
	switch role {
	case RoleStrategist:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#AA66FF")).Bold(true)
	case RoleCritic:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)
	case RoleAccelerator:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF41")).Bold(true)
	case RoleDesigner:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFAA")).Bold(true)
	case RoleWatcher:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800")).Bold(true)
	case RoleSynthesizer:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB000")).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
}

func RoleShortName(role AgentRole) string {
	switch role {
	case RoleStrategist:
		return "EST"
	case RoleCritic:
		return "CRI"
	case RoleAccelerator:
		return "ACE"
	case RoleDesigner:
		return "DIS"
	case RoleWatcher:
		return "VIG"
	case RoleSynthesizer:
		return "SIN"
	}
	return "???"
}

// VoteState represents how an agent is voting on the current topic
type VoteState string

const (
	VoteAgree        VoteState = "ok"
	VoteDisagree     VoteState = "no"
	VoteConfused     VoteState = "?"
	VoteReformulate  VoteState = "~"
	VoteWait         VoteState = "_"
	VoteCustom       VoteState = "*"
)

// AgentContext tracks an agent's context level and state
type AgentContext struct {
	AgentName  string
	Role       AgentRole
	Vote       VoteState
	ContextPct int // 0-100
	StatusText string
	Color      string
	LastVote   time.Time
}

// ChatMessage is a single message in the debate
type ChatMessage struct {
	Sender    string // "user" or agent name
	Role      AgentRole
	Text      string
	Timestamp time.Time
	IsSystem  bool
}

// Question represents one of the 5 diagnostic questions an agent can ask
type Question struct {
	ID       int
	Text     string
	Options  []string
	Answer   string // selected answer
}

// AgentDiagnostic tracks when user clicks a confused agent
type AgentDiagnostic struct {
	AgentName  string
	Questions  []Question
	Answers    int
	Complete   bool
	Report     string // final mini-report
}

// DebateSession holds the full debate state
type DebateSession struct {
	Active     bool
	Topic      string
	Messages   []ChatMessage
	AgentCtx   []AgentContext
	Diagnostic *AgentDiagnostic
	StartedAt  time.Time
	FocusAgent string // currently selected agent in list
}

// ConsensusState tracks real-time voting
type ConsensusState struct {
	TotalAgents   int
	VotedAgents   int
	AgreeCount    int
	DisagreeCount int
	ConfusedCount int
	WaitCount     int
	AvgContextPct int
}

func NewDebateSession(topic string) *DebateSession {
	return &DebateSession{
		Active:    false,
		Topic:     topic,
		Messages:  []ChatMessage{},
		AgentCtx:  []AgentContext{},
		StartedAt: time.Now(),
	}
}

func (d *DebateSession) CalcConsensus() ConsensusState {
	cs := ConsensusState{
		TotalAgents: len(d.AgentCtx),
	}
	totalCtx := 0
	for _, ac := range d.AgentCtx {
		cs.VotedAgents++
		totalCtx += ac.ContextPct
		switch ac.Vote {
		case VoteAgree:
			cs.AgreeCount++
		case VoteDisagree:
			cs.DisagreeCount++
		case VoteConfused:
			cs.ConfusedCount++
		case VoteWait:
			cs.WaitCount++
		}
	}
	if cs.VotedAgents > 0 {
		cs.AvgContextPct = totalCtx / cs.VotedAgents
	}
	return cs
}

// AddSystemMsg adds a system message to the debate
func (d *DebateSession) AddSystemMsg(text string) {
	d.Messages = append(d.Messages, ChatMessage{
		Sender:    "system",
		Text:      text,
		Timestamp: time.Now(),
		IsSystem:  true,
	})
}

// AddUserMsg adds a user message
func (d *DebateSession) AddUserMsg(text string) {
	d.Messages = append(d.Messages, ChatMessage{
		Sender:    "user",
		Text:      text,
		Timestamp: time.Now(),
		IsSystem:  false,
	})
}

// AddAgentMsg adds an agent message
func (d *DebateSession) AddAgentMsg(agentName string, role AgentRole, text string) {
	d.Messages = append(d.Messages, ChatMessage{
		Sender:    agentName,
		Role:      role,
		Text:      text,
		Timestamp: time.Now(),
		IsSystem:  false,
	})
}

// UpdateVote updates an agent's vote and context
func (d *DebateSession) UpdateVote(agentName string, vote VoteState, ctxPct int, statusText string) {
	for i, ac := range d.AgentCtx {
		if ac.AgentName == agentName {
			d.AgentCtx[i].Vote = vote
			d.AgentCtx[i].ContextPct = ctxPct
			d.AgentCtx[i].StatusText = statusText
			d.AgentCtx[i].LastVote = time.Now()
			return
		}
	}
}

func (d *DebateSession) ConsensusPct() int {
	cs := d.CalcConsensus()
	if cs.TotalAgents == 0 {
		return 0
	}
	return (cs.AgreeCount * 100) / cs.TotalAgents
}

// ---------- 5 Diagnostic Questions ----------

var DefaultQuestions = []Question{
	{
		ID:   1,
		Text: "¿Qué archivo del proyecto necesito revisar para entender el contexto?",
		Options: []string{
			"No necesita archivos, ya tengo suficiente",
			"Un archivo específico (específica en el chat)",
			"Varios archivos del proyecto",
			"Documentación externa (API docs, etc.)",
		},
	},
	{
		ID:   2,
		Text: "¿Hay alguna decisión ya tomada que deba considerar?",
		Options: []string{
			"No, el tema está abierto",
			"Sí, hay una decisión parcial",
			"Sí, ya está decidido, solo falta implementar",
			"No estoy seguro",
		},
	},
	{
		ID:   3,
		Text: "¿Cuál es el objetivo principal de este debate?",
		Options: []string{
			"Elegir entre alternativas técnicas",
			"Diseñar una solución desde cero",
			"Revisar y mejorar algo existente",
			"Debuggear o solucionar un bug",
		},
	},
	{
		ID:   4,
		Text: "¿Hay restricciones técnicas que deba conocer?",
		Options: []string{
			"Lenguaje/framework específico",
			"Deadline ajustado",
			"Presupuesto/recursos limitados",
			"Sin restricciones importantes",
		},
	},
	{
		ID:   5,
		Text: "¿Hay algo que ya funcione bien y no quieras cambiar?",
		Options: []string{
			"No, todo es nuevo",
			"Sí, hay partes que no se tocan",
			"Prefiero que sugieras y luego filtramos",
			"Ya tengo una solución parcial funcionando",
		},
	},
}

func NewAgentDiagnostic(agentName string) *AgentDiagnostic {
	qs := make([]Question, len(DefaultQuestions))
	copy(qs, DefaultQuestions)
	return &AgentDiagnostic{
		AgentName: agentName,
		Questions: qs,
		Answers:   0,
		Complete:  false,
	}
}

func (d *AgentDiagnostic) AnswerQuestion(qID int, answer string) {
	for i := range d.Questions {
		if d.Questions[i].ID == qID {
			d.Questions[i].Answer = answer
			d.Answers++
			break
		}
	}
	if d.Answers >= 5 {
		d.Complete = true
		d.generateReport()
	}
}

func (d *AgentDiagnostic) generateReport() {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Informe de contexto — %s\n", d.AgentName))
	sb.WriteString(fmt.Sprintf("Preguntas respondidas: %d/5\n", d.Answers))
	sb.WriteString("\nResumen:\n")
	for _, q := range d.Questions {
		if q.Answer != "" {
			sb.WriteString(fmt.Sprintf("- %s\n", q.Text))
			sb.WriteString(fmt.Sprintf("  → %s\n", q.Answer))
		}
	}
	sb.WriteString("\nNuevo contexto: 85%% — Puedo continuar el debate.")
	d.Report = sb.String()
}

// RenderChat renders the last N messages for the TUI
func (d *DebateSession) RenderChat(maxLines int) string {
	if len(d.Messages) == 0 {
		return " Debate vacío. Presiona 'D' para iniciar un debate."
	}

	var lines []string
	start := 0
	if len(d.Messages) > maxLines {
		start = len(d.Messages) - maxLines
	}

	for _, msg := range d.Messages[start:] {
		if msg.IsSystem {
			lines = append(lines, subtleStyle.Render(fmt.Sprintf(" ═ %s ═", msg.Text)))
			continue
		}
		if msg.Sender == "user" {
			lines = append(lines, chatUserMsg.Render(fmt.Sprintf(" Tú > %s", msg.Text)))
		} else {
			roleTag := fmt.Sprintf("[%s]", RoleShortName(msg.Role))
			lines = append(lines, fmt.Sprintf(" %s %s > %s",
				roleStyleFromRole(msg.Role).Render(roleTag),
				chatUsername.Render(msg.Sender),
				msg.Text,
			))
		}
	}
	return strings.Join(lines, "\n")
}

func roleStyleFromRole(role AgentRole) lipgloss.Style {
	switch role {
	case RoleStrategist:
		return chatAgentEstratega
	case RoleCritic:
		return chatAgentCritico
	case RoleAccelerator:
		return chatAgentAcelerador
	case RoleDesigner:
		return chatAgentDisenador
	case RoleWatcher:
		return chatAgentVigia
	case RoleSynthesizer:
		return chatAgentSinte
	}
	return subtleStyle
}

// RenderAgentList renders the list with vote states
func RenderAgentContextList(agents []AgentContext, focusAgent string) string {
	var b strings.Builder
	b.WriteString(agentListTitleStyle.Render(" Agentes"))
	b.WriteString("\n\n")

	if len(agents) == 0 {
		b.WriteString(subtleStyle.Render(" Sin agentes"))
		b.WriteString("\n\n")
		b.WriteString(subtleStyle.Render(" Ctrl+K: API Keys"))
		return b.String()
	}

	for _, ac := range agents {
		prefix := " "
		if ac.AgentName == focusAgent {
			prefix = "▸"
		}

		// Vote indicator
		voteStr := voteSymbol(ac.Vote)
		voteColor := voteStyle(ac.Vote)

		// Context bar
		ctxBar := renderContextBar(ac.ContextPct)

		line := fmt.Sprintf("%s %s %s %s",
			prefix,
			voteColor.Render(voteStr),
			ac.AgentName,
			ctxBar,
		)
		if ac.AgentName == focusAgent {
			line = agentSelectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")

		// Status text
		if ac.StatusText != "" {
			b.WriteString(subtleStyle.Render(fmt.Sprintf("   %s\n", ac.StatusText)))
		}
	}

	return b.String()
}

func voteSymbol(v VoteState) string {
	switch v {
	case VoteAgree:
		return "●"
	case VoteDisagree:
		return "○"
	case VoteConfused:
		return "?"
	case VoteReformulate:
		return "~"
	case VoteWait:
		return "·"
	case VoteCustom:
		return "*"
	}
	return "?"
}

func voteStyle(v VoteState) lipgloss.Style {
	switch v {
	case VoteAgree:
		return agentRunningStyle
	case VoteDisagree:
		return agentErrorStyle
	case VoteConfused:
		return agentErrorStyle
	case VoteReformulate:
		return agentSelectedStyle
	default:
		return agentStoppedStyle
	}
}

func renderContextBar(pct int) string {
	filled := pct / 10
	empty := 10 - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf(" %s %d%%", bar, pct)
}

// RenderConsensusMeter renders the bottom consensus bar
func RenderConsensusMeter(cs ConsensusState) string {
	pct := 0
	if cs.TotalAgents > 0 {
		pct = (cs.AgreeCount * 100) / cs.TotalAgents
	}

	barWidth := 30
	filled := (pct * barWidth) / 100
	empty := barWidth - filled

	bar := consensusFillStyle.Render(strings.Repeat("█", filled)) +
		consensusEmptyStyle.Render(strings.Repeat("░", empty))

	voteSymbols := fmt.Sprintf("●%d ○%d ?%d ·%d",
		cs.AgreeCount, cs.DisagreeCount,
		cs.ConfusedCount, cs.WaitCount)

	avgCtx := cs.AvgContextPct
	ctxLabel := fmt.Sprintf("🧠 %d%%", avgCtx)

	return fmt.Sprintf(" %s  %s  %s",
		bar,
		voteSymbols,
		ctxLabel,
	)
}

// RenderDonut renders a simple ASCII donut chart
func RenderDonut(cs ConsensusState) string {
	if cs.TotalAgents == 0 {
		return " 🍩 No data"
	}
	agreePct := (cs.AgreeCount * 100) / cs.TotalAgents
	disagreePct := (cs.DisagreeCount * 100) / cs.TotalAgents
	confusedPct := (cs.ConfusedCount * 100) / cs.TotalAgents

	return fmt.Sprintf(" 🍩 ●%d%% ○%d%% ?%d%%",
		agreePct, disagreePct, confusedPct)
}

// RenderQuestionPanel renders the diagnostic questions
func RenderQuestionPanel(diag *AgentDiagnostic) string {
	if diag == nil {
		return ""
	}

	if diag.Complete {
		return reportStyle.Render(diag.Report)
	}

	var b strings.Builder
	b.WriteString(questionTextStyle.Render(fmt.Sprintf(" %s necesita más información", diag.AgentName)))
	b.WriteString("\n\n")

	for _, q := range diag.Questions {
		if q.Answer == "" {
			b.WriteString(fmt.Sprintf(" %d. %s\n", q.ID, q.Text))
			for j, opt := range q.Options {
				b.WriteString(fmt.Sprintf("    %d) %s\n", j+1, opt))
			}
			break
		}
	}

	return questionPanelStyle.Render(b.String())
}

// Ensure agents get context updates from helpers
func UpdateAllAgentContext(agents []*agent.Agent, session *DebateSession) {
	for _, a := range agents {
		found := false
		for i, ac := range session.AgentCtx {
			if ac.AgentName == a.Name {
				status := a.CheckStatus()
				if status == agent.StatusRunning && ac.Vote == VoteWait {
					session.AgentCtx[i].Vote = VoteAgree
					session.AgentCtx[i].ContextPct = min(100, ac.ContextPct+5)
				}
				found = true
				break
			}
		}
		if !found {
			session.AgentCtx = append(session.AgentCtx, AgentContext{
				AgentName:  a.Name,
				Role:       inferRole(a.Name),
				Vote:       VoteWait,
				ContextPct: 50,
				StatusText: "Esperando...",
				LastVote:   time.Now(),
			})
		}
	}
}

func inferRole(name string) AgentRole {
	roles := map[string]AgentRole{
		"claude":   RoleStrategist,
		"opencode": RoleAccelerator,
		"codex":    RoleCritic,
		"gemini":   RoleStrategist,
		"amp":      RoleDesigner,
		"droid":    RoleWatcher,
	}
	if r, ok := roles[name]; ok {
		return r
	}
	return RoleSynthesizer
}
