package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amuxasi/amuxasi/debate"
	"github.com/amuxasi/amuxasi/memory"
	"github.com/amuxasi/amuxasi/notify"
	"github.com/amuxasi/amuxasi/search"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	port       string
	mux        *http.ServeMux
	workspace  string
	debate     *debate.DebateSession
	debateMu   sync.Mutex
	authToken  string // empty = no auth
	rateLimit  *RateLimiter

	// Integraciones
	searchClient *search.Client
	memoryStore  *memory.Store
	notifyClient *notify.Client
}

// RateLimiter simple para prevenir abusos.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]int // IP -> count
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]int),
	}
}

// Allow verifica si una IP puede hacer una request.
// Retorna true si está dentro del límite.
func (rl *RateLimiter) Allow(ip string, maxPerMinute int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests[ip]++
	return rl.requests[ip] <= maxPerMinute
}

// ResetRateLimit se llama periódicamente para limpiar contadores.
func (s *Server) resetRateLimits() {
	s.rateLimit.mu.Lock()
	defer s.rateLimit.mu.Unlock()
	// Limpiar IPs con pocas requests
	for ip, count := range s.rateLimit.requests {
		if count < 10 {
			delete(s.rateLimit.requests, ip)
		} else {
			s.rateLimit.requests[ip] = count / 2
		}
	}
}

func NewServer(port int, workspacePath string) *Server {
	token := os.Getenv("AMUXASI_TOKEN")
	s := &Server{
		port:       fmt.Sprintf(":%d", port),
		mux:        http.NewServeMux(),
		workspace:  workspacePath,
		debate:     debate.NewDebateSession(""),
		authToken:  token,
		rateLimit:  NewRateLimiter(),

		// Integraciones (modo silencioso si los servicios no están disponibles)
		searchClient: search.NewClient(),
		memoryStore:  memory.NewStore(),
		notifyClient: notify.NewClient(),
	}
	s.routes()
	logInit()
	return s
}

func logInit() {
	fmt.Println("   Integraciones: search (SearXNG), memory (ChromaDB), notify (ntfy)")
	if os.Getenv("SEARXNG_URL") != "" {
		fmt.Printf("   → SearXNG: %s\n", os.Getenv("SEARXNG_URL"))
	}
	if os.Getenv("CHROMADB_URL") != "" {
		fmt.Printf("   → ChromaDB: %s\n", os.Getenv("CHROMADB_URL"))
	}
	if os.Getenv("NTFY_URL") != "" {
		fmt.Printf("   → ntfy: %s\n", os.Getenv("NTFY_URL"))
	}
}

func (s *Server) routes() {
	// API — protegidas con rate limit + auth + CORS
	s.mux.HandleFunc("/api/health", cors(s.requireAuth(s.rateLimitMiddleware(s.handleHealth))))
	s.mux.HandleFunc("/api/status", cors(s.requireAuth(s.rateLimitMiddleware(s.handleStatus))))
	s.mux.HandleFunc("/api/workspace", cors(s.requireAuth(s.rateLimitMiddleware(s.handleWorkspace))))
	s.mux.HandleFunc("/api/agents", cors(s.requireAuth(s.rateLimitMiddleware(s.handleAgents))))
	s.mux.HandleFunc("/api/agents/", cors(s.requireAuth(s.rateLimitMiddleware(s.handleAgentAction))))
	s.mux.HandleFunc("/api/debate", cors(s.requireAuth(s.rateLimitMiddleware(s.handleDebate))))
	s.mux.HandleFunc("/api/debate/message", cors(s.requireAuth(s.rateLimitMiddleware(s.handleDebateMessage))))
	s.mux.HandleFunc("/api/debate/vote", cors(s.requireAuth(s.rateLimitMiddleware(s.handleDebateVote))))
	s.mux.HandleFunc("/api/debate/diagnostic", cors(s.requireAuth(s.rateLimitMiddleware(s.handleDebateDiagnostic))))
	s.mux.HandleFunc("/api/keys", cors(s.requireAuth(s.rateLimitMiddleware(s.handleKeys))))
	s.mux.HandleFunc("/api/config", cors(s.requireAuth(s.rateLimitMiddleware(s.handleConfig))))
	s.mux.HandleFunc("/api/logs", cors(s.requireAuth(s.rateLimitMiddleware(s.handleLogs))))

	// Integraciones
	s.mux.HandleFunc("/api/search", cors(s.requireAuth(s.rateLimitMiddleware(s.handleSearch))))
	s.mux.HandleFunc("/api/memory", cors(s.requireAuth(s.rateLimitMiddleware(s.handleMemory))))
	s.mux.HandleFunc("/api/memory/decisions", cors(s.requireAuth(s.rateLimitMiddleware(s.handleMemoryDecisions))))
	s.mux.HandleFunc("/api/notify/test", cors(s.requireAuth(s.rateLimitMiddleware(s.handleNotifyTest))))

	// Static files (SPA) — sin auth para que funcione el frontend
	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) Start() error {
	fmt.Printf("🌐 Amuxasi Web UI → http://localhost%s\n", s.port)
	fmt.Printf("   Workspace: %s\n", s.workspace)
	fmt.Printf("   Auth: %s\n", map[bool]string{true: "enabled (AMUXASI_TOKEN)", false: "disabled"}[s.authToken != ""])
	fmt.Println("   Presiona Ctrl+C para detener")

	// Start rate limit reset ticker
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.resetRateLimits()
		}
	}()

	return http.ListenAndServe(s.port, s.mux)
}

