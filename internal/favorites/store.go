package favorites

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"api-banque-datadex/internal/finance"
)

type Store struct {
	filePath string
	mu       sync.Mutex
}

func NewStore(filePath string) (*Store, error) {
	if filePath == "" {
		return nil, errors.New("chemin du fichier de favoris vide")
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(filePath, []byte("[]\n"), 0644); err != nil {
			return nil, err
		}
	}

	return &Store{filePath: filePath}, nil
}

func (s *Store) GetAll() ([]finance.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return []finance.Transaction{}, nil
	}

	var items []finance.Transaction
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) saveAll(items []finance.Transaction) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) Add(tx finance.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var items []finance.Transaction
	if len(data) > 0 {
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
	}

	for _, existing := range items {
		if existing.ID == tx.ID && tx.ID != "" {
			return nil
		}
	}

	items = append(items, tx)
	return s.saveAll(items)
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var items []finance.Transaction
	if len(data) > 0 {
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
	}

	var updated []finance.Transaction
	for _, item := range items {
		if item.ID != id {
			updated = append(updated, item)
		}
	}

	if updated == nil {
		updated = []finance.Transaction{}
	}

	return s.saveAll(updated)
}