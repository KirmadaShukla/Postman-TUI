package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"my-new-go/internal/models"
)

const (
	dirName  = ".apitui"
	fileName = "workspace.json"
	maxHist  = 100
)

type Store struct {
	path string
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName, fileName), nil
}

func New(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &Store{path: path}, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (models.Workspace, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		ws := models.DefaultWorkspace()
		if saveErr := s.Save(ws); saveErr != nil {
			return ws, saveErr
		}
		return ws, nil
	}
	if err != nil {
		return models.Workspace{}, err
	}

	var ws models.Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return models.Workspace{}, err
	}
	if len(ws.Collections) == 0 {
		ws = models.DefaultWorkspace()
	} else {
		models.MigrateWorkspace(&ws)
	}
	return ws, nil
}

func (s *Store) Save(ws models.Workspace) error {
	if len(ws.History) > maxHist {
		ws.History = ws.History[:maxHist]
	}
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
