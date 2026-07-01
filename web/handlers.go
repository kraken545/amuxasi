package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/amuxasi/amuxasi/agent"
	"github.com/amuxasi/amuxasi/config"
	"github.com/amuxasi/amuxasi/debate"
	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/workspace"
)

// ---- Health ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok", "version": "0.2.0"})
}

// ---- Status ----

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ws, err := workspace.Open(s.workspace)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("workspace: %v", err))
		return
	}

	detected := agent.DetectAgents()
	agents := s.collectAgentStatus(ws)

	jsonResp(w, map[string]interface{}{
		"workspace":  ws.Cfg.Workspace.Name,
		"agents":     agents,
		"detected":   detected,
		"hasConfig":  config.Exists(filepath.Join(s.workspace, workspace.ConfigFile)),
		"hasTmux":    agent.TmuxExists(),
	})
}

// ---- Workspace ----

func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, err := workspace.Open(s.workspace)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("workspace: %v", err))
		return
	}
	jsonResp(w, ws.Cfg)
}

// ---- Agents ----

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	ws, err := workspace.Open(s.workspace)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("workspace: %v", err))
		return
	}

	switch r.Method {
	case "GET":
		agents := s.collectAgentStatus(ws)
		jsonResp(w, agents)
	default:
		jsonErr(w, 405, "method not allowed")
	}
}

func (s *Server) collectAgentStatus(ws *workspace.Manager) []map[string]interface{} {
	var result []map[string]interface{}

	// From config
	for name, cfg := range ws.Cfg.Agents {
		a := agent.New(name, cfg.Command, ws.Cfg.Workspace.Name, cfg.Args, cfg.Env)
		st := a.CheckStatus()
		result = append(result, map[string]interface{}{
			"name":    name,
			"command": cfg.Command,
			"args":    cfg.Args,
			"status":  st.String(),
			"running": st == agent.StatusRunning,
			"session": a.Session,
			"source":  "config",
		})
	}

	// Detected but not in config
	detected := agent.DetectAgents()
	for _, d := range detected {
		found := false
		for _, r := range result {
			if r["name"] == d.Name {
				found = true
				break
			}
		}
		if !found {
			a := agent.New(d.Name, d.Command, ws.Cfg.Workspace.Name, nil, nil)
			st := a.CheckStatus()
			result = append(result, map[string]interface{}{
				"name":    d.Name,
				"command": d.Command,
				"args":    []string{},
				"status":  st.String(),
				"running": st == agent.StatusRunning,
				"session": a.Session,
				"source":  "detected",
			})
		}
	}

	return result
}

// ---- Agent Actions ----

func (s *Server) handleAgentAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/")
	if len(parts) < 1 {
		jsonErr(w, 400, "agent name required")
		return
	}
	agentName := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	ws, err := workspace.Open(s.workspace)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("workspace: %v", err))
		return
	}

	// Find agent config
	cfg, ok := ws.Cfg.Agents[agentName]
	if !ok {
		// Check if it's a detected agent
		detected := agent.DetectAgents()
		found := false
		for _, d := range detected {
			if d.Name == agentName {
				cfg = config.AgentConfig{Command: d.Command}
				found = true
				break
			}
		}
		if !found {
			jsonErr(w, 404, fmt.Sprintf("agent %s not found", agentName))
			return
		}
	}

	a := agent.New(agentName, cfg.Command, ws.Cfg.Workspace.Name, cfg.Args, cfg.Env)

	switch action {
	case "launch":
		if err := a.Launch(); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		// Notificar
		go s.notifyAgentEvent(agentName, "launch")
		jsonResp(w, map[string]string{"status": "launched", "name": agentName, "session": a.Session})

	case "stop":
		if err := a.Stop(); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		// Notificar
		go s.notifyAgentEvent(agentName, "stop")
		jsonResp(w, map[string]string{"status": "stopped", "name": agentName})

	case "output":
		output := a.RefreshOutput()
		if output == "" {
			output = a.GetOutput()
		}
		jsonResp(w, map[string]string{"name": agentName, "output": output})

	case "status":
		st := a.CheckStatus()
		jsonResp(w, map[string]interface{}{
			"name":    agentName,
			"status":  st.String(),
			"running": st == agent.StatusRunning,
			"session": a.Session,
		})

	default:
		jsonErr(w, 400, fmt.Sprintf("unknown action: %s", action))
	}
}

