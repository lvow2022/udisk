package ufs

import (
	"gorm.io/gorm"
	"sync"
)

// UserManager manages file systems for multiple users.
type UserManager struct {
	users map[string]*UserFileSystem
	mutex sync.RWMutex
	db    *gorm.DB
}

// NewUserManager creates a new UserManager instance.
func NewUserManager(db *gorm.DB) *UserManager {
	return &UserManager{
		users: make(map[string]*UserFileSystem),
		db:    db,
	}
}

// User returns the file system for a specific user, creating it if necessary.
func (um *UserManager) User(username string) *UserFileSystem {
	um.mutex.RLock()
	ufs, ok := um.users[username]
	um.mutex.RUnlock()
	if ok {
		return ufs
	}

	um.mutex.Lock()
	defer um.mutex.Unlock()

	// Check again to ensure no race condition occurred.
	if ufs, ok := um.users[username]; ok {
		return ufs
	}

	ufs = NewUserFileSystem(um.db)
	um.users[username] = ufs
	return ufs
}
