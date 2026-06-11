package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type PendingLogin struct {
	DeviceID  uint
	Phone     string
	CodeHash  string
	ExpiresAt time.Time
}

type PendingStore struct {
	mu    sync.Mutex
	items map[string]PendingLogin
}

func NewPendingStore() *PendingStore {
	return &PendingStore{items: make(map[string]PendingLogin)}
}

func (s *PendingStore) Create(deviceID uint, phone, codeHash string) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[id] = PendingLogin{
		DeviceID:  deviceID,
		Phone:     phone,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	return id, nil
}

func (s *PendingStore) Get(id string) (PendingLogin, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[id]
	if !ok || time.Now().After(item.ExpiresAt) {
		delete(s.items, id)
		return PendingLogin{}, false
	}
	return item, true
}

func (s *PendingStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
