package storage

import (
	"fmt"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

// InitDB initializes the database connection
func InitDB(dataDir string) (*bbolt.DB, error) {
	dbPath := filepath.Join(dataDir, "lilmail.db")

	// Open the database
	// It will be created if it doesn't exist.
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{"Users", "Accounts", "UserEmails"} // Added UserEmails for index
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("create bucket %s: %s", bucket, err)
			}
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
