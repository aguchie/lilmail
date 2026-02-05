package storage

import (
	"encoding/json"
	"fmt"
	"lilmail/models"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// DraftStorage handles draft email persistence
type DraftStorage struct {
	baseDir string
}

// NewDraftStorage creates a new draft storage instance
func NewDraftStorage(baseDir string) *DraftStorage {
	return &DraftStorage{
		baseDir: baseDir,
	}
}

// getDraftDir returns the drafts directory for a user
func (ds *DraftStorage) getDraftDir(userID string) string {
	return filepath.Join(ds.baseDir, "drafts", userID)
}

// SaveDraft saves or updates a draft
func (ds *DraftStorage) SaveDraft(userID, draftID string, draft *models.Draft) error {
	dir := ds.getDraftDir(userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create draft directory: %w", err)
	}

	// Generate new ID if not provided
	if draftID == "" {
		draftID = uuid.New().String()
		draft.CreatedAt = time.Now()
	}
	draft.ID = draftID
	draft.UserID = userID
	draft.UpdatedAt = time.Now()

	// Serialize draft
	data, err := json.MarshalIndent(draft, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal draft: %w", err)
	}

	// Write to file
	filePath := filepath.Join(dir, draftID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write draft file: %w", err)
	}

	return nil
}

// GetDraft retrieves a specific draft
func (ds *DraftStorage) GetDraft(userID, draftID string) (*models.Draft, error) {
	filePath := filepath.Join(ds.getDraftDir(userID), draftID+".json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("draft not found")
		}
		return nil, fmt.Errorf("failed to read draft: %w", err)
	}

	var draft models.Draft
	if err := json.Unmarshal(data, &draft); err != nil {
		return nil, fmt.Errorf("failed to unmarshal draft: %w", err)
	}

	return &draft, nil
}

// GetDrafts retrieves all drafts for a user
func (ds *DraftStorage) GetDrafts(userID string) ([]*models.Draft, error) {
	dir := ds.getDraftDir(userID)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create draft directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read drafts directory: %w", err)
	}

	var drafts []*models.Draft
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		draftID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		draft, err := ds.GetDraft(userID, draftID)
		if err != nil {
			continue // Skip invalid drafts
		}

		drafts = append(drafts, draft)
	}

	// Sort by update time (newest first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(drafts)-1; i++ {
		for j := i + 1; j < len(drafts); j++ {
			if drafts[i].UpdatedAt.Before(drafts[j].UpdatedAt) {
				drafts[i], drafts[j] = drafts[j], drafts[i]
			}
		}
	}

	return drafts, nil
}

// DeleteDraft deletes a draft
func (ds *DraftStorage) DeleteDraft(userID, draftID string) error {
	filePath := filepath.Join(ds.getDraftDir(userID), draftID+".json")
	
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("draft not found")
		}
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	return nil
}

// DeleteAllDrafts deletes all drafts for a user
func (ds *DraftStorage) DeleteAllDrafts(userID string) error {
	dir := ds.getDraftDir(userID)
	
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete drafts: %w", err)
	}

	return nil
}
