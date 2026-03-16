package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/user/http-cli/internal/models"
)

type Store struct {
	mu           sync.RWMutex
	dataDir      string
	requests     map[string]*models.Request
	collections  map[string]*models.Collection
	environments map[string]*models.Environment
	history      []*models.Response
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	s := &Store{
		dataDir:      dataDir,
		requests:     make(map[string]*models.Request),
		collections:  make(map[string]*models.Collection),
		environments: make(map[string]*models.Environment),
		history:      make([]*models.Response, 0),
	}
	s.load()
	return s, nil
}

func (s *Store) SaveRequest(ctx context.Context, req *models.Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.ID == "" {
		req.ID = generateID()
	}
	req.UpdatedAt = time.Now()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = req.UpdatedAt
	}
	s.requests[req.ID] = req
	return s.persistRequests()
}

func (s *Store) GetRequest(ctx context.Context, id string) (*models.Request, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.requests[id]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", id)
	}
	return req, nil
}

func (s *Store) ListRequests(ctx context.Context) ([]*models.Request, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*models.Request, 0, len(s.requests))
	for _, r := range s.requests {
		result = append(result, r)
	}
	return result, nil
}

func (s *Store) DeleteRequest(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requests, id)
	return s.persistRequests()
}

func (s *Store) SaveCollection(ctx context.Context, col *models.Collection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if col.ID == "" {
		col.ID = generateID()
	}
	col.UpdatedAt = time.Now()
	if col.CreatedAt.IsZero() {
		col.CreatedAt = col.UpdatedAt
	}
	s.collections[col.ID] = col
	return s.persistCollections()
}

func (s *Store) ListCollections(ctx context.Context) ([]*models.Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*models.Collection, 0, len(s.collections))
	for _, c := range s.collections {
		result = append(result, c)
	}
	return result, nil
}

func (s *Store) DeleteCollection(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.collections, id)
	return s.persistCollections()
}

func (s *Store) SaveEnvironment(ctx context.Context, env *models.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if env.ID == "" {
		env.ID = generateID()
	}
	s.environments[env.ID] = env
	return s.persistEnvironments()
}

func (s *Store) ListEnvironments(ctx context.Context) ([]*models.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*models.Environment, 0, len(s.environments))
	for _, e := range s.environments {
		result = append(result, e)
	}
	return result, nil
}

func (s *Store) GetActiveEnvironment(ctx context.Context) (*models.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.environments {
		if e.IsActive {
			return e, nil
		}
	}
	return nil, nil
}

func (s *Store) AddHistory(ctx context.Context, resp *models.Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append([]*models.Response{resp}, s.history...)
	if len(s.history) > 100 {
		s.history = s.history[:100]
	}
	return nil
}

func (s *Store) load() {
	s.loadJSON(filepath.Join(s.dataDir, "requests.json"), &s.requests)
	s.loadJSON(filepath.Join(s.dataDir, "collections.json"), &s.collections)
	s.loadJSON(filepath.Join(s.dataDir, "environments.json"), &s.environments)
}

func (s *Store) loadJSON(path string, dest interface{}) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, dest) //nolint
}

func (s *Store) persistRequests() error {
	return s.writeJSON(filepath.Join(s.dataDir, "requests.json"), s.requests)
}

func (s *Store) persistCollections() error {
	return s.writeJSON(filepath.Join(s.dataDir, "collections.json"), s.collections)
}

func (s *Store) persistEnvironments() error {
	return s.writeJSON(filepath.Join(s.dataDir, "environments.json"), s.environments)
}

func (s *Store) writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