// StartServer is a convenience function to create and start a server from main.go
func StartServer(port int, workspacePath string) error {
	s := NewServer(port, workspacePath)
	return s.Start()
}

// ---- Static File Handler ----

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Sanitize path: prevent directory traversal
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Clean the path to prevent traversal
	clean := filepath.Clean(path)
	// Prevent escaping the static directory
	if strings.HasPrefix(clean, "..") || strings.HasPrefix(clean, "/") || strings.Contains(clean, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	// Prevent absolute paths
	if filepath.IsAbs(clean) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	path = clean

	// Try embedded files first
	data, err := fs.ReadFile(staticFiles, "static/"+path)
	if err == nil {
		contentType := detectContentType(path)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(data)
		return
	}

	// Fallback: try local filesystem (dev mode) — ONLY if path is safe
	localCandidate := filepath.Join("web", "static", path)
	absCandidate, _ := filepath.Abs(localCandidate)
	absStatic, _ := filepath.Abs("web/static")
	if strings.HasPrefix(absCandidate, absStatic) {
		if data, err := os.ReadFile(localCandidate); err == nil {
			contentType := detectContentType(path)
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Write(data)
			return
		}
	}

	// SPA fallback: serve index.html with auth config injected
	indexData, err := fs.ReadFile(staticFiles, "static/index.html")
	if err != nil {
		localIndex := filepath.Join("web", "static", "index.html")
		if indexData, err = os.ReadFile(localIndex); err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	}
	// Inject auth config into the SPA
	authRequired := "false"
	if s.authToken != "" {
		authRequired = "true"
	}
	injected := strings.Replace(string(indexData),
		"</head>",
		fmt.Sprintf("<meta name=\"amuxasi-auth\" content=\"%s\"></head>", authRequired),
		1,
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write([]byte(injected))
}

func detectContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	default:
		return "text/plain; charset=utf-8"
	}
}

// ---- Rate Limiting ----

func (s *Server) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extraer IP del cliente
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}
		// Limitar: 60 requests/minuto por IP para APIs
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if !s.rateLimit.Allow(ip, 60) {
				w.Header().Set("Retry-After", "60")
				jsonErr(w, 429, "rate limit exceeded — 60 requests per minute max")
				return
			}
		}
		next(w, r)
	}
}

// ---- Auth ----

// requireAuth es un middleware que verifica el token de autenticación.
// Si AMUXASI_TOKEN no está configurado, permite acceso sin autenticación
// (comportamiento por defecto para instalaciones locales/Docker).
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authToken != "" {
			provided := r.Header.Get("Authorization")
			// Accept "Bearer <token>" or "<token>" directly
			if strings.HasPrefix(provided, "Bearer ") {
				provided = strings.TrimPrefix(provided, "Bearer ")
			}
			if provided != s.authToken {
				if provided == "" {
					// First request — return 401 with a hint (only for non-API paths)
					if strings.HasPrefix(r.URL.Path, "/api/") {
						w.Header().Set("WWW-Authenticate", `Bearer realm="amuxasi"`)
						jsonErr(w, 401, "authentication required — set AMUXASI_TOKEN or provide Authorization header")
						return
					}
				}
				jsonErr(w, 401, "invalid token")
				return
			}
		}
		next(w, r)
	}
}

// ---- CORS ----

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// En producción con auth, no mezclar CORS abierto
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		// Para requests normales, solo agregar CORS si no hay auth
		// (con auth, el frontend se sirve del mismo origen)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next(w, r)
	}
}

func jsonResp(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
