package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"lilmail/models"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ThreadStorage manages thread data persistence
type ThreadStorage struct {
	dataDir string
	mu      sync.RWMutex
}

// NewThreadStorage creates a new thread storage instance
func NewThreadStorage(dataDir string) (*ThreadStorage, error) {
	threadDir := filepath.Join(dataDir, "threads")
	if err := os.MkdirAll(threadDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create threads directory: %v", err)
	}

	return &ThreadStorage{
		dataDir: threadDir,
	}, nil
}

// SaveThread saves a thread
func (s *ThreadStorage) SaveThread(thread *models.EmailThread) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thread.ID == "" {
		thread.ID = uuid.New().String()
	}

	now := time.Now()
	if thread.CreatedAt.IsZero() {
		thread.CreatedAt = now
	}
	thread.UpdatedAt = now

	return s.saveThread(thread)
}

// GetThread retrieves a thread by ID
func (s *ThreadStorage) GetThread(threadID string) (*models.EmailThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadThread(threadID)
}

// GetThreadsByFolder retrieves all threads for a folder
func (s *ThreadStorage) GetThreadsByFolder(userID, folder string) ([]*models.EmailThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read threads directory: %v", err)
	}

	var threads []*models.EmailThread
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		threadID := file.Name()[:len(file.Name())-5]
		thread, err := s.loadThread(threadID)
		if err != nil {
			continue
		}

		if thread.UserID == userID && thread.Folder == folder {
			threads = append(threads, thread)
		}
	}

	return threads, nil
}

// GetThreadsByUser retrieves all threads for a user
func (s *ThreadStorage) GetThreadsByUser(userID string) ([]*models.EmailThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read threads directory: %v", err)
	}

	var threads []*models.EmailThread
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		threadID := file.Name()[:len(file.Name())-5]
		thread, err := s.loadThread(threadID)
		if err != nil {
			continue
		}

		if thread.UserID == userID {
			threads = append(threads, thread)
		}
	}

	return threads, nil
}

// UpdateThread updates an existing thread
func (s *ThreadStorage) UpdateThread(thread *models.EmailThread) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadThread(thread.ID)
	if err != nil {
		return fmt.Errorf("thread not found: %v", err)
	}

	thread.CreatedAt = existing.CreatedAt
	thread.UpdatedAt = time.Now()

	return s.saveThread(thread)
}

// DeleteThread deletes a thread
func (s *ThreadStorage) DeleteThread(threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	threadPath := filepath.Join(s.dataDir, threadID+".json")
	if err := os.Remove(threadPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("thread not found")
		}
		return fmt.Errorf("failed to delete thread: %v", err)
	}

	return nil
}

// DeleteThreadsByFolder deletes all threads in a folder
func (s *ThreadStorage) DeleteThreadsByFolder(userID, folder string) error {
	threads, err := s.GetThreadsByFolder(userID, folder)
	if err != nil {
		return err
	}

	for _, thread := range threads {
		if err := s.DeleteThread(thread.ID); err != nil {
			return err
		}
	}

	return nil
}

// saveThread saves thread to file (must be called with lock held)
func (s *ThreadStorage) saveThread(thread *models.EmailThread) error {
	threadPath := filepath.Join(s.dataDir, thread.ID+".json")

	data, err := json.MarshalIndent(thread, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal thread: %v", err)
	}

	return os.WriteFile(threadPath, data, 0600)
}

// loadThread loads thread from file (must be called with lock held)
func (s *ThreadStorage) loadThread(threadID string) (*models.EmailThread, error) {
	threadPath := filepath.Join(s.dataDir, threadID+".json")

	data, err := os.ReadFile(threadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("thread not found")
		}
		return nil, fmt.Errorf("failed to read thread file: %v", err)
	}

	var thread models.EmailThread
	if err := json.Unmarshal(data, &thread); err != nil {
		return nil, fmt.Errorf("failed to unmarshal thread: %v", err)
	}

	return &thread, nil
}
