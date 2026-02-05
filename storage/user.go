package storage

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"lilmail/models"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserStorage manages user data persistence
type UserStorage struct {
	dataDir string
	mu      sync.RWMutex
}

// NewUserStorage creates a new user storage instance
func NewUserStorage(dataDir string) (*UserStorage, error) {
	userDir := filepath.Join(dataDir, "users")
	if err := os.MkdirAll(userDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create users directory: %v", err)
	}

	return &UserStorage{
		dataDir: userDir,
	}, nil
}

// CreateUser creates a new user
func (s *UserStorage) CreateUser(user *models.User, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not set
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	user.PasswordHash = string(hashedPassword)

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Default values
	if user.Role == "" {
		user.Role = "user"
	}
	if user.Language == "" {
		user.Language = "en"
	}
	if user.Theme == "" {
		user.Theme = "light"
	}

	return s.saveUser(user)
}

// GetUser retrieves a user by ID
func (s *UserStorage) GetUser(userID string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadUser(userID)
}

// GetUserByUsername retrieves a user by username
func (s *UserStorage) GetUserByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read users directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		userID := file.Name()[:len(file.Name())-5]
		user, err := s.loadUser(userID)
		if err != nil {
			continue
		}

		if user.Username == username {
			return user, nil
		}
	}

	return nil, errors.New("user not found")
}

// GetUserByEmail retrieves a user by email
func (s *UserStorage) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read users directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		userID := file.Name()[:len(file.Name())-5]
		user, err := s.loadUser(userID)
		if err != nil {
			continue
		}

		if user.Email == email {
			return user, nil
		}
	}

	return nil, errors.New("user not found")
}

// UpdateUser updates an existing user
func (s *UserStorage) UpdateUser(user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadUser(user.ID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	// Update timestamp
	user.UpdatedAt = time.Now()
	user.CreatedAt = existing.CreatedAt

	// Preserve password hash if not updated
	if user.PasswordHash == "" {
		user.PasswordHash = existing.PasswordHash
	}

	return s.saveUser(user)
}

// UpdatePassword updates a user's password
func (s *UserStorage) UpdatePassword(userID, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.loadUser(userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	return s.saveUser(user)
}

// VerifyPassword verifies a password for a user
func (s *UserStorage) VerifyPassword(userID, password string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.loadUser(userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
}

// DeleteUser deletes a user
func (s *UserStorage) DeleteUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userPath := filepath.Join(s.dataDir, userID+".json")
	if err := os.Remove(userPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}

// ListUsers retrieves all users
func (s *UserStorage) ListUsers() ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read users directory: %v", err)
	}

	var users []*models.User
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		userID := file.Name()[:len(file.Name())-5]
		user, err := s.loadUser(userID)
		if err != nil {
			continue
		}

		users = append(users, user)
	}

	return users, nil
}

// UpdateLastLogin updates the last login timestamp
func (s *UserStorage) UpdateLastLogin(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.loadUser(userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	user.LastLoginAt = time.Now()
	user.UpdatedAt = time.Now()

	return s.saveUser(user)
}

// saveUser saves user to file (must be called with lock held)
func (s *UserStorage) saveUser(user *models.User) error {
	userPath := filepath.Join(s.dataDir, user.ID+".json")

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %v", err)
	}

	return os.WriteFile(userPath, data, 0600)
}

// loadUser loads user from file (must be called with lock held)
func (s *UserStorage) loadUser(userID string) (*models.User, error) {
	userPath := filepath.Join(s.dataDir, userID+".json")

	data, err := os.ReadFile(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to read user file: %v", err)
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %v", err)
	}

	return &user, nil
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}
