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
		jsonResp(w, map[string]string{"status": "launched", "name": agentName, "session": a.Session})

	case "stop":
		if err := a.Stop(); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
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
	jsonResp(w, map[string]string{"status": "debate endpoint ready", "message": "Debate coming soon"})
}

func (s *Server) handleDebateMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, 405, "POST required")
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	jsonResp(w, map[string]string{"status": "received", "message": body.Message})
}

// ---- API Keys ----

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
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
			"name":   k,
			"set":    val != "",
			"prefix": safePrefix(val),
		})
	}

	jsonResp(w, keys)
}

func safePrefix(key string) string {
	if len(key) <= 8 {
		return ""
	}
	return key[:8] + "..."
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
