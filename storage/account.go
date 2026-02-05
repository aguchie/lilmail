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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AccountStorage manages account data persistence
type AccountStorage struct {
	dataDir string
	mu      sync.RWMutex
}

// NewAccountStorage creates a new account storage instance
func NewAccountStorage(dataDir string) (*AccountStorage, error) {
	// Create data directory if it doesn't exist
	accountDir := filepath.Join(dataDir, "accounts")
	if err := os.MkdirAll(accountDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create accounts directory: %v", err)
	}

	return &AccountStorage{
		dataDir: accountDir,
	}, nil
}

// CreateAccount creates a new account
func (s *AccountStorage) CreateAccount(account *models.Account, encryptionKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	// Save to file
	return s.saveAccount(&storedAccount)
}

// GetAccount retrieves an account by ID
func (s *AccountStorage) GetAccount(accountID string, encryptionKey []byte) (*models.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, err := s.loadAccount(accountID)
	if err != nil {
		return nil, err
	}

	// Decrypt password
	decryptedPassword, err := decrypt(account.Password, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %v", err)
	}
	account.Password = decryptedPassword

	return account, nil
}

// GetAccountsByUser retrieves all accounts for a user
func (s *AccountStorage) GetAccountsByUser(userID string, encryptionKey []byte) ([]*models.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read accounts directory: %v", err)
	}

	var accounts []*models.Account
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		accountID := file.Name()[:len(file.Name())-5] // Remove .json extension
		account, err := s.loadAccount(accountID)
		if err != nil {
			continue // Skip invalid accounts
		}

		if account.UserID == userID {
			// Decrypt password
			decryptedPassword, err := decrypt(account.Password, encryptionKey)
			if err != nil {
				continue // Skip accounts with decryption errors
			}
			account.Password = decryptedPassword
			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

// UpdateAccount updates an existing account
func (s *AccountStorage) UpdateAccount(account *models.Account, encryptionKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if account exists
	existing, err := s.loadAccount(account.ID)
	if err != nil {
		return fmt.Errorf("account not found: %v", err)
	}

	// Update timestamp
	account.UpdatedAt = time.Now()
	account.CreatedAt = existing.CreatedAt

	// Encrypt password
	encryptedPassword, err := encrypt(account.Password, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}

	// Create a copy with encrypted password for storage
	storedAccount := *account
	storedAccount.Password = encryptedPassword

	return s.saveAccount(&storedAccount)
}

// DeleteAccount deletes an account
func (s *AccountStorage) DeleteAccount(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	accountPath := filepath.Join(s.dataDir, accountID+".json")
	if err := os.Remove(accountPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("account not found")
		}
		return fmt.Errorf("failed to delete account: %v", err)
	}

	return nil
}

// saveAccount saves account to file (must be called with lock held)
func (s *AccountStorage) saveAccount(account *models.Account) error {
	accountPath := filepath.Join(s.dataDir, account.ID+".json")

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account: %v", err)
	}

	return os.WriteFile(accountPath, data, 0600)
}

// loadAccount loads account from file (must be called with lock held)
func (s *AccountStorage) loadAccount(accountID string) (*models.Account, error) {
	accountPath := filepath.Join(s.dataDir, accountID+".json")

	data, err := os.ReadFile(accountPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("account not found")
		}
		return nil, fmt.Errorf("failed to read account file: %v", err)
	}

	var account models.Account
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %v", err)
	}

	return &account, nil
}

// encrypt encrypts plaintext using AES-GCM
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
