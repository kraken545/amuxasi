package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Status int

const (
	StatusStopped Status = iota
	StatusRunning
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusRunning:
		return "running"
	case StatusError:
		return "error"
	}
	return "unknown"
}

type Agent struct {
	Name       string
	Command    string
	Args       []string
	Env        map[string]string
	Status     Status
	Output     string
	Session    string
	statusMu   sync.RWMutex
	outputMu   sync.RWMutex
	detected   bool
	workspace  string
}

func New(name, command, workspace string, args []string, env map[string]string) *Agent {
	return &Agent{
		Name:      name,
		Command:   command,
		Args:      args,
		Env:       env,
		Status:    StatusStopped,
		Session:   SessionName(workspace, name),
		workspace: workspace,
	}
}

type DetectedAgent struct {
	Name    string
	Command string
	Path    string
}

func DetectAgents() []DetectedAgent {
	candidates := []struct {
		name    string
		command string
	}{
		{"claude", "claude"},
		{"opencode", "opencode"},
		{"codex", "codex"},
		{"gemini", "gemini"},
		{"amp", "amp"},
		{"droid", "droid"},
		{"aide", "aide"},
		{"copilot", "copilot"},
	}

	var detected []DetectedAgent
	for _, c := range candidates {
		path, err := exec.LookPath(c.command)
		if err == nil {
			detected = append(detected, DetectedAgent{
				Name:    c.name,
				Command: c.command,
				Path:    path,
			})
		}
	}
	return detected
}

func (a *Agent) Launch() error {
	if !TmuxExists() {
		return fmt.Errorf("tmux is required but not found in PATH")
	}

	a.statusMu.Lock()
	defer a.statusMu.Unlock()

	if a.Status == StatusRunning {
		return fmt.Errorf("agent %s is already running", a.Name)
	}

	env := os.Environ()
	for k, v := range a.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	cmdStr := a.Command
	if len(a.Args) > 0 {
		cmdStr += " " + strings.Join(a.Args, " ")
	}

	if err := TmuxNewSession(a.Session, cmdStr, env); err != nil {
		a.Status = StatusError
		return fmt.Errorf("launch agent %s: %w", a.Name, err)
	}

	a.Status = StatusRunning
	a.Output = ""
	return nil
}

func (a *Agent) Stop() error {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()

	if a.Status == StatusStopped {
		return nil
	}

	if err := TmuxKillSession(a.Session); err != nil {
		return err
	}

	a.Status = StatusStopped
	return nil
}

func (a *Agent) RefreshOutput() string {
	a.statusMu.RLock()
	session := a.Session
	status := a.Status
	a.statusMu.RUnlock()

	if status != StatusRunning {
		return ""
	}

	output, err := TmuxCapturePane(session)
	if err != nil {
		return ""
	}

	a.outputMu.Lock()
	defer a.outputMu.Unlock()
	a.Output = output
	return output
}

func (a *Agent) GetOutput() string {
	a.outputMu.RLock()
	defer a.outputMu.RUnlock()
	return a.Output
}

func (a *Agent) CheckStatus() Status {
	a.statusMu.RLock()
	session := a.Session
	a.statusMu.RUnlock()

	alive := TmuxHasSession(session)

	a.statusMu.Lock()
	defer a.statusMu.Unlock()

	if !alive && a.Status == StatusRunning {
		a.Status = StatusStopped
	} else if alive && a.Status == StatusStopped {
		a.Status = StatusRunning
	}

	return a.Status
}

func (a *Agent) IsRunning() bool {
	a.statusMu.RLock()
	defer a.statusMu.RUnlock()
	return a.Status == StatusRunning
}

func (a *Agent) AttachCommand() string {
	return fmt.Sprintf("tmux attach-session -t %s", a.Session)
}
