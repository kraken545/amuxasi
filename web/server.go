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

	"github.com/amuxasi/amuxasi/debate"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	port       string
	mux        *http.ServeMux
	workspace  string
	debate     *debate.DebateSession
	debateMu   sync.Mutex
}

func NewServer(port int, workspacePath string) *Server {
	s := &Server{
		port:      fmt.Sprintf(":%d", port),
		mux:       http.NewServeMux(),
		workspace: workspacePath,
		debate:    debate.NewDebateSession(""),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// API
	s.mux.HandleFunc("/api/health", cors(s.handleHealth))
	s.mux.HandleFunc("/api/status", cors(s.handleStatus))
	s.mux.HandleFunc("/api/workspace", cors(s.handleWorkspace))
	s.mux.HandleFunc("/api/agents", cors(s.handleAgents))
	s.mux.HandleFunc("/api/agents/", cors(s.handleAgentAction))
	s.mux.HandleFunc("/api/debate", cors(s.handleDebate))
	s.mux.HandleFunc("/api/debate/message", cors(s.handleDebateMessage))
	s.mux.HandleFunc("/api/debate/vote", cors(s.handleDebateVote))
	s.mux.HandleFunc("/api/debate/diagnostic", cors(s.handleDebateDiagnostic))
	s.mux.HandleFunc("/api/keys", cors(s.handleKeys))
	s.mux.HandleFunc("/api/config", cors(s.handleConfig))
	s.mux.HandleFunc("/api/logs", cors(s.handleLogs))

	// Static files (SPA)
	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) Start() error {
	fmt.Printf("🌐 Amuxasi Web UI → http://localhost%s\n", s.port)
	fmt.Printf("   Workspace: %s\n", s.workspace)
	fmt.Println("   Presiona Ctrl+C para detener")
	return http.ListenAndServe(s.port, s.mux)
}

// StartServer is a convenience function to create and start a server from main.go
func StartServer(port int, workspacePath string) error {
	s := NewServer(port, workspacePath)
	return s.Start()
}

// ---- Static File Handler ----

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try embedded files first
	data, err := fs.ReadFile(staticFiles, "static/"+path)
	if err == nil {
		contentType := detectContentType(path)
		w.Header().Set("Content-Type", contentType)
		w.Write(data)
		return
	}

	// Fallback: try local filesystem (dev mode)
	localPath := filepath.Join("web", "static", path)
	if data, err := os.ReadFile(localPath); err == nil {
		contentType := detectContentType(path)
		w.Header().Set("Content-Type", contentType)
		w.Write(data)
		return
	}

	// SPA fallback: serve index.html
	indexData, err := fs.ReadFile(staticFiles, "static/index.html")
	if err != nil {
		localIndex := filepath.Join("web", "static", "index.html")
		if indexData, err = os.ReadFile(localIndex); err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexData)
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

// ---- CORS ----

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
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
