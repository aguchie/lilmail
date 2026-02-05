package storage

import (
	"encoding/json"
	"fmt"
	"lilmail/models"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

const (
	labelBucket      = "labels"
	emailLabelBucket = "email_labels"
)

// LabelStorage manages label data persistence using BoltDB
type LabelStorage struct {
	db *bbolt.DB
}

// NewLabelStorage creates a new label storage instance
func NewLabelStorage(dataDir string) (*LabelStorage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "lilmail.db")
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Initialize buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(labelBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(emailLabelBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize buckets: %v", err)
	}

	return &LabelStorage{db: db}, nil
}

// Close closes the database connection
func (s *LabelStorage) Close() error {
	return s.db.Close()
}

// CreateLabel creates a new label
func (s *LabelStorage) CreateLabel(label *models.Label) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(labelBucket))
		
		key := []byte(label.ID)
		
		data, err := json.Marshal(label)
		if err != nil {
			return err
		}
		
		return b.Put(key, data)
	})
}

// GetLabelsByUser retrieves all labels for a user
func (s *LabelStorage) GetLabelsByUser(userID string) ([]models.Label, error) {
	var labels []models.Label
	
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(labelBucket))
		
		return b.ForEach(func(k, v []byte) error {
			var label models.Label
			if err := json.Unmarshal(v, &label); err != nil {
				return err
			}
			
			if label.UserID == userID {
				labels = append(labels, label)
			}
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return labels, nil
}

// GetLabel retrieves a specific label
func (s *LabelStorage) GetLabel(id string) (*models.Label, error) {
	var label models.Label
	
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(labelBucket))
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("label not found")
		}
		
		return json.Unmarshal(data, &label)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &label, nil
}

// DeleteLabel deletes a label
func (s *LabelStorage) DeleteLabel(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(labelBucket))
		
		// Delete logical label
		if err := b.Delete([]byte(id)); err != nil {
			return err
		}
		
		// TODO: Also delete associated email_labels? 
		// For simplicity/performance, we might leave them or clean up lazily.
		// A full scan to clean up might be expensive.
		return nil
	})
}

// AssignLabel assigns a label to an email
func (s *LabelStorage) AssignLabel(emailID, labelID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(emailLabelBucket))
		
		key := []byte(fmt.Sprintf("%s:%s", emailID, labelID))
		el := models.EmailLabel{
			EmailID: emailID,
			LabelID: labelID,
		}
		
		data, err := json.Marshal(el)
		if err != nil {
			return err
		}
		
		return b.Put(key, data)
	})
}

// RemoveLabel removes a label from an email
func (s *LabelStorage) RemoveLabel(emailID, labelID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(emailLabelBucket))
		
		key := []byte(fmt.Sprintf("%s:%s", emailID, labelID))
		return b.Delete(key)
	})
}

// GetLabelsForEmail retrieves all labels for a specific email
func (s *LabelStorage) GetLabelsForEmail(emailID string) ([]models.Label, error) {
	var labelIDs []string
	
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(emailLabelBucket))
		c := b.Cursor()
		
		prefix := []byte(emailID + ":")
		for k, v := c.Seek(prefix); k != nil && bytesHasPrefix(k, prefix); k, v = c.Next() {
			var el models.EmailLabel
			if err := json.Unmarshal(v, &el); err == nil {
				labelIDs = append(labelIDs, el.LabelID)
			}
		}
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	var labels []models.Label
	for _, id := range labelIDs {
		l, err := s.GetLabel(id)
		if err == nil {
			labels = append(labels, *l)
		}
	}
	
	return labels, nil
}

// Helper for prefix check
func bytesHasPrefix(s, prefix []byte) bool {
	return len(s) >= len(prefix) && string(s[0:len(prefix)]) == string(prefix)
}
