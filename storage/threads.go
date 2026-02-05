package storage

import (
	"encoding/json"
	"fmt"
	"lilmail/models"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	threadBucket = []byte("threads")
	labelBucket  = []byte("labels")
	emailLabelBucket = []byte("email_labels")
)

// ThreadStorage manages thread storage using BoltDB
type ThreadStorage struct {
	db *bolt.DB
}

// NewThreadStorage creates a new thread storage
func NewThreadStorage(dbPath string) (*ThreadStorage, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(threadBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(labelBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(emailLabelBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create buckets: %v", err)
	}

	return &ThreadStorage{db: db}, nil
}

// Close closes the database
func (ts *ThreadStorage) Close() error {
	return ts.db.Close()
}

// SaveThread saves a thread to storage
func (ts *ThreadStorage) SaveThread(thread *models.EmailThread) error {
	return ts.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(threadBucket)
		
		encoded, err := json.Marshal(thread)
		if err != nil {
			return fmt.Errorf("failed to encode thread: %v", err)
		}
		
		return b.Put([]byte(thread.ID), encoded)
	})
}

// GetThread retrieves a thread by ID
func (ts *ThreadStorage) GetThread(id string) (*models.EmailThread, error) {
	var thread models.EmailThread
	
	err := ts.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(threadBucket)
		data := b.Get([]byte(id))
		
		if data == nil {
			return fmt.Errorf("thread not found")
		}
		
		return json.Unmarshal(data, &thread)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &thread, nil
}

// GetAllThreads retrieves all threads
func (ts *ThreadStorage) GetAllThreads() ([]*models.EmailThread, error) {
	var threads []*models.EmailThread
	
	err := ts.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(threadBucket)
		
		return b.ForEach(func(k, v []byte) error {
			var thread models.EmailThread
			if err := json.Unmarshal(v, &thread); err != nil {
				return err
			}
			threads = append(threads, &thread)
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return threads, nil
}

// SaveLabel saves a label
func (ts *ThreadStorage) SaveLabel(label *models.Label) error {
	return ts.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(labelBucket)
		
		encoded, err := json.Marshal(label)
		if err != nil {
			return fmt.Errorf("failed to encode label: %v", err)
		}
		
		return b.Put([]byte(label.ID), encoded)
	})
}

// GetLabel retrieves a label by ID
func (ts *ThreadStorage) GetLabel(id string) (*models.Label, error) {
	var label models.Label
	
	err := ts.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(labelBucket)
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

// GetAllLabels retrieves all labels
func (ts *ThreadStorage) GetAllLabels() ([]*models.Label, error) {
	var labels []*models.Label
	
	err := ts.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(labelBucket)
		
		return b.ForEach(func(k, v []byte) error {
			var label models.Label
			if err := json.Unmarshal(v, &label); err != nil {
				return err
			}
			labels = append(labels, &label)
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return labels, nil
}

// AddLabelToEmail associates a label with an email
func (ts *ThreadStorage) AddLabelToEmail(emailID, labelID string) error {
	return ts.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(emailLabelBucket)
		
		emailLabel := models.EmailLabel{
			EmailID: emailID,
			LabelID: labelID,
		}
		
		encoded, err := json.Marshal(emailLabel)
		if err != nil {
			return fmt.Errorf("failed to encode email label: %v", err)
		}
		
		key := fmt.Sprintf("%s:%s", emailID, labelID)
		return b.Put([]byte(key), encoded)
	})
}

// RemoveLabelFromEmail removes a label from an email
func (ts *ThreadStorage) RemoveLabelFromEmail(emailID, labelID string) error {
	return ts.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(emailLabelBucket)
		key := fmt.Sprintf("%s:%s", emailID, labelID)
		return b.Delete([]byte(key))
	})
}

// GetLabelsByEmail retrieves all labels for an email
func (ts *ThreadStorage) GetLabelsByEmail(emailID string) ([]*models.Label, error) {
	var labels []*models.Label
	
	err := ts.db.View(func(tx *bolt.Tx) error {
		emailLabelB := tx.Bucket(emailLabelBucket)
		labelB := tx.Bucket(labelBucket)
		
		return emailLabelB.ForEach(func(k, v []byte) error {
			var emailLabel models.EmailLabel
			if err := json.Unmarshal(v, &emailLabel); err != nil {
				return err
			}
			
			if emailLabel.EmailID == emailID {
				labelData := labelB.Get([]byte(emailLabel.LabelID))
				if labelData != nil {
					var label models.Label
					if err := json.Unmarshal(labelData, &label); err != nil {
						return err
					}
					labels = append(labels, &label)
				}
			}
			
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return labels, nil
}

// DeleteLabel deletes a label and all its associations
func (ts *ThreadStorage) DeleteLabel(labelID string) error {
	return ts.db.Update(func(tx *bolt.Tx) error {
		// Delete label
		if err := tx.Bucket(labelBucket).Delete([]byte(labelID)); err != nil {
			return err
		}
		
		// Delete all email-label associations
		emailLabelB := tx.Bucket(emailLabelBucket)
		keysToDelete := [][]byte{}
		
		emailLabelB.ForEach(func(k, v []byte) error {
			var emailLabel models.EmailLabel
			if err := json.Unmarshal(v, &emailLabel); err == nil {
				if emailLabel.LabelID == labelID {
					keysToDelete = append(keysToDelete, k)
				}
			}
			return nil
		})
		
		for _, key := range keysToDelete {
			if err := emailLabelB.Delete(key); err != nil {
				return err
			}
		}
		
		return nil
	})
}
