// Package compare permite ejecutar el mismo prompt contra múltiples
// agentes simultáneamente y ver los resultados lado a lado.
// Ideal para evaluar qué modelo/sistema responde mejor a cada tipo
// de pregunta o tarea.
package compare

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// RunnerStatus representa el estado de un runner individual.
type RunnerStatus string

const (
	StatusPending   RunnerStatus = "pending"
	StatusRunning   RunnerStatus = "running"
	StatusComplete  RunnerStatus = "complete"
	StatusError     RunnerStatus = "error"
	StatusTimeout   RunnerStatus = "timeout"
)

// RunnerResult es el resultado de un agente para el prompt.
type RunnerResult struct {
	AgentName string       `json:"agent_name"`
	Command   string       `json:"command"`
	Status    RunnerStatus `json:"status"`
	Output    string       `json:"output,omitempty"`
	Error     string       `json:"error,omitempty"`
	Duration  string       `json:"duration"`
	StartedAt time.Time    `json:"started_at"`
	runID     int
}

// CompareSession representa una ejecución de comparación.
type CompareSession struct {
	ID        string                 `json:"id"`
	Label     string                 `json:"label"`
	Prompt    string                 `json:"prompt"`
	Results   []*RunnerResult        `json:"results"`
	Status    string                 `json:"status"` // "running" | "complete"
	CreatedAt time.Time              `json:"created_at"`
	mu        sync.Mutex
}

// NewSession crea una nueva sesión de comparación.
func NewSession(id, label, prompt string) *CompareSession {
	return &CompareSession{
		ID:        id,
		Label:     label,
		Prompt:    prompt,
		Results:   []*RunnerResult{},
		Status:    "running",
		CreatedAt: time.Now(),
	}
}

// AddRunner agrega un runner (agente) a la comparación.
func (cs *CompareSession) AddRunner(name, command string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.Results = append(cs.Results, &RunnerResult{
		AgentName: name,
		Command:   command,
		Status:    StatusPending,
		StartedAt: time.Now(),
		runID:     len(cs.Results),
	})
}

// RunAll ejecuta el prompt contra todos los runners simultáneamente.
// Cada runner se ejecuta en su propia goroutine.
func (cs *CompareSession) RunAll(ctx context.Context, timeout time.Duration) {
	var wg sync.WaitGroup

	for _, r := range cs.Results {
		wg.Add(1)
		go func(rr *RunnerResult) {
			defer wg.Done()
			cs.runOne(ctx, rr, timeout)
		}(r)
	}

	wg.Wait()

	cs.mu.Lock()
	cs.Status = "complete"
	cs.mu.Unlock()
}

// runOne ejecuta el prompt contra un runner individual.
func (cs *CompareSession) runOne(ctx context.Context, rr *RunnerResult, timeout time.Duration) {
	rr.mu.Lock()
	rr.Status = StatusRunning
	start := time.Now()
	rr.mu.Unlock()

	// Crear contexto con timeout
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Construir comando: echo "<prompt>" | <command>
	// O bien: <command> "<prompt>"
	cmdStr := rr.Command
	hasStdin := false

	// Detectar si el comando acepta stdin
	switch {
	case strings.Contains(cmdStr, "claude"):
		// claude < prompt (lee de stdin)
		hasStdin = true
	case strings.Contains(cmdStr, "opencode"):
		// opencode accepta prompt como argumento
		cmdStr = fmt.Sprintf("%s -p %s", cmdStr, shellEscape(cs.Prompt))
	case strings.Contains(cmdStr, "gemini"):
		hasStdin = true
	case strings.Contains(cmdStr, "codex"):
		cmdStr = fmt.Sprintf("%s '%s'", cmdStr, strings.ReplaceAll(cs.Prompt, "'", "'\\''"))
	default:
		// Por defecto: pasar como argumento quoteado
		cmdStr = fmt.Sprintf("%s '%s'", cmdStr, strings.ReplaceAll(cs.Prompt, "'", "'\\''"))
	}

	var cmd *exec.Cmd
	if hasStdin {
		cmd = exec.CommandContext(runCtx, "sh", "-c", fmt.Sprintf("echo %s | %s", shellEscape(cs.Prompt), rr.Command))
	} else {
		cmd = exec.CommandContext(runCtx, "sh", "-c", cmdStr)
	}

	output, err := cmd.CombinedOutput()

	elapsed := time.Since(start)

	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.Duration = fmt.Sprintf("%.1fs", elapsed.Seconds())

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			rr.Status = StatusTimeout
			rr.Error = fmt.Sprintf("timeout after %v", timeout)
		} else {
			rr.Status = StatusError
			rr.Error = err.Error()
			rr.Output = string(output)
		}
		return
	}

	rr.Status = StatusComplete
	rr.Output = string(output)

	// Truncar output largo
	if len(rr.Output) > 50000 {
		rr.Output = rr.Output[:50000] + "\n... (truncated)"
	}
}

// GetResults devuelve los resultados ordenados por ID de runner.
func (cs *CompareSession) GetResults() []*RunnerResult {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	out := make([]*RunnerResult, len(cs.Results))
	copy(out, cs.Results)
	return out
}

// State devuelve el estado serializable de la sesión.
func (cs *CompareSession) State() map[string]interface{} {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	results := make([]map[string]interface{}, len(cs.Results))
	for i, r := range cs.Results {
		results[i] = map[string]interface{}{
			"agent_name": r.AgentName,
			"command":    r.Command,
			"status":     string(r.Status),
			"output":     r.Output,
			"error":      r.Error,
			"duration":   r.Duration,
		}
	}

	return map[string]interface{}{
		"id":        cs.ID,
		"label":     cs.Label,
		"prompt":    cs.Prompt,
		"results":   results,
		"status":    cs.Status,
		"created_at": cs.CreatedAt,
	}
}

// shellEscape escapa un string para usarlo seguro en shell.
func shellEscape(s string) string {
	// Reemplazar comillas simples y caracteres peligrosos
	s = strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", s)
}
