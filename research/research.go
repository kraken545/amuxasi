// Package research implementa Deep Research: un proceso multi-ronda
// que busca información en la web, identifica subtemas, y sintetiza
// un informe final. Cada ronda profundiza más en los hallazgos de la anterior.
package research

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Status del research session.
type Status string

const (
	StatusPending   Status = "pending"
	StatusSearching Status = "searching"
	StatusAnalyzing Status = "analyzing"
	StatusComplete  Status = "complete"
	StatusError     Status = "error"
)

// Finding es un hallazgo individual de investigación.
type Finding struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Source  string `json:"source"`
	Round   int    `json:"round"`
	Query   string `json:"query"`
}

// ResearchRound es una ronda de investigación.
type ResearchRound struct {
	Number       int       `json:"number"`
	Query        string    `json:"query"`
	Findings     []Finding `json:"findings"`
	Subtopics    []string  `json:"subtopics"`
	Status       Status    `json:"status"`
}

// Report es el informe final sintetizado.
type Report struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Sections    []string `json:"sections"`
	Sources     []string `json:"sources"`
	WordCount   int      `json:"word_count"`
	GeneratedAt string   `json:"generated_at"`
}

// Searcher es la interfaz que debe implementar el buscador web.
// Así el paquete research no depende directamente de search.Client.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Result, error)
}

// Result es un resultado de búsqueda (definido localmente para no acoplar).
type Result struct {
	Title   string
	URL     string
	Content string
	Source  string
}

// ResearchSession mantiene el estado de una investigación completa.
type ResearchSession struct {
	ID        string          `json:"id"`
	Query     string          `json:"query"`
	Depth     int             `json:"depth"`
	MaxDepth  int             `json:"max_depth"`
	Status    Status          `json:"status"`
	Rounds    []ResearchRound `json:"rounds"`
	Findings  []Finding       `json:"findings"`
	Report    *Report         `json:"report,omitempty"`
	Progress  string          `json:"progress"`
	CreatedAt time.Time       `json:"created_at"`

	searcher Searcher
	mu       sync.Mutex
}

// NewSession crea una nueva sesión de investigación profunda.
func NewSession(id, query string, maxDepth int, searcher Searcher) *ResearchSession {
	if maxDepth < 1 {
		maxDepth = 3
	}
	if maxDepth > 5 {
		maxDepth = 5
	}
	return &ResearchSession{
		ID:        id,
		Query:     query,
		MaxDepth:  maxDepth,
		Status:    StatusPending,
		Rounds:    []ResearchRound{},
		Findings:  []Finding{},
		CreatedAt: time.Now(),
		searcher:  searcher,
	}
}

// Run ejecuta la investigación completa de forma bloqueante.
// Llamar en una goroutine.
func (rs *ResearchSession) Run(ctx context.Context) {
	rs.mu.Lock()
	rs.Status = StatusSearching
	rs.Progress = fmt.Sprintf("Ronda 1: Investigando \"%s\"...", rs.Query)
	rs.mu.Unlock()

	visited := make(map[string]bool) // URLs ya visitadas

	// Cola de investigación: empieza con la query inicial
	type queueItem struct {
		query string
		depth int
	}
	queue := []queueItem{{query: rs.Query, depth: 0}}

	for len(queue) > 0 && rs.Depth < rs.MaxDepth {
		item := queue[0]
		queue = queue[1:]

		select {
		case <-ctx.Done():
			rs.mu.Lock()
			rs.Status = StatusError
			rs.Progress = "Investigación cancelada"
			rs.mu.Unlock()
			return
		default:
		}

		rs.mu.Lock()
		rs.Depth = item.depth + 1
		rs.Progress = fmt.Sprintf("Ronda %d/%d: \"%s\"...", rs.Depth, rs.MaxDepth, item.query)
		rs.Status = StatusSearching
		rs.mu.Unlock()

		// Buscar
		results, err := rs.searcher.Search(ctx, item.query, 8)
		if err != nil || len(results) == 0 {
			continue
		}

		round := ResearchRound{
			Number:       rs.Depth,
			Query:        item.query,
			Findings:     []Finding{},
			Subtopics:    []string{},
			Status:       StatusSearching,
		}

		// Extraer findings y subtopics
		var subtopics []string
		for _, res := range results {
			if visited[res.URL] {
				continue
			}
			visited[res.URL] = true

			content := res.Content
			if content == "" {
				content = res.Title
			}

			finding := Finding{
				URL:     res.URL,
				Title:   res.Title,
				Content: content,
				Source:  res.Source,
				Round:   rs.Depth,
				Query:   item.query,
			}
			round.Findings = append(round.Findings, finding)
			rs.Findings = append(rs.Findings, finding)

			// Extraer subtopics del título y contenido
			subtopic := extractSubtopic(res.Title, res.Content)
			if subtopic != "" {
				subtopics = append(subtopics, subtopic)
			}
		}

		// Deducir subtopics únicos y relevantes
		round.Subtopics = dedupeSubtopics(subtopics)
		round.Status = StatusComplete

		rs.mu.Lock()
		rs.Rounds = append(rs.Rounds, round)
		rs.Progress = fmt.Sprintf("Ronda %d completada — %d hallazgos, %d subtemas",
			rs.Depth, len(round.Findings), len(round.Subtopics))
		rs.mu.Unlock()

		// Agregar subtopics a la cola (siguiente ronda)
		time.Sleep(500 * time.Millisecond) // pausa entre rondas
		for _, st := range round.Subtopics {
			if rs.Depth < rs.MaxDepth {
				queue = append(queue, queueItem{
					query: fmt.Sprintf("%s %s", rs.Query, st),
					depth: rs.Depth,
				})
			}
		}
	}

	// Sintetizar informe final
	rs.mu.Lock()
	rs.Status = StatusAnalyzing
	rs.Progress = "Sintetizando informe final..."
	rs.mu.Unlock()

	rs.synthesizeReport()

	rs.mu.Lock()
	rs.Status = StatusComplete
	rs.Progress = fmt.Sprintf("Investigación completa — %d rondas, %d hallazgos",
		len(rs.Rounds), len(rs.Findings))
	rs.mu.Unlock()
}

