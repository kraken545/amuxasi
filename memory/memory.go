// Package memory proporciona un cliente para ChromaDB,
// una base de datos vectorial para memoria persistente de debates,
// decisiones de agentes, y búsqueda semántica en logs.
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Collection representa una colección en ChromaDB.
type Collection struct {
	Name      string            `json:"name"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Embedding  string            `json:"-"`
}

// MemoryItem representa un item almacenado con su embedding.
type MemoryItem struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
	Distance  float64           `json:"distance,omitempty"`
}

// Store maneja la conexión con ChromaDB.
type Store struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewStore crea un nuevo Store de ChromaDB.
// La URL se lee de CHROMADB_URL (por defecto http://localhost:8100).
func NewStore() *Store {
	baseURL := os.Getenv("CHROMADB_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8100"
	}
	return &Store{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Timeout:    10 * time.Second,
	}
}

// Health verifica que ChromaDB esté disponible.
func (s *Store) Health(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+"/api/v1/heartbeat", nil)
	if err != nil {
		return false
	}
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// EnsureCollection crea una colección si no existe.
func (s *Store) EnsureCollection(ctx context.Context, name string) error {
	url := fmt.Sprintf("%s/api/v1/collections", s.BaseURL)

	// Verificar si ya existe
	checkReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, url+"/"+name, nil)
	checkResp, err := s.HTTPClient.Do(checkReq)
	if err == nil && checkResp.StatusCode == http.StatusOK {
		checkResp.Body.Close()
		return nil // ya existe
	}
	if checkResp != nil {
		checkResp.Body.Close()
	}

	// Crear colección
	body := map[string]interface{}{
		"name": name,
		"metadata": map[string]string{
			"created_by": "amuxasi",
			"created_at": time.Now().Format(time.RFC3339),
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create collection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection: %s: %s", resp.Status, string(respBody))
	}
	return nil
}

// Add agrega un item a la memoria de una colección.
// ChromaDB maneja el embedding internamente si se configura un modelo.
func (s *Store) Add(ctx context.Context, collection, id, content string, metadata map[string]string) error {
	if err := s.EnsureCollection(ctx, collection); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/add", s.BaseURL, collection)
	body := map[string]interface{}{
		"ids":      []string{id},
		"documents": []string{content},
		"metadatas": []map[string]string{metadata},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("add request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("add to memory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add to memory: %s: %s", resp.Status, string(respBody))
	}
	return nil
}

// Query busca items similares en la memoria por contenido.
func (s *Store) Query(ctx context.Context, collection, query string, limit int) ([]MemoryItem, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}

	if err := s.EnsureCollection(ctx, collection); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", s.BaseURL, collection)
	body := map[string]interface{}{
		"query_texts": []string{query},
		"n_results":   limit,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, nil // ChromaDB no disponible — modo silencioso
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var result struct {
		IDs        [][]string           `json:"ids"`
		Documents  [][]string           `json:"documents"`
		Metadatas  [][]map[string]string `json:"metadatas"`
		Distances  [][]float64          `json:"distances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode query result: %w", err)
	}

	var items []MemoryItem
	for i := range result.IDs {
		for j := range result.IDs[i] {
			item := MemoryItem{
				ID:       result.IDs[i][j],
				Content:  result.Documents[i][j],
				Distance: result.Distances[i][j],
				Metadata: result.Metadatas[i][j],
			}
			if item.Metadata == nil {
				item.Metadata = make(map[string]string)
			}
			items = append(items, item)
		}
	}
	return items, nil
}

// SaveDecision guarda una decisión de debate en memoria persistente.
func (s *Store) SaveDecision(ctx context.Context, topic, decision string, consensusPct int, agents []string) error {
	metadata := map[string]string{
		"type":         "debate_decision",
		"topic":        topic,
		"consensus":    fmt.Sprintf("%d%%", consensusPct),
		"agents":       strings.Join(agents, ","),
		"timestamp":    time.Now().Format(time.RFC3339),
	}
	id := fmt.Sprintf("decision-%d", time.Now().UnixNano())
	return s.Add(ctx, "decisions", id, decision, metadata)
}

// GetRecentDecisions recupera decisiones recientes del debate.
func (s *Store) GetRecentDecisions(ctx context.Context, limit int) ([]MemoryItem, error) {
	return s.Query(ctx, "decisions", "recent debate decisions", limit)
}
