package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"personal-ledger/internal/ledger"
)

// Instances share a lock so opening the same path twice cannot lose an append.
var jsonStoreMu sync.RWMutex

type JSONStore struct {
	path string
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func (s *JSONStore) List() ([]ledger.Record, error) {
	jsonStoreMu.RLock()
	defer jsonStoreMu.RUnlock()
	return s.read()
}

func (s *JSONStore) Append(record ledger.Record) error {
	jsonStoreMu.Lock()
	defer jsonStoreMu.Unlock()

	records, err := s.read()
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create ledger directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".ledger-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary ledger: %w", err)
	}
	defer func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}()

	if err := json.NewEncoder(temp).Encode(append(records, record)); err != nil {
		return fmt.Errorf("write temporary ledger: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary ledger: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary ledger: %w", err)
	}
	if err := os.Rename(temp.Name(), s.path); err != nil {
		return fmt.Errorf("replace ledger: %w", err)
	}
	return nil
}

func (s *JSONStore) read() ([]ledger.Record, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []ledger.Record{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}

	var entries []*ledger.Record
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode ledger: %w", err)
	}
	if entries == nil {
		return nil, fmt.Errorf("decode ledger: expected an array of records")
	}

	records := make([]ledger.Record, len(entries))
	for i, entry := range entries {
		if entry == nil {
			return nil, fmt.Errorf("decode ledger: record %d is null", i+1)
		}
		records[i] = *entry
	}
	return records, nil
}
