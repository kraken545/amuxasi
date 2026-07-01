// Package notify proporciona un cliente para ntfy.sh,
// un servicio de notificaciones push autohosteado que envía
// alertas al teléfono/escritorio cuando los agentes completan
// tareas, se alcanza consenso, o ocurren errores.
package notify

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Priority representa la prioridad de una notificación.
type Priority int

const (
	PriorityMin    Priority = 1
	PriorityLow    Priority = 2
	PriorityNormal Priority = 3
	PriorityHigh   Priority = 4
	PriorityMax    Priority = 5
)

// Client maneja el envío de notificaciones via ntfy.
type Client struct {
	BaseURL    string
	Topic      string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewClient crea un nuevo cliente ntfy.
// La URL se lee de NTFY_URL (por defecto http://localhost:8091).
// El topic por defecto es "amuxasi" pero se puede cambiar.
func NewClient() *Client {
	baseURL := os.Getenv("NTFY_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8091"
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Topic:      "amuxasi",
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Timeout:    5 * time.Second,
	}
}

// NewClientWithTopic crea un cliente con un topic personalizado.
func NewClientWithTopic(topic string) *Client {
	c := NewClient()
	c.Topic = topic
	return c
}

// Notify envía una notificación push con título y mensaje.
func (c *Client) Notify(ctx context.Context, title, message string, priority Priority) error {
	url := fmt.Sprintf("%s/%s", c.BaseURL, c.Topic)

	body := message
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("notify request: %w", err)
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", fmt.Sprintf("%d", priority))
	req.Header.Set("Tags", "amuxasi")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil // ntfy no disponible — modo silencioso
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("ntfy: %s", resp.Status)
	}
	return nil
}

// NotifyAgentReady notifica que un agente completó una tarea.
func (c *Client) NotifyAgentReady(ctx context.Context, agentName string) error {
	return c.Notify(ctx,
		fmt.Sprintf("✅ %s listo", agentName),
		fmt.Sprintf("El agente %s ha completado su tarea.", agentName),
		PriorityNormal,
	)
}

// NotifyConsensus notifica que se alcanzó consenso en un debate.
func (c *Client) NotifyConsensus(ctx context.Context, topic string, consensusPct int) error {
	return c.Notify(ctx,
		"🤝 Consenso alcanzado",
		fmt.Sprintf("Tema: %s\nConsenso: %d%%", topic, consensusPct),
		PriorityHigh,
	)
}

// NotifyError notifica un error crítico en el sistema.
func (c *Client) NotifyError(ctx context.Context, errMsg string) error {
	return c.Notify(ctx,
		"⚠️ Error en Amuxasi",
		errMsg,
		PriorityMax,
	)
}

// NotifyAgentStuck notifica que un agente está atascado (contexto bajo).
func (c *Client) NotifyAgentStuck(ctx context.Context, agentName string, contextPct int) error {
	return c.Notify(ctx,
		fmt.Sprintf("⚠️ %s necesita ayuda", agentName),
		fmt.Sprintf("El agente %s tiene contexto bajo (%d%%). Responde las preguntas de diagnóstico.", agentName, contextPct),
		PriorityLow,
	)
}
