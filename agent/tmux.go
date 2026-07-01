package agent

import (
	"fmt"
	"os/exec"
	"strings"
)

func TmuxExists() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func TmuxNewSession(name, command string, env []string) error {
	args := []string{"new-session", "-d", "-s", name}
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, command)
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux new-session: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func TmuxKillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if !strings.Contains(outStr, "no server running") &&
			!strings.Contains(outStr, "session not found") {
			return fmt.Errorf("tmux kill-session: %s: %w", outStr, err)
		}
	}
	return nil
}

func TmuxCapturePane(name string) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-S", "-", "-e")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func TmuxHasSession(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

func TmuxListSessions(prefix string) []string {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var sessions []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			sessions = append(sessions, line)
		}
	}
	return sessions
}

func TmuxSendKeys(name, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", name, keys, "Enter")
	return cmd.Run()
}

func SessionName(workspace, agentName string) string {
	safeWs := strings.NewReplacer("/", "-", ".", "-", " ", "-").Replace(workspace)
	return fmt.Sprintf("amuxasi-%s-%s", safeWs, agentName)
}
