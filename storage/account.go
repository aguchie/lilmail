package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lilmail/models"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

// AccountStorage manages account data persistence using BoltDB
type AccountStorage struct {
	db *bbolt.DB
	mu sync.RWMutex
}

// NewAccountStorage creates a new account storage instance
func NewAccountStorage(db *bbolt.DB) *AccountStorage {
	return &AccountStorage{
		db: db,
	}
}

// CreateAccount creates a new account
func (s *AccountStorage) CreateAccount(account *models.Account, encryptionKey []byte) error {
	// Generate ID if not set
	if account.ID == "" {
		account.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	// Encrypt password
	encryptedPassword, err := encrypt(account.Password, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}

	// Create a copy with encrypted password for storage
	storedAccount := *account
	storedAccount.Password = encryptedPassword

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Accounts"))
		
		data, err := json.Marshal(storedAccount)
		if err != nil {
			return fmt.Errorf("failed to marshal account: %v", err)
		}

		return b.Put([]byte(account.ID), data)
	})
}

// GetAccount retrieves an account by ID
func (s *AccountStorage) GetAccount(accountID string, encryptionKey []byte) (*models.Account, error) {
	var account models.Account

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Accounts"))
		data := b.Get([]byte(accountID))
		if data == nil {
			return errors.New("account not found")
		}
		return json.Unmarshal(data, &account)
	})

	if err != nil {
		return nil, err
	}

	// Decrypt password
	decryptedPassword, err := decrypt(account.Password, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %v", err)
	}
	account.Password = decryptedPassword

	return &account, nil
}

// GetAccountsByUser retrieves all accounts for a user (Scan)
func (s *AccountStorage) GetAccountsByUser(userID string, encryptionKey []byte) ([]*models.Account, error) {
	var accounts []*models.Account

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Accounts"))
		return b.ForEach(func(k, v []byte) error {
			var account models.Account
			if err := json.Unmarshal(v, &account); err != nil {
				return nil // Skip corrupted
			}
			
			if account.UserID == userID {
				// Decrypt password
				decryptedPassword, err := decrypt(account.Password, encryptionKey)
				if err != nil {
					return nil // Skip decryption errors
				}
				account.Password = decryptedPassword
				accounts = append(accounts, &account)
			}
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// UpdateAccount updates an existing account
func (s *AccountStorage) UpdateAccount(account *models.Account, encryptionKey []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Accounts"))
		
		// Check existence and get creation time
		existingData := b.Get([]byte(account.ID))
		if existingData == nil {
			return errors.New("account not found")
		}
		var existing models.Account
		json.Unmarshal(existingData, &existing)

		// Create copy to store
		toStore := *account
		toStore.CreatedAt = existing.CreatedAt
		toStore.UpdatedAt = time.Now()

		// Encrypt password
		encryptedPassword, err := encrypt(account.Password, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %v", err)
		}
		toStore.Password = encryptedPassword

		data, err := json.Marshal(toStore)
		if err != nil {
			return err
		}

		return b.Put([]byte(account.ID), data)
	})
}

// DeleteAccount deletes an account
func (s *AccountStorage) DeleteAccount(accountID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Accounts"))
		return b.Delete([]byte(accountID))
	})
}

// encrypt encrypts plaintext using AES-GCM
// Copied from original file
func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return fmt.Sprintf("%x", ciphertext), nil
}

// decrypt decrypts ciphertext using AES-GCM
func decrypt(ciphertextHex string, key []byte) (string, error) {
	var ciphertext []byte
	if _, err := fmt.Sscanf(ciphertextHex, "%x", &ciphertext); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
