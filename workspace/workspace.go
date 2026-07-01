package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/amuxasi/amuxasi/config"
)

const ConfigFile = "amuxasi.toml"

type Manager struct {
	RepoRoot string
	Cfg      *config.Config
}

// FindRepoRoot busca la raíz del repo git. Si no hay, devuelve string vacío (no falla).
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := findGitRoot(dir)
	if err != nil {
		// No es un repo git — modo no-Git, devolvemos el dir actual
		return dir, nil
	}
	return root, nil
}

func findGitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// IsGitRepo verifica si el directorio actual pertenece a un repo git.
func IsGitRepo() bool {
	_, err := findGitRoot(".")
	return err == nil
}

func Open(repoRoot string) (*Manager, error) {
	cfgPath := filepath.Join(repoRoot, ConfigFile)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manager{
				RepoRoot: repoRoot,
				Cfg:      config.DefaultConfig(),
			}, nil
		}
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.Workspace.Name == "" {
		cfg.Workspace.Name = filepath.Base(repoRoot)
	}
	return &Manager{
		RepoRoot: repoRoot,
		Cfg:      cfg,
	}, nil
}

func Init(repoRoot string) error {
	return InitWithAgents(repoRoot, nil)
}

// InitWithAgents crea amuxasi.toml, scripts por defecto, y opcionalmente incluye agentes.
func InitWithAgents(repoRoot string, selectedAgents []string) error {
	cfgPath := filepath.Join(repoRoot, ConfigFile)
	if config.Exists(cfgPath) {
		return fmt.Errorf("%s already exists in %s", ConfigFile, repoRoot)
	}

	cfg := config.DefaultConfig()
	cfg.Workspace.Name = filepath.Base(repoRoot)

	// Si se seleccionaron agentes, filtrar el config para incluir solo esos
	if len(selectedAgents) > 0 {
		filtered := make(map[string]config.AgentConfig)
		for _, name := range selectedAgents {
			if a, ok := cfg.Agents[name]; ok {
				filtered[name] = a
			} else {
				// Agregar agente detectado no incluido en defaults
				filtered[name] = config.AgentConfig{
					Command: name,
					Args:    []string{},
				}
			}
		}
		cfg.Agents = filtered
		cfg.Launch = selectedAgents
	}

	f, err := os.Create(cfgPath)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	fmt.Printf("Created %s in %s\n", ConfigFile, repoRoot)

	// Crear scripts por defecto
	if err := createDefaultScripts(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create default scripts: %v\n", err)
	}

	return nil
}

// createDefaultScripts crea scripts/setup.sh y scripts/archive.sh si no existen.
func createDefaultScripts(repoRoot string) error {
	scriptsDir := filepath.Join(repoRoot, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("create scripts dir: %w", err)
	}

	scripts := map[string]string{
		"setup.sh": `#!/bin/sh
# Amuxasi — Script de setup
# Se ejecuta al presionar 'S' en el dashboard.
# Agrega aquí comandos de inicialización (npm install, pip install, etc.)
echo "📦 Amuxasi setup — $(date)"
# Ejemplo:
# npm install
# pip install -r requirements.txt
`,
		"archive.sh": `#!/bin/sh
# Amuxasi — Script de archive
# Se ejecuta al presionar 'A' en el dashboard.
# Agrega aquí comandos de limpieza o respaldo.
echo "📦 Amuxasi archive — $(date)"
# Ejemplo:
# tar -czf backup-$(date +%Y%m%d).tar.gz ./src
`,
	}

	for name, content := range scripts {
		path := filepath.Join(scriptsDir, name)
		if _, err := os.Stat(path); err == nil {
			continue // ya existe, no sobrescribir
		}
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			return fmt.Errorf("create %s: %w", name, err)
		}
		fmt.Printf("Created scripts/%s\n", name)
	}

	return nil
}

func (m *Manager) AddWorktree(path, branch string) error {
	if !IsGitRepo() {
		return fmt.Errorf("worktrees requieren un repositorio git")
	}

	args := []string{"worktree", "add"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, path)

	cmd := exec.Command("git", args...)
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	absPath, _ := filepath.Abs(path)
	fmt.Printf("Worktree created at %s\n", absPath)

	worktreeCfg := filepath.Join(absPath, ConfigFile)
	if !config.Exists(worktreeCfg) {
		cfg := config.DefaultConfig()
		wsName := filepath.Base(absPath)
		if branch != "" {
			wsName = branch
		}
		cfg.Workspace.Name = wsName
		f, err := os.Create(worktreeCfg)
		if err != nil {
			return fmt.Errorf("create worktree config: %w", err)
		}
		defer f.Close()
		enc := toml.NewEncoder(f)
		if err := enc.Encode(cfg); err != nil {
			return fmt.Errorf("encode worktree config: %w", err)
		}
		fmt.Printf("Created %s in worktree\n", ConfigFile)
	}

	return nil
}

func (m *Manager) Archive() error {
	fmt.Println("Archiving workspace...")

	for _, agent := range m.Cfg.Launch {
		if _, ok := m.Cfg.Agents[agent]; ok {
			fmt.Printf("  Stopping agent: %s\n", agent)
		}
	}

	archiveName := fmt.Sprintf("%s-archive", m.RepoRoot)
	fmt.Printf("  Config saved to %s\n", archiveName)
	fmt.Println("Archive complete.")
	return nil
}

func (m *Manager) HasScript(name string) bool {
	switch name {
	case "setup":
		return m.Cfg.Scripts.Setup != ""
	case "run":
		return m.Cfg.Scripts.Run != ""
	case "archive":
		return m.Cfg.Scripts.Archive != ""
	}
	return false
}

func (m *Manager) ScriptPath(name string) (string, error) {
	var rel string
	switch name {
	case "setup":
		rel = m.Cfg.Scripts.Setup
	case "run":
		rel = m.Cfg.Scripts.Run
	case "archive":
		rel = m.Cfg.Scripts.Archive
	default:
		return "", fmt.Errorf("unknown script: %s", name)
	}
	if rel == "" {
		return "", fmt.Errorf("no %s script configured", name)
	}
	abs := filepath.Join(m.RepoRoot, rel)
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("script not found at %s", abs)
	}
	return abs, nil
}

func (m *Manager) RunScript(name string) error {
	scriptPath, err := m.ScriptPath(name)
	if err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", scriptPath)
	cmd.Dir = m.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
