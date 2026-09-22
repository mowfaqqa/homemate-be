package waitlist

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrDuplicateEmail = errors.New("This email is already on the waitlist.")

type Filters struct {
	Search            string
	AssistantInterest *bool
}

type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]Entry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]Entry),
	}
}

func (s *MemoryStore) Add(_ context.Context, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.entries[entry.Email]; exists {
		return ErrDuplicateEmail
	}

	entry.ID = int64(len(s.entries) + 1)
	entry.CreatedAt = time.Now().UTC()
	s.entries[entry.Email] = entry
	return nil
}

func (s *MemoryStore) List(_ context.Context, filters Filters) ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	search := strings.ToLower(strings.TrimSpace(filters.Search))
	entries := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		if filters.AssistantInterest != nil && entry.AssistantInterest != *filters.AssistantInterest {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(entry.FirstName+" "+entry.Email+" "+entry.City), search) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	return entries, nil
}
