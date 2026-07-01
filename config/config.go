package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type AgentConfig struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
}

type ScriptsConfig struct {
	Setup   string `toml:"setup"`
	Run     string `toml:"run"`
	Archive string `toml:"archive"`
}

type WorkspaceConfig struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}

type Config struct {
	Workspace WorkspaceConfig            `toml:"workspace"`
	Agents    map[string]AgentConfig     `toml:"agents"`
	Launch    []string                   `toml:"launch"`
	Scripts   ScriptsConfig              `toml:"scripts"`
	Extra     map[string]toml.Unmarshaler `toml:"-"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}
	return &cfg, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func DefaultConfig() *Config {
	return &Config{
		Workspace: WorkspaceConfig{
			Name:        "",
			Description: "",
		},
		Agents: map[string]AgentConfig{
			"claude": {
				Command: "claude",
				Args:    []string{},
			},
			"opencode": {
				Command: "opencode",
				Args:    []string{},
			},
			"codex": {
				Command: "codex",
				Args:    []string{},
			},
		},
		Launch: []string{},
		Scripts: ScriptsConfig{},
	}
}
