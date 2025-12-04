package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Store struct {
	entries []Entry
	mu      sync.Mutex
}

// NewStore returns a new store
func NewStore() (*Store, error) {
	f, err := os.OpenFile(DBPATH, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return &Store{}, err
}

// List lists all entries in the store
func (s *Store) List() ([]*Entry, error) {
	b, err := os.ReadFile(DBPATH)
	if err != nil {
		return nil, err
	}

	var result []*Entry
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Save adds an entry to the store
func (s *Store) Save(e *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.List()
	if err != nil {
		return err
	}

	entries = append(entries, e)

	b, err := json.Marshal(&entries)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(".", "messages-")
	if err != nil {
		return err
	}

	_, err = tmp.Write(b)
	if err != nil {
		return err
	}

	return os.Rename(tmp.Name(), DBPATH)
}

// Pop returns the first element from the entries slice. The
// popped element is then removed from the store.
func (s *Store) Pop() (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.List()
	if err != nil {
		return nil, err
	}

	// Don't try do pop an entry from an empty list
	if len(entries) < 1 {
		return nil, fmt.Errorf("can't pop entry from empty list")
	}

	popped := entries[0]
	entries = entries[1:]

	b, err := json.Marshal(&entries)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp(".", "messages-")
	if err != nil {
		return nil, err
	}

	_, err = tmp.Write(b)
	if err != nil {
		return nil, err
	}

	err = os.Rename(tmp.Name(), DBPATH)
	if err != nil {
		return nil, err
	}

	return popped, nil
}
