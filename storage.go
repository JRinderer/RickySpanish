package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// dataDir returns the platform-appropriate directory for application data.
func dataDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		base = filepath.Join(base, "rickspanish")
	default:
		// macOS and Linux: prefer XDG_DATA_HOME, fallback to ~/.local/share
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			base = filepath.Join(xdg, "rickspanish")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share", "rickspanish")
		}
	}
	return base, nil
}

// Storage manages the encrypted project database.
type Storage struct {
	path string
	key  []byte
}

// NewStorage initialises the storage, creating the data directory if needed.
func NewStorage() (*Storage, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, fmt.Errorf("resolving data directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	key, err := getOrCreateEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("obtaining encryption key: %w", err)
	}

	return &Storage{
		path: filepath.Join(dir, "projects.enc"),
		key:  key,
	}, nil
}

// load reads and decrypts the database from disk. Returns an empty database if
// the file does not yet exist.
func (s *Storage) load() (*Database, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &Database{Projects: []Project{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading data file: %w", err)
	}

	plaintext, err := decrypt(s.key, data)
	if err != nil {
		return nil, fmt.Errorf("decrypting database: %w", err)
	}

	var db Database
	if err := json.Unmarshal(plaintext, &db); err != nil {
		return nil, fmt.Errorf("parsing database: %w", err)
	}
	if db.Projects == nil {
		db.Projects = []Project{}
	}
	return &db, nil
}

// save encrypts and writes the database to disk atomically.
func (s *Storage) save(db *Database) error {
	plaintext, err := json.Marshal(db)
	if err != nil {
		return fmt.Errorf("serialising database: %w", err)
	}

	ciphertext, err := encrypt(s.key, plaintext)
	if err != nil {
		return fmt.Errorf("encrypting database: %w", err)
	}

	// Write to a temp file then rename for atomicity
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, ciphertext, 0600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("committing database: %w", err)
	}
	return nil
}

// --- Project CRUD operations ---

func (s *Storage) AddProject(p Project) error {
	db, err := s.load()
	if err != nil {
		return err
	}
	db.Projects = append(db.Projects, p)
	return s.save(db)
}

func (s *Storage) ListProjects() ([]Project, error) {
	db, err := s.load()
	if err != nil {
		return nil, err
	}
	return db.Projects, nil
}

func (s *Storage) GetProject(id string) (*Project, error) {
	db, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range db.Projects {
		if db.Projects[i].ID == id || db.Projects[i].Name == id {
			return &db.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not found", id)
}

func (s *Storage) UpdateProject(updated Project) error {
	db, err := s.load()
	if err != nil {
		return err
	}
	for i := range db.Projects {
		if db.Projects[i].ID == updated.ID {
			db.Projects[i] = updated
			return s.save(db)
		}
	}
	return fmt.Errorf("project %q not found", updated.ID)
}

func (s *Storage) DeleteProject(id string) error {
	db, err := s.load()
	if err != nil {
		return err
	}
	for i, p := range db.Projects {
		if p.ID == id || p.Name == id {
			db.Projects = append(db.Projects[:i], db.Projects[i+1:]...)
			return s.save(db)
		}
	}
	return fmt.Errorf("project %q not found", id)
}

func (s *Storage) AddNote(projectID, noteID, content string) error {
	db, err := s.load()
	if err != nil {
		return err
	}
	for i := range db.Projects {
		if db.Projects[i].ID == projectID || db.Projects[i].Name == projectID {
			db.Projects[i].Notes = append(db.Projects[i].Notes, Note{
				ID:        noteID,
				Content:   content,
				CreatedAt: now(),
			})
			db.Projects[i].UpdatedAt = now()
			return s.save(db)
		}
	}
	return fmt.Errorf("project %q not found", projectID)
}

func (s *Storage) DeleteNote(projectID, noteID string) error {
	db, err := s.load()
	if err != nil {
		return err
	}
	for i := range db.Projects {
		p := &db.Projects[i]
		if p.ID == projectID || p.Name == projectID {
			for j, n := range p.Notes {
				if n.ID == noteID || (len(n.ID) >= 8 && n.ID[:8] == noteID) {
					p.Notes = append(p.Notes[:j], p.Notes[j+1:]...)
					p.UpdatedAt = now()
					return s.save(db)
				}
			}
			return fmt.Errorf("note %q not found in project %q", noteID, projectID)
		}
	}
	return fmt.Errorf("project %q not found", projectID)
}
