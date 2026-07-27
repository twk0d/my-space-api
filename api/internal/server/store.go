package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type Currently struct {
	Listening string      `json:"listening"`
	IsOnline  bool        `json:"isOnline"`
	Track     *MusicTrack `json:"track,omitempty"`
}

type SignatureStatus string

const (
	SignaturePending  SignatureStatus = "pending"
	SignatureApproved SignatureStatus = "approved"
	SignatureRejected SignatureStatus = "rejected"
)

type Signature struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Message    string          `json:"message"`
	Status     SignatureStatus `json:"status"`
	CreatedAt  time.Time       `json:"createdAt"`
	ApprovedAt *time.Time      `json:"approvedAt,omitempty"`
	RejectedAt *time.Time      `json:"rejectedAt,omitempty"`
}

type DataStore interface {
	Currently() (Currently, error)
	UpdateCurrently(Currently) error
	CreateSignature(name, message string, now time.Time) (Signature, error)
	Signatures(status SignatureStatus, limit, offset int) ([]Signature, error)
	UpdateSignatureStatus(id string, status SignatureStatus, now time.Time) (Signature, error)
	IncrementVisitors() (int64, error)
	VisitorCount() (int64, error)
	Close()
}

type JSONStore struct {
	path string
	mu   sync.Mutex
	data database
}

type database struct {
	Currently    Currently   `json:"currently"`
	Signatures   []Signature `json:"signatures"`
	VisitorCount int64       `json:"visitorCount"`
}

func NewJSONStore(path string) (*JSONStore, error) {
	store := &JSONStore{path: path}
	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (store *JSONStore) Currently() (Currently, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.data.Currently, nil
}

func (store *JSONStore) UpdateCurrently(currently Currently) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.data.Currently = currently

	return store.saveLocked()
}

func (store *JSONStore) CreateSignature(name, message string, now time.Time) (Signature, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	signature := Signature{
		ID:        newID(now),
		Name:      name,
		Message:   message,
		Status:    SignaturePending,
		CreatedAt: now.UTC(),
	}

	store.data.Signatures = append(store.data.Signatures, signature)

	return signature, store.saveLocked()
}

func (store *JSONStore) Signatures(status SignatureStatus, limit, offset int) ([]Signature, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	matches := make([]Signature, 0)
	for i := len(store.data.Signatures) - 1; i >= 0; i-- {
		signature := store.data.Signatures[i]
		if signature.Status == status {
			matches = append(matches, signature)
		}
	}

	return paginate(matches, limit, offset), nil
}

func (store *JSONStore) UpdateSignatureStatus(id string, status SignatureStatus, now time.Time) (Signature, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	index := slices.IndexFunc(store.data.Signatures, func(signature Signature) bool {
		return signature.ID == id
	})
	if index < 0 {
		return Signature{}, errNotFound
	}

	signature := store.data.Signatures[index]
	signature.Status = status
	signature.ApprovedAt = nil
	signature.RejectedAt = nil

	timestamp := now.UTC()
	switch status {
	case SignatureApproved:
		signature.ApprovedAt = &timestamp
	case SignatureRejected:
		signature.RejectedAt = &timestamp
	}

	store.data.Signatures[index] = signature

	return signature, store.saveLocked()
}

func (store *JSONStore) IncrementVisitors() (int64, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.data.VisitorCount++

	return store.data.VisitorCount, store.saveLocked()
}

func (store *JSONStore) VisitorCount() (int64, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.data.VisitorCount, nil
}

func (store *JSONStore) Close() {
}

func (store *JSONStore) load() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.data = defaultDatabase()

	file, err := os.Open(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return store.saveLocked()
	}
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&store.data)
}

func (store *JSONStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(store.path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(store.data)
}

func defaultDatabase() database {
	return database{
		Currently: Currently{
			Listening: "Shoegaze mix.mp3",
			IsOnline:  true,
		},
		VisitorCount: 123,
	}
}

func paginate[T any](items []T, limit, offset int) []T {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	if offset >= len(items) {
		return []T{}
	}

	end := min(offset+limit, len(items))

	return items[offset:end]
}

func newID(now time.Time) string {
	return now.UTC().Format("20060102150405.000000000")
}

var errNotFound = errors.New("not found")
