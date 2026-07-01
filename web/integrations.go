package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amuxasi/amuxasi/agent"
	"github.com/amuxasi/amuxasi/compare"
	"github.com/amuxasi/amuxasi/debate"
	"github.com/amuxasi/amuxasi/workspace"
)

// ──────────────────────────────────────────────
//  COMPARE — Prompt lado a lado
// ──────────────────────────────────────────────

// handleCompare lista sesiones o inicia una nueva.
// GET  /api/compare — lista sesiones
// POST /api/compare — inicia nueva comparación
func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.compareMu.Lock()
		sessions := make([]map[string]interface{}, 0, len(s.compareSessions))
		for _, cs := range s.compareSessions {
			st := cs.State()
			st["prompt"] = truncate(st["prompt"].(string), 100)
			sessions = append(sessions, st)
		}
		s.compareMu.Unlock()
		jsonResp(w, sessions)

	case "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 65536) // 64KB
		var body struct {
			Label       string   `json:"label"`
			Prompt      string   `json:"prompt"`
			AgentNames  []string `json:"agent_names"`
			Timeout     int      `json:"timeout"` // seconds
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "invalid JSON")
			return
		}
		body.Prompt = strings.TrimSpace(body.Prompt)
		if body.Prompt == "" {
			jsonErr(w, 400, "prompt is required")
			return
		}
		if len(body.Prompt) > 10000 {
			jsonErr(w, 400, "prompt too long (max 10k)")
			return
		}
		if len(body.AgentNames) == 0 {
			jsonErr(w, 400, "at least one agent is required")
			return
		}

		timeout := time.Duration(body.Timeout) * time.Second
		if timeout <= 0 || timeout > 120*time.Second {
			timeout = 30 * time.Second
		}

		// Crear sesión
		s.compareMu.Lock()
		s.compareIDCounter++
		id := fmt.Sprintf("cmp-%d", s.compareIDCounter)
		label := body.Label
		if label == "" {
			label = truncate(body.Prompt, 50)
		}
		cs := compare.NewSession(id, label, body.Prompt)
		s.compareSessions[id] = cs
		s.compareMu.Unlock()

		// Resolver nombres de agentes a comandos
		ws, err := workspace.Open(s.workspace)
		if err == nil {
			for _, name := range body.AgentNames {
				cmd := ""
				if cfg, ok := ws.Cfg.Agents[name]; ok {
					cmd = cfg.Command
				} else {
					// Buscar en detectados
					for _, d := range agent.DetectAgents() {
						if d.Name == name {
							cmd = d.Command
							break
						}
					}
				}
				if cmd != "" {
					cs.AddRunner(name, cmd)
				}
			}
		}

		// Si no se resolvieron runners, usar detectados
		if len(cs.GetResults()) == 0 {
			for _, d := range agent.DetectAgents() {
				cs.AddRunner(d.Name, d.Command)
			}
		}

		// Ejecutar en background
		ctx := context.Background()
		go cs.RunAll(ctx, timeout)

		jsonResp(w, map[string]interface{}{
			"status":   "started",
			"id":       id,
			"sessions": cs.State(),
		})

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

// handleCompareByID devuelve el estado de una sesión específica.
// GET /api/compare/{id}
func (s *Server) handleCompareByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/compare/")
	if id == "" {
		jsonErr(w, 400, "compare session ID required")
		return
	}

	s.compareMu.Lock()
	cs, ok := s.compareSessions[id]
	s.compareMu.Unlock()

	if !ok {
		jsonErr(w, 404, fmt.Sprintf("compare session %s not found", id))
		return
	}

	jsonResp(w, cs.State())
}

// ──────────────────────────────────────────────
//  BÚSQUEDA WEB (SearXNG)
// ──────────────────────────────────────────────

