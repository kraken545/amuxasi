// Package debate proporciona tipos y lógica compartida para el debate multi-agente.
// Usado tanto por la TUI (Bubble Tea) como por la Web UI (REST API).
package debate

import (
	"fmt"
	"strings"
	"time"
)

// AgentRole define el rol de un agente en el debate.
type AgentRole string

const (
	RoleStrategist  AgentRole = "estratega"
	RoleCritic      AgentRole = "critico"
	RoleAccelerator AgentRole = "acelerador"
	RoleDesigner    AgentRole = "disenador"
	RoleWatcher     AgentRole = "vigia"
	RoleSynthesizer AgentRole = "sintetizador"
)

// RoleDisplayName devuelve el nombre legible del rol.
func (r AgentRole) DisplayName() string {
	switch r {
	case RoleStrategist:
		return "Estratega"
	case RoleCritic:
		return "Crítico"
	case RoleAccelerator:
		return "Acelerador"
	case RoleDesigner:
		return "Diseñador"
	case RoleWatcher:
		return "Vigía"
	case RoleSynthesizer:
		return "Sintetizador"
	}
	return string(r)
}

// RoleShortName devuelve la abreviación de 3 letras del rol.
func (r AgentRole) ShortName() string {
	switch r {
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

// RoleColor devuelve el color hex para el rol (usado en Web UI y TUI).
func (r AgentRole) Color() string {
	switch r {
	case RoleStrategist:
		return "#AA66FF" // Púrpura
	case RoleCritic:
		return "#FF3333" // Rojo
	case RoleAccelerator:
		return "#00FF41" // Verde
	case RoleDesigner:
		return "#00FFAA" // Cian
	case RoleWatcher:
		return "#FF8800" // Naranja
	case RoleSynthesizer:
		return "#FFB000" // Ámbar
	}
	return "#CCCCCC"
}

// InferRole asigna un rol por defecto según el nombre del agente.
func InferRole(name string) AgentRole {
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

// VoteState representa el voto de un agente.
type VoteState string

const (
	VoteAgree       VoteState = "agree"
	VoteDisagree    VoteState = "disagree"
	VoteConfused    VoteState = "confused"
	VoteReformulate VoteState = "reformulate"
	VoteWait        VoteState = "wait"
)

// VoteSymbol devuelve el símbolo visual del voto.
func (v VoteState) Symbol() string {
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
	}
	return "·"
}

// AgentContext rastrea el estado de un agente en el debate.
type AgentContext struct {
	AgentName  string    `json:"agent_name"`
	Role       AgentRole `json:"role"`
	RoleLabel  string    `json:"role_label"`
	Vote       VoteState `json:"vote"`
	VoteSymbol string    `json:"vote_symbol"`
	ContextPct int       `json:"context_pct"` // 0-100
	StatusText string    `json:"status_text"`
	Color      string    `json:"color"`
	LastVote   time.Time `json:"last_vote"`
}

// NewAgentContext crea un nuevo contexto de agente con valores por defecto.
func NewAgentContext(name string) AgentContext {
	role := InferRole(name)
	return AgentContext{
		AgentName:  name,
		Role:       role,
		RoleLabel:  role.DisplayName(),
		Vote:       VoteWait,
		VoteSymbol: VoteWait.Symbol(),
		ContextPct: 50,
		StatusText: "Esperando...",
		Color:      role.Color(),
		LastVote:   time.Now(),
	}
}

// ChatMessage es un mensaje individual en el debate.
type ChatMessage struct {
	Sender    string    `json:"sender"`
	Role      AgentRole `json:"role,omitempty"`
	RoleLabel string    `json:"role_label,omitempty"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	IsSystem  bool      `json:"is_system"`
}

// NewSystemMsg crea un mensaje del sistema.
func NewSystemMsg(text string) ChatMessage {
	return ChatMessage{
		Sender:    "system",
		Text:      text,
		Timestamp: time.Now(),
		IsSystem:  true,
	}
}

// NewUserMsg crea un mensaje del usuario.
func NewUserMsg(text string) ChatMessage {
	return ChatMessage{
		Sender:    "user",
		Text:      text,
		Timestamp: time.Now(),
		IsSystem:  false,
	}
}

// NewAgentMsg crea un mensaje de un agente.
func NewAgentMsg(agentName string, role AgentRole, text string) ChatMessage {
	return ChatMessage{
		Sender:    agentName,
		Role:      role,
		RoleLabel: role.DisplayName(),
		Text:      text,
		Timestamp: time.Now(),
		IsSystem:  false,
	}
}

// Question representa una de las 5 preguntas de diagnóstico.
type Question struct {
	ID      int      `json:"id"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
	Answer  string   `json:"answer,omitempty"`
}

// DefaultQuestions son las 5 preguntas de diagnóstico por defecto.
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

// AgentDiagnostic rastrea el diagnóstico de 5 preguntas.
type AgentDiagnostic struct {
	AgentName string     `json:"agent_name"`
	Questions []Question `json:"questions"`
	Answers   int        `json:"answers"`
	Complete  bool       `json:"complete"`
	Report    string     `json:"report,omitempty"` // mini-informe final
}

// NewAgentDiagnostic crea un nuevo diagnóstico para un agente.
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

// AnswerQuestion registra la respuesta a una pregunta.
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
	sb.WriteString("\nNuevo contexto: 85% — Puedo continuar el debate.")
	d.Report = sb.String()
}

// ConsensusState rastrea el consenso en tiempo real.
type ConsensusState struct {
	TotalAgents   int `json:"total_agents"`
	VotedAgents   int `json:"voted_agents"`
	AgreeCount    int `json:"agree_count"`
	DisagreeCount int `json:"disagree_count"`
	ConfusedCount int `json:"confused_count"`
	WaitCount     int `json:"wait_count"`
	AvgContextPct int `json:"avg_context_pct"`
	ConsensusPct  int `json:"consensus_pct"`
}

// CalcConsensus calcula el estado de consenso a partir de los agentes.
func (cs *ConsensusState) CalcConsensus(agents []AgentContext) {
	cs.TotalAgents = len(agents)
	cs.VotedAgents = 0
	cs.AgreeCount = 0
	cs.DisagreeCount = 0
	cs.ConfusedCount = 0
	cs.WaitCount = 0
	totalCtx := 0

	for _, ac := range agents {
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
	if cs.TotalAgents > 0 {
		cs.ConsensusPct = (cs.AgreeCount * 100) / cs.TotalAgents
	}
}

// DebateSession mantiene el estado completo de un debate.
type DebateSession struct {
	Active     bool              `json:"active"`
	Topic      string            `json:"topic"`
	Messages   []ChatMessage     `json:"messages"`
	AgentCtx   []AgentContext    `json:"agent_context"`
	Consensus  ConsensusState    `json:"consensus"`
	Diagnostic *AgentDiagnostic  `json:"diagnostic,omitempty"`
	StartedAt  time.Time         `json:"started_at"`
}

// NewDebateSession crea una nueva sesión de debate.
func NewDebateSession(topic string) *DebateSession {
	return &DebateSession{
		Active:    false,
		Topic:     topic,
		Messages:  []ChatMessage{},
		AgentCtx:  []AgentContext{},
		StartedAt: time.Now(),
	}
}

// Start inicia el debate.
func (d *DebateSession) Start() {
	d.Active = true
	d.StartedAt = time.Now()
	d.AddSystemMsg(fmt.Sprintf("🧠 Debate iniciado: %s", d.Topic))
}

// Stop detiene el debate.
func (d *DebateSession) Stop() {
	d.Active = false
	d.AddSystemMsg("🛑 Debate detenido")
}

// AddSystemMsg agrega un mensaje del sistema.
func (d *DebateSession) AddSystemMsg(text string) {
	d.Messages = append(d.Messages, NewSystemMsg(text))
}

// AddUserMsg agrega un mensaje del usuario.
func (d *DebateSession) AddUserMsg(text string) {
	d.Messages = append(d.Messages, NewUserMsg(text))
}

// AddAgentMsg agrega un mensaje de un agente.
func (d *DebateSession) AddAgentMsg(agentName string, role AgentRole, text string) {
	d.Messages = append(d.Messages, NewAgentMsg(agentName, role, text))
}

// UpdateVote actualiza el voto y contexto de un agente.
func (d *DebateSession) UpdateVote(agentName string, vote VoteState, ctxPct int, statusText string) {
	for i, ac := range d.AgentCtx {
		if ac.AgentName == agentName {
			d.AgentCtx[i].Vote = vote
			d.AgentCtx[i].VoteSymbol = vote.Symbol()
			d.AgentCtx[i].ContextPct = ctxPct
			d.AgentCtx[i].StatusText = statusText
			d.AgentCtx[i].LastVote = time.Now()
			d.refreshConsensus()
			return
		}
	}
}

// AddOrUpdateAgent agrega un agente al debate o actualiza su estado.
func (d *DebateSession) AddOrUpdateAgent(name string) {
	for _, ac := range d.AgentCtx {
		if ac.AgentName == name {
			return // ya existe
		}
	}
	d.AgentCtx = append(d.AgentCtx, NewAgentContext(name))
	d.AddSystemMsg(fmt.Sprintf("🤖 %s se unió al debate", name))
	d.refreshConsensus()
}

// refreshConsensus recalcula el consenso.
func (d *DebateSession) refreshConsensus() {
	d.Consensus.CalcConsensus(d.AgentCtx)
}

// MessagesJSON devuelve los mensajes en formato JSON-friendly.
func (d *DebateSession) MessagesJSON() []map[string]interface{} {
	result := make([]map[string]interface{}, len(d.Messages))
	for i, msg := range d.Messages {
		m := map[string]interface{}{
			"sender":    msg.Sender,
			"text":      msg.Text,
			"timestamp": msg.Timestamp,
			"is_system": msg.IsSystem,
		}
		if msg.Role != "" {
			m["role"] = string(msg.Role)
			m["role_label"] = msg.Role.DisplayName()
		}
		result[i] = m
	}
	return result
}

// SessionState devuelve el estado completo del debate como mapa JSON-friendly.
func (d *DebateSession) SessionState() map[string]interface{} {
	return map[string]interface{}{
		"active":    d.Active,
		"topic":     d.Topic,
		"messages":  d.MessagesJSON(),
		"agents":    d.AgentCtx,
		"consensus": d.Consensus,
		"started":   d.StartedAt,
	}
}
