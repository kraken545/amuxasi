// Package search proporciona un cliente para SearXNG,
// un meta-buscador privado que agrega resultados de múltiples motores.
// Se integra con el sistema de debate para que los agentes puedan
// buscar información automáticamente cuando su contexto es bajo.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Result representa un resultado individual de búsqueda.
type Result struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Engine  string  `json:"engine"`
	Score   float64 `json:"score,omitempty"`
	Source  string  `json:"-"`
}

// SearchResponse es la respuesta completa de la API JSON de SearXNG.
type SearchResponse struct {
	Query       string   `json:"query"`
	NumberOfResults int  `json:"number_of_results"`
	Results     []Result `json:"results"`
	Answers     []string `json:"answers"`
	Infoboxes   []struct {
		InfoboxURL string `json:"infobox_url"`
		Engine     string `json:"engine"`
		Content    string `json:"content"`
		URLs       []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"urls"`
	} `json:"infoboxes"`
	Suggestions []string `json:"suggestions"`
	UnresponsiveEngines []string `json:"unresponsive_engines"`
}

// Client es el cliente HTTP para SearXNG.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewClient crea un nuevo cliente SearXNG.
// La URL base se lee de SEARXNG_URL (por defecto http://localhost:8081).
// Si no hay variable de entorno, se asume que SearXNG no está disponible
// y el cliente opera en modo silencioso (siempre devuelve resultados vacíos).
func NewClient() *Client {
	baseURL := os.Getenv("SEARXNG_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		Timeout:    15 * time.Second,
	}
}

// NewClientWithURL crea un cliente con URL explícita (útil para tests).
func NewClientWithURL(baseURL string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		Timeout:    15 * time.Second,
	}
}

// Search ejecuta una búsqueda en SearXNG y devuelve resultados estructurados.
// Si el servidor no responde, devuelve resultados vacíos (no error) para
// que el sistema siga funcionando sin SearXNG.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	u, _ := url.Parse(c.BaseURL + "/search")
	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	q.Set("language", "es")
	q.Set("categories", "general")
	q.Set("pageno", "1")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("search: crear request: %w", err)
	}
	req.Header.Set("User-Agent", "Amuxasi/0.1 (meta-buscador interno; privado; https://github.com/kraken545/amuxasi)")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		// Servidor no disponible — modo silencioso
		return nil, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("search: leer respuesta: %w", err)
	}

	var sr SearchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("search: decodificar JSON: %w", err)
	}

	// Limitar resultados y marcar fuente
	results := sr.Results
	if len(results) > limit {
		results = results[:limit]
	}
	for i := range results {
		results[i].Source = "searxng"
	}

	return results, nil
}

// SearchAndFormat busca y devuelve un string formateado para insertar
// directamente en el contexto de un agente.
func (c *Client) SearchAndFormat(ctx context.Context, query string, limit int) string {
	results, err := c.Search(ctx, query, limit)
	if err != nil || results == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📡 Búsqueda web: \"%s\"\n", query))
	b.WriteString(strings.Repeat("─", 40) + "\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		b.WriteString(fmt.Sprintf("   %s\n", r.URL))
		if r.Content != "" {
			// Truncar contenido largo
			content := r.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			b.WriteString(fmt.Sprintf("   %s\n", content))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// AutoSearch determina si se necesita una búsqueda basada en la
// confianza promedio del contexto de los agentes en un debate.
// Retorna las consultas sugeridas si hay agentes con contexto < 70%.
func AutoSearch(agentContexts map[string]float64) []string {
	if len(agentContexts) == 0 {
		return nil
	}

	lowCount := 0
	var lowAgents []string
	for agent, ctx := range agentContexts {
		if ctx < 0.7 {
			lowCount++
			lowAgents = append(lowAgents, agent)
		}
	}

	if lowCount == 0 {
		return nil
	}

	// Sugerir búsqueda para mejorar contexto
	suggestions := make([]string, 0, len(lowAgents))
	for _, agent := range lowAgents {
		suggestions = append(suggestions,
			fmt.Sprintf("Agente %s tiene contexto bajo (%.0f%%) — buscar información relevante",
				agent, agentContexts[agent]*100))
	}
	return suggestions
}