// handleSearch permite búsqueda manual desde la Web UI.
// POST /api/search  body: {"query": "..."}
// GET  /api/search?q=...
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var query string

	switch r.Method {
	case "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "invalid JSON")
			return
		}
		query = strings.TrimSpace(body.Query)
	case "GET":
		query = strings.TrimSpace(r.URL.Query().Get("q"))
	default:
		jsonErr(w, 405, "method not allowed")
		return
	}

	if query == "" {
		jsonErr(w, 400, "query is required")
		return
	}
	if len(query) > 500 {
		jsonErr(w, 400, "query too long")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	results, err := s.searchClient.Search(ctx, query, 10)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("search error: %v", err))
		return
	}

	jsonResp(w, map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// autoSearchForDebate busca información si hay agentes con contexto bajo
// e inyecta los resultados en el debate como mensaje del sistema.
func (s *Server) autoSearchForDebate(topic string) {
	// Recolectar contextos de agentes
	ctxMap := make(map[string]float64)
	for _, ac := range s.debate.AgentCtx {
		ctxMap[ac.AgentName] = float64(ac.ContextPct) / 100.0
	}

	suggestions := search.AutoSearch(ctxMap)
	if len(suggestions) == 0 {
		return
	}

	// Buscar información relevante sobre el tema del debate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Usar el último mensaje del usuario como query
	lastUserMsg := ""
	for i := len(s.debate.Messages) - 1; i >= 0; i-- {
		if !s.debate.Messages[i].IsSystem && s.debate.Messages[i].Sender == "user" {
			lastUserMsg = s.debate.Messages[i].Text
			break
		}
	}

	query := topic
	if lastUserMsg != "" {
		query = lastUserMsg
	}

	results := s.searchClient.SearchAndFormat(ctx, query, 5)
	if results != "" {
		s.debate.AddSystemMsg(fmt.Sprintf("📡 Resultados de búsqueda para: %s\n%s", query, results))
		// Mejorar contexto de agentes con contexto bajo
		for i, ac := range s.debate.AgentCtx {
			if ac.ContextPct < 70 {
				s.debate.AgentCtx[i].ContextPct = min(85, ac.ContextPct+20)
				s.debate.AgentCtx[i].StatusText = "Contexto mejorado vía búsqueda"
			}
		}
		s.debate.AddSystemMsg("🔍 Contexto mejorado para agentes con información de búsqueda web")
	}
}

// ──────────────────────────────────────────────
//  MEMORIA PERSISTENTE (ChromaDB)
// ──────────────────────────────────────────────

// handleMemory permite almacenar y consultar memoria.
// POST /api/memory  body: {"content": "...", "metadata": {...}}
// GET  /api/memory?q=...&limit=5
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch r.Method {
	case "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 32768) // 32KB
		var body struct {
			Collection string            `json:"collection"`
			ID         string            `json:"id,omitempty"`
			Content    string            `json:"content"`
			Metadata   map[string]string `json:"metadata,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonErr(w, 400, "invalid JSON")
			return
		}
		body.Content = strings.TrimSpace(body.Content)
		if body.Content == "" {
			jsonErr(w, 400, "content is required")
			return
		}
		if body.Collection == "" {
			body.Collection = "general"
		}
		if body.ID == "" {
			body.ID = fmt.Sprintf("mem-%d", time.Now().UnixNano())
		}
		if body.Metadata == nil {
			body.Metadata = map[string]string{}
		}
		body.Metadata["source"] = "web_ui"
		body.Metadata["timestamp"] = time.Now().Format(time.RFC3339)

		if err := s.memoryStore.Add(ctx, body.Collection, body.ID, body.Content, body.Metadata); err != nil {
			jsonErr(w, 500, fmt.Sprintf("memory add error: %v", err))
			return
		}
		jsonResp(w, map[string]string{"status": "stored", "id": body.ID, "collection": body.Collection})

	case "GET":
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		collection := strings.TrimSpace(r.URL.Query().Get("collection"))
		if collection == "" {
			collection = "general"
		}
		if query == "" {
			jsonErr(w, 400, "query param 'q' is required")
			return
		}
		limit := 5
		items, err := s.memoryStore.Query(ctx, collection, query, limit)
		if err != nil {
			jsonErr(w, 500, fmt.Sprintf("memory query error: %v", err))
			return
		}
		jsonResp(w, map[string]interface{}{
			"query":      query,
			"collection": collection,
			"results":    items,
			"count":      len(items),
		})

	default:
		jsonErr(w, 405, "method not allowed")
	}
}

// handleMemoryDecisions devuelve decisiones recientes del debate.
func (s *Server) handleMemoryDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, 405, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	items, err := s.memoryStore.GetRecentDecisions(ctx, 10)
	if err != nil {
		jsonErr(w, 500, fmt.Sprintf("memory query error: %v", err))
		return
	}
	jsonResp(w, map[string]interface{}{
		"results": items,
		"count":   len(items),
	})
}

// saveDebateToMemory guarda el estado del debate como decisión cuando
// hay consenso >= 80%.
func (s *Server) saveDebateToMemory() {
	if !s.debate.Active || s.debate.Consensus.ConsensusPct < 80 {
		return
	}

	// Extraer el último mensaje relevante como "decisión"
	lastDecision := ""
	for i := len(s.debate.Messages) - 1; i >= 0; i-- {
		msg := s.debate.Messages[i]
		if !msg.IsSystem && msg.Sender != "user" {
			lastDecision = msg.Text
			break
		}
	}
	if lastDecision == "" {
		return
	}

	agentNames := make([]string, len(s.debate.AgentCtx))
	for i, ac := range s.debate.AgentCtx {
		agentNames[i] = ac.AgentName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.memoryStore.SaveDecision(ctx,
		s.debate.Topic,
		lastDecision,
		s.debate.Consensus.ConsensusPct,
		agentNames,
	); err == nil {
		s.debate.AddSystemMsg("🧠 Decisión guardada en memoria persistente")
	}
}

// ──────────────────────────────────────────────
//  NOTIFICACIONES PUSH (ntfy)
// ──────────────────────────────────────────────

// handleNotifyTest envía una notificación de prueba.
func (s *Server) handleNotifyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, 405, "POST required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.notifyClient.Notify(ctx,
		"🧪 Amuxasi — Prueba",
		"Si ves esto, las notificaciones push funcionan.",
		notify.PriorityNormal,
	); err != nil {
		jsonErr(w, 500, fmt.Sprintf("notify error: %v", err))
		return
	}

	jsonResp(w, map[string]string{"status": "notification sent"})
}

// notifyAgentEvent envía notificación según el evento del agente.
func (s *Server) notifyAgentEvent(agentName, event string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch event {
	case "launch":
		s.notifyClient.NotifyAgentReady(ctx, agentName)
	case "stop":
		s.notifyClient.Notify(ctx,
			fmt.Sprintf("⏹️ %s detenido", agentName),
			fmt.Sprintf("El agente %s fue detenido.", agentName),
			notify.PriorityLow,
		)
	case "error":
		s.notifyClient.NotifyError(ctx, fmt.Sprintf("Error en agente %s", agentName))
	}
}

// loadRelatedDecisions carga decisiones previas relacionadas con el tema
// del debate y las inyecta como contexto.
func (s *Server) loadRelatedDecisions() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	items, err := s.memoryStore.GetRecentDecisions(ctx, 5)
	if err != nil || len(items) == 0 {
		return
	}

	s.debateMu.Lock()
	defer s.debateMu.Unlock()

	s.debate.AddSystemMsg("📚 Decisiones previas cargadas de memoria:")
	for _, item := range items {
		content := item.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		s.debate.AddSystemMsg(fmt.Sprintf("  • %s", content))
	}
}

// notifyConsensus envía notificación cuando se alcanza consenso.
func (s *Server) notifyConsensus() {
	if !s.debate.Active || s.debate.Consensus.ConsensusPct < 80 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.notifyClient.NotifyConsensus(ctx, s.debate.Topic, s.debate.Consensus.ConsensusPct)
}