// ---- Debate ----

func (s *Server) handleDebate(w http.ResponseWriter, r *http.Request) {
	s.debateMu.Lock()
	defer s.debateMu.Unlock()

	switch r.Method {
	case "GET":
		// GET /api/debate — devuelve estado completo
		jsonResp(w, s.debate.SessionState())

	case "POST":
		// POST /api/debate — iniciar/detener debate
		r.Body = http.MaxBytesReader(w, r.Body, 4096) // 4KB

		var body struct {
			Action string `json:"action"` // "start" | "stop"
			Topic  string `json:"topic,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "invalid JSON")
			return
		}

		body.Action = strings.TrimSpace(body.Action)
		body.Topic = strings.TrimSpace(body.Topic)

		switch body.Action {
		case "start":
			if s.debate.Active {
				jsonErr(w, 400, "Debate already active")
				return
			}
			if body.Topic == "" {
				body.Topic = "General discussion"
			}
			if len(body.Topic) > 500 {
				jsonErr(w, 400, "topic too long")
				return
			}
			s.debate = debate.NewDebateSession(body.Topic)
			// Add agents from config
			ws, err := workspace.Open(s.workspace)
			if err == nil {
				for name := range ws.Cfg.Agents {
					s.debate.AddOrUpdateAgent(name)
				}
			}
			s.debate.Start()
			// Cargar decisiones previas relacionadas
			go s.loadRelatedDecisions()
			jsonResp(w, map[string]interface{}{
				"status": "started",
				"topic":  body.Topic,
				"state":  s.debate.SessionState(),
			})

		case "stop":
			if !s.debate.Active {
				jsonErr(w, 400, "No active debate")
				return
			}
			// Guardar estado final en memoria antes de detener
			s.saveDebateToMemory()
			s.debate.Stop()
			jsonResp(w, map[string]interface{}{
				"status": "stopped",
				"state":  s.debate.SessionState(),
			})

		default:
			jsonErr(w, 400, fmt.Sprintf("unknown action: %s", body.Action))
		}

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

const (
	maxMessageLen  = 10000 // máx caracteres por mensaje
	maxMessages    = 500   // máx mensajes en un debate
	maxAgentNameLen = 100  // máx caracteres para nombre de agente
	maxStatusLen   = 200   // máx caracteres para status text
)

func (s *Server) handleDebateMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, 405, "POST required")
		return
	}

	// Limit body size
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64KB max

	s.debateMu.Lock()
	defer s.debateMu.Unlock()

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}

	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		jsonErr(w, 400, "message is required")
		return
	}
	if len(body.Message) > maxMessageLen {
		jsonErr(w, 400, "message too long")
		return
	}
	if len(s.debate.Messages) >= maxMessages {
		jsonErr(w, 400, "debate message limit reached")
		return
	}

	// Add user message
	s.debate.AddUserMsg(body.Message)

	// If debate is active, simulate agent responses
	if s.debate.Active {
		for _, ac := range s.debate.AgentCtx {
			response := fmt.Sprintf(
				"Respondiendo a: \"%s\" desde %s (%s)...",
				truncate(body.Message, 50),
				ac.AgentName,
				ac.Role.DisplayName(),
			)
			s.debate.AddAgentMsg(ac.AgentName, ac.Role, response)
			s.debate.UpdateVote(ac.AgentName, randomVote(), randomCtx(ac.ContextPct), "Analizando...")
		}

		// Auto-search: si hay agentes con contexto < 70%, buscar información
		go s.autoSearchForDebate(s.debate.Topic)

		// Guardar decisión si hay consenso >= 80%
		s.saveDebateToMemory()

		// Notificar si hay consenso
		go s.notifyConsensus()
	}

	jsonResp(w, map[string]interface{}{
		"status": "ok",
		"state":  s.debate.SessionState(),
	})
}

// ---- Vote ----

func (s *Server) handleDebateVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, 405, "POST required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096) // 4KB max

	s.debateMu.Lock()
	defer s.debateMu.Unlock()

	var body struct {
		AgentName string `json:"agent_name"`
		Vote      string `json:"vote"` // "agree" | "disagree" | "confused"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}

	body.AgentName = strings.TrimSpace(body.AgentName)
	if body.AgentName == "" || len(body.AgentName) > maxAgentNameLen {
		jsonErr(w, 400, "invalid agent_name")
		return
	}

	vote := debate.VoteState(body.Vote)
	switch vote {
	case debate.VoteAgree, debate.VoteDisagree, debate.VoteConfused:
		// valid
	default:
		jsonErr(w, 400, fmt.Sprintf("invalid vote: %s", body.Vote))
		return
	}

	s.debate.UpdateVote(body.AgentName, vote, randomCtx(50), fmt.Sprintf("Votó: %s", vote.Symbol()))

	// Verificar consenso después del voto
	s.saveDebateToMemory()
	go s.notifyConsensus()

	jsonResp(w, map[string]interface{}{
		"status": "voted",
		"state":  s.debate.SessionState(),
	})
}

// ---- Diagnostics ----

func (s *Server) handleDebateDiagnostic(w http.ResponseWriter, r *http.Request) {
	s.debateMu.Lock()
	defer s.debateMu.Unlock()

	switch r.Method {
	case "GET":
		if s.debate.Diagnostic == nil {
			jsonResp(w, map[string]string{"status": "no_active_diagnostic"})
			return
		}
		jsonResp(w, s.debate.Diagnostic)

	case "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 4096) // 4KB

		var body struct {
			AgentName string `json:"agent_name"`
			Action    string `json:"action"` // "start" | "answer"
			QID       int    `json:"q_id,omitempty"`
			Answer    string `json:"answer,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "invalid JSON")
			return
		}

		body.AgentName = strings.TrimSpace(body.AgentName)
		if body.AgentName == "" || len(body.AgentName) > maxAgentNameLen {
			jsonErr(w, 400, "invalid agent_name")
			return
		}
		body.Answer = strings.TrimSpace(body.Answer)

		switch body.Action {
		case "start":
			s.debate.Diagnostic = debate.NewAgentDiagnostic(body.AgentName)
			jsonResp(w, s.debate.Diagnostic)

		case "answer":
			if s.debate.Diagnostic == nil {
				jsonErr(w, 400, "no active diagnostic")
				return
			}
			if len(body.Answer) > 500 {
				jsonErr(w, 400, "answer too long")
				return
			}
			if body.QID < 1 || body.QID > 5 {
				jsonErr(w, 400, "invalid question id (1-5)")
				return
			}
			s.debate.Diagnostic.AnswerQuestion(body.QID, body.Answer)
			if s.debate.Diagnostic.Complete {
				s.debate.UpdateVote(body.AgentName, debate.VoteAgree, 85, "Diagnóstico completado")
			}
			jsonResp(w, s.debate.Diagnostic)

		default:
			jsonErr(w, 400, fmt.Sprintf("unknown action: %s", body.Action))
		}

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

// Helpers

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func randomVote() debate.VoteState {
	votes := []debate.VoteState{
		debate.VoteAgree,
		debate.VoteAgree,
		debate.VoteAgree,
		debate.VoteDisagree,
		debate.VoteConfused,
	}
	return votes[os.Getpid()%len(votes)]
}

func randomCtx(base int) int {
	return min(100, base+(os.Getpid()%20-10))
}

// ---- API Keys ----

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	// Solo devolver qué keys están configuradas, sin exponer prefijos
	envKeys := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
		"OPENROUTER_API_KEY",
		"MISTRAL_API_KEY",
		"GROQ_API_KEY",
		"TOGETHER_API_KEY",
	}

	var keys []map[string]interface{}
	for _, k := range envKeys {
		val := os.Getenv(k)
		keys = append(keys, map[string]interface{}{
			"name": k,
			"set":  val != "",
		})
	}

	jsonResp(w, keys)
}

// ---- Config ----

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	ws, err := workspace.Open(s.workspace)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("workspace: %v", err))
		return
	}

	switch r.Method {
	case "GET":
		jsonResp(w, ws.Cfg)
	case "POST":
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			jsonErr(w, 400, "invalid JSON")
			return
		}
		jsonResp(w, map[string]string{"status": "config update coming soon"})
	default:
		jsonErr(w, 405, "method not allowed")
	}
}

// ---- Logs ----

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if log.Global() != nil {
		lines := log.Global().Lines()
		if len(lines) > 100 {
			lines = lines[len(lines)-100:]
		}
		jsonResp(w, map[string]interface{}{
			"lines": lines,
			"count": len(lines),
		})
		return
	}
	jsonResp(w, map[string]interface{}{"lines": []string{}, "count": 0})
}