// synthesizeReport genera el informe final a partir de los hallazgos.
func (rs *ResearchSession) synthesizeReport() {
	title := fmt.Sprintf("Informe: %s", rs.Query)

	// Construir resumen
	var summaryParts []string
	summaryParts = append(summaryParts, fmt.Sprintf(
		"Investigación sobre \"%s\" (%d rondas, %d fuentes).",
		rs.Query, len(rs.Rounds), len(rs.Findings)))

	// Agrupar hallazgos por ronda
	sections := []string{}
	for _, round := range rs.Rounds {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("\n## Ronda %d: %s\n", round.Number, round.Query))
		for _, f := range round.Findings {
			sb.WriteString(fmt.Sprintf("\n### %s\n", f.Title))
			if f.Content != "" {
				content := f.Content
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				sb.WriteString(fmt.Sprintf("\n%s\n", content))
			}
			sb.WriteString(fmt.Sprintf("\nFuente: %s\n", f.URL))
		}
		sections = append(sections, sb.String())
	}

	// Recolectar fuentes únicas
	sourceSet := make(map[string]bool)
	var sources []string
	for _, f := range rs.Findings {
		if !sourceSet[f.URL] {
			sourceSet[f.URL] = true
			sources = append(sources, f.URL)
		}
	}

	// Contar palabras totales
	totalWords := 0
	for _, f := range rs.Findings {
		totalWords += len(strings.Fields(f.Content))
	}

	summaryParts = append(summaryParts, fmt.Sprintf(
		"Se recopilaron %d fuentes con aproximadamente %d palabras de contenido.",
		len(sources), totalWords))

	report := &Report{
		Title:       title,
		Summary:     strings.Join(summaryParts, " "),
		Sections:    sections,
		Sources:     sources,
		WordCount:   totalWords,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	rs.Report = report
}

// State devuelve el estado serializable de la sesión.
func (rs *ResearchSession) State() map[string]interface{} {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rounds := make([]map[string]interface{}, len(rs.Rounds))
	for i, r := range rs.Rounds {
		rounds[i] = map[string]interface{}{
			"number":    r.Number,
			"query":     r.Query,
			"findings":  r.Findings,
			"subtopics": r.Subtopics,
			"status":    string(r.Status),
		}
	}

	// Limitar findings a los últimos 50 para respuestas rápidas
	allFindings := rs.Findings
	if len(allFindings) > 50 {
		allFindings = allFindings[len(allFindings)-50:]
	}

	return map[string]interface{}{
		"id":         rs.ID,
		"query":      rs.Query,
		"depth":      rs.Depth,
		"max_depth":  rs.MaxDepth,
		"status":     string(rs.Status),
		"rounds":     rounds,
		"findings":   allFindings,
		"findings_count": len(rs.Findings),
		"report":     rs.Report,
		"progress":   rs.Progress,
		"created_at": rs.CreatedAt,
	}
}

// extractSubtopic extrae un posible subtema del título y contenido.
func extractSubtopic(title, content string) string {
	// Palabras clave que indican un subtema importante
	keywords := []string{
		"architecture", "implementation", "approach", "method",
		"comparison", "vs", "versus", "alternative", "example",
		"best practice", "guide", "tutorial", "introduction",
		"overview", "analysis", "survey", "review",
		"case study", "benchmark", "performance", "security",
		"limitation", "challenge", "future", "trend",
	}
	text := strings.ToLower(title + " " + content)
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			// Usar las primeras palabras del título como subtema
			words := strings.Fields(title)
			if len(words) > 5 {
				words = words[:5]
			}
			return strings.Join(words, " ")
		}
	}
	return ""
}

// dedupeSubtopics elimina subtopics duplicados y vacíos.
func dedupeSubtopics(subtopics []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range subtopics {
		s = strings.TrimSpace(s)
		s = strings.ToLower(s)
		if s == "" || seen[s] || len(s) < 10 {
			continue
		}
		seen[s] = true
		result = append(result, s)
	}
	// Limitar a 3 subtopics por ronda
	if len(result) > 3 {
		result = result[:3]
	}
	return result
}
