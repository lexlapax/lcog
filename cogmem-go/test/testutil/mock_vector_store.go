package testutil

import (
	"context"
	"sync"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
)

// MockVectorStore implements the ltm.VectorCapableLTMStore interface for testing
type MockVectorStore struct {
	mu                 sync.RWMutex
	records            map[string]ltm.MemoryRecord
	lastQueryEmbedding []float32
}

// NewMockVectorStore creates a new mock LTM store with vector capabilities
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		records: make(map[string]ltm.MemoryRecord),
	}
}

// SupportsVectorSearch returns true as this is a vector-capable store
func (m *MockVectorStore) SupportsVectorSearch() bool {
	return true
}

// Store implements the LTMStore interface
func (m *MockVectorStore) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Store the record by ID
	m.records[record.ID] = record
	
	return record.ID, nil
}

// Retrieve implements the LTMStore interface
func (m *MockVectorStore) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Store the query embedding for test verification
	if len(query.Embedding) > 0 {
		m.lastQueryEmbedding = query.Embedding
	}
	
	// For testing, just return any records that match the entity ID check
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}
	
	var results []ltm.MemoryRecord
	for _, record := range m.records {
		if record.EntityID == entityCtx.EntityID {
			if record.AccessLevel != entity.PrivateToUser || record.UserID == entityCtx.UserID {
				results = append(results, record)
			}
		}
	}
	
	return results, nil
}

// Update implements the LTMStore interface
func (m *MockVectorStore) Update(ctx context.Context, record ltm.MemoryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Update the record
	m.records[record.ID] = record
	
	return nil
}

// Delete implements the LTMStore interface
func (m *MockVectorStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Delete the record
	delete(m.records, id)
	
	return nil
}

// GetRecord is a helper method for tests to directly access a stored record
func (m *MockVectorStore) GetRecord(id string) *ltm.MemoryRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	record, ok := m.records[id]
	if !ok {
		return nil
	}
	
	return &record
}

// GetLastQueryEmbedding returns the last query embedding that was used
func (m *MockVectorStore) GetLastQueryEmbedding() []float32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.lastQueryEmbedding
}