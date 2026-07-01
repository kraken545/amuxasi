package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/amuxasi/amuxasi/agent"
	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/web"
	"github.com/amuxasi/amuxasi/workspace"
)

func main() {
	// Init logging
	dataDir, err := os.UserCacheDir()
	if err != nil {
		dataDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	logDir := filepath.Join(dataDir, "amuxasi", "logs")
	os.MkdirAll(logDir, 0755)
	if err := log.Init(logDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: logger: %v\n", err)
	}
	defer log.Close()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cmdInit()
			return
		case "web":
			cmdWeb()
			return
		case "add-worktree":
			cmdAddWorktree()
			return
		case "archive":
			cmdArchive()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "version", "--version", "-v":
			fmt.Println("amuxasi v0.2.0")
			return
		}
	}

	// Default: launch TUI
	launchTUI()
}

// askYesNo pregunta al usuario y devuelve true si responde afirmativamente.
func askYesNo(question string) bool {
	fmt.Printf("%s (y/n): ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "si"
}

// hasTTY detecta si hay una terminal interactiva disponible.
func hasTTY() bool {
	// Intentar abrir /dev/tty
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	tty.Close()

	// Verificar que stdin es una terminal
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func launchTUI() {
	// ── Auto-init si no hay config ──
	repoRoot, _ := workspace.FindRepoRoot()
	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
	}

	cfgPath := filepath.Join(repoRoot, workspace.ConfigFile)
	if !configExists(cfgPath) && hasTTY() {
		if askYesNo("📄 No hay amuxasi.toml. ¿Inicializar configuración ahora?") {
			if err := workspace.Init(repoRoot); err != nil {
				fmt.Fprintf(os.Stderr, "Error al init: %v\n", err)
			}
		}
	}

	// ── Intentar ejecutar TUI ──
	tuiBin := "amuxasi-tui"
	if path, err := exec.LookPath(tuiBin); err == nil && hasTTY() {
		cmd := exec.Command(path, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// ── TUI no encontrada ──
	if hasTTY() {
		// Hay terminal: preguntar si compilar
		fmt.Println("🖥️  El dashboard TUI (amuxasi-tui) no está instalado.")
		if askYesNo("¿Compilarlo ahora (requiere Go)?") {
			if compileTUI() {
				// Reintentar ejecutar
				if path, err := exec.LookPath(tuiBin); err == nil {
					cmd := exec.Command(path, os.Args[1:]...)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Run()
				}
			}
		} else if askYesNo("¿Quieres abrir la Web UI en su lugar?") {
			startWebFallback(repoRoot)
		} else {
			printHelp()
		}
	} else {
		// No hay terminal: abrir Web UI automáticamente
		if askYesNo("🌐 No hay terminal disponible. ¿Iniciar Web UI?") {
			startWebFallback(repoRoot)
		} else {
			fmt.Println("Usa: amuxasi web  para iniciar la Web UI")
		}
	}
}

func compileTUI() bool {
	fmt.Println("⚙️  Compilando amuxasi-tui...")
	cmd := exec.Command("go", "build", "-o", "amuxasi-tui", "./cmd/amuxasi-tui/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error compilando TUI: %v\n", err)
		fmt.Println("Asegúrate de tener Go instalado: https://go.dev/dl/")
		return false
	}

	// Mover a GOPATH/bin o al directorio actual
	dest := "amuxasi-tui"
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		dest = filepath.Join(gopath, "bin", "amuxasi-tui")
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.Rename("amuxasi-tui", dest)
	} else if home, _ := os.UserHomeDir(); home != "" {
		dest = filepath.Join(home, "go", "bin", "amuxasi-tui")
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.Rename("amuxasi-tui", dest)
	}

	fmt.Printf("✅ TUI compilada: %s\n", dest)
	return true
}

func startWebFallback(workspacePath string) {
	port := 7000
	fmt.Printf("🌐 Iniciando Web UI en http://localhost:%d\n", port)

	if askYesNo("¿Abrir el navegador automáticamente?") {
		openBrowser(fmt.Sprintf("http://localhost:%d", port))
	}

	fmt.Println("Presiona Ctrl+C para detener")
	if err := web.StartServer(port, workspacePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func openBrowser(url string) {
	switch {
	case exec.Command("xdg-open", url).Run() == nil: // Linux
	case exec.Command("open", url).Run() == nil: // macOS
	case exec.Command("cmd", "/c", "start", url).Run() == nil: // Windows
	default:
		fmt.Printf("   Abre: %s\n", url)
	}
}

func configExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cmdInit() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Detectar agentes instalados
	detected := agent.DetectAgents()
	if len(detected) > 0 && hasTTY() {
		fmt.Println("🔍 Agentes detectados en tu sistema:")
		selected := []string{}
		for _, d := range detected {
			if askYesNo(fmt.Sprintf("  ¿Incluir '%s' (%s)?", d.Name, d.Path)) {
				selected = append(selected, d.Name)
			}
		}
		if len(selected) > 0 {
			fmt.Printf("✅ Se incluirán: %s\n", strings.Join(selected, ", "))
			workspace.InitWithAgents(dir, selected)
			return
		}
	}

	if err := workspace.Init(dir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdWeb() {
	port := 7000
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--port", "-p":
			if i+1 < len(os.Args) {
				if p, err := strconv.Atoi(os.Args[i+1]); err == nil {
					port = p
				}
				i++
			}
		}
	}

	// Find repo root (no falla si no es git, solo usa cwd)
	repoRoot, _ := workspace.FindRepoRoot()
	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
	}

	log.Info("starting web server on port %d, workspace: %s", port, repoRoot)
	fmt.Printf("🌐 Amuxasi Web UI → http://localhost:%d\n", port)
	fmt.Fprintf(os.Stderr, "   Workspace: %s\n", repoRoot)
	fmt.Fprintf(os.Stderr, "   Presiona Ctrl+C para detener\n")

	if err := web.StartServer(port, repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdAddWorktree() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: amuxasi add-worktree <path> [branch]\n")
		os.Exit(1)
	}
	repoRoot, err := workspace.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	ws, err := workspace.Open(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	path := os.Args[2]
	branch := ""
	if len(os.Args) > 3 {
		branch = os.Args[3]
	}
	if err := ws.AddWorktree(path, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdArchive() {
	repoRoot, err := workspace.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	ws, err := workspace.Open(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := ws.Archive(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Amuxasi — Multi-Agent Coding Dashboard

Usage:
  amuxasi                Open TUI dashboard
  amuxasi web            Start Web UI (http://localhost:7000)
  amuxasi init           Create amuxasi.toml in current directory
  amuxasi add-worktree <path> [branch]  Create git worktree + config
  amuxasi archive        Archive workspace
  amuxasi help           Show this help
  amuxasi version        Show version

Web UI:   http://localhost:7000 (run: amuxasi web)
Docker:   docker compose up (includes web UI)

Docs: https://github.com/kraken545/amuxasi`)
}
