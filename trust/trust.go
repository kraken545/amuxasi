package trust

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	Hashes map[string]string `json:"hashes"`
	path   string
}

func LoadStore(configDir string) (*Store, error) {
	path := filepath.Join(configDir, "trust.json")
	s := &Store{
		Hashes: make(map[string]string),
		path:   path,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read trust store: %w", err)
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parse trust store: %w", err)
	}
	return s, nil
}

func (s *Store) IsApproved(scriptPath string) bool {
	hash, err := HashFile(scriptPath)
	if err != nil {
		return false
	}
	stored, ok := s.Hashes[scriptPath]
	return ok && stored == hash
}

func (s *Store) Approve(scriptPath string) error {
	hash, err := HashFile(scriptPath)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	s.Hashes[scriptPath] = hash
	return s.save()
}

func (s *Store) Revoke(scriptPath string) error {
	delete(s.Hashes, scriptPath)
	return s.save()
}

func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trust store: %w", err)
	}
	// Escritura atómica: escribir a archivo temporal + renombrar
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write trust store (tmp): %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath) // cleanup
		return fmt.Errorf("rename trust store: %w", err)
	}
	return nil
}

func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}
