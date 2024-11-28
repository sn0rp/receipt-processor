package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu       sync.RWMutex
	receipts map[string]*Receipt
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		receipts: make(map[string]*Receipt),
	}
}

func (s *MemoryStore) SaveReceipt(receipt *Receipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if receipt.ID == "" {
		receipt.ID = uuid.New().String()
	}
	receipt.CreatedAt = time.Now()
	s.receipts[receipt.ID] = receipt
	return nil
}

func (s *MemoryStore) GetReceipt(id string) (*Receipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	receipt, exists := s.receipts[id]
	if !exists {
		return nil, fmt.Errorf("receipt not found: %s", id)
	}
	return receipt, nil
}

func (s *MemoryStore) ListReceipts() ([]*Receipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	receipts := make([]*Receipt, 0, len(s.receipts))
	for _, receipt := range s.receipts {
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}
