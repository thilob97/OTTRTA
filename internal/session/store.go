package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const sessionsFileName = "sessions.json"

type storedSession struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Kind      SessionKind `json:"kind"`
	Command   string      `json:"command"`
	Args      []string    `json:"args,omitempty"`
	WorkDir   string      `json:"workDir,omitempty"`
	AgentKind AgentKind   `json:"agentKind,omitempty"`
}

func DefaultStorePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "ottrta", sessionsFileName), nil
}

func LoadSessions(path string) ([]Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var stored []storedSession
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(stored))
	for _, item := range stored {
		sessions = append(sessions, Session{
			ID:        item.ID,
			Name:      item.Name,
			Kind:      item.Kind,
			Status:    StatusStopped,
			Command:   item.Command,
			Args:      append([]string(nil), item.Args...),
			WorkDir:   item.WorkDir,
			AgentKind: item.AgentKind,
		})
	}
	return sessions, nil
}

func SaveSessions(path string, sessions []Session) error {
	stored := make([]storedSession, 0, len(sessions))
	for _, s := range sessions {
		stored = append(stored, storedSession{
			ID:        s.ID,
			Name:      s.Name,
			Kind:      s.Kind,
			Command:   s.Command,
			Args:      append([]string(nil), s.Args...),
			WorkDir:   s.WorkDir,
			AgentKind: s.AgentKind,
		})
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, sessionsFileName+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
