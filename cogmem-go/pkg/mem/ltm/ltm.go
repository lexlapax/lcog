package ltm

import (
	"context"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
)

// MemoryRecord represents a single memory entry in long-term memory.
type MemoryRecord struct {
	// ID is a unique identifier for the record
	ID string
	
	// EntityID is the entity that owns this memory
	EntityID entity.EntityID
	
	// UserID is optional and indicates a specific user within the entity
	// Used with AccessLevel.PrivateToUser
	UserID string
	
	// AccessLevel determines the visibility of this memory within the entity
	AccessLevel entity.AccessLevel
	
	// Content is the actual memory content (text)
	Content string
	
	// Metadata is additional structured data about this memory
	Metadata map[string]interface{}
	
	// Embedding is the vector representation for semantic search (empty for SQL/KV stores)
	Embedding []float32
	
	// CreatedAt is when this memory was initially stored
	CreatedAt time.Time
	
	// UpdatedAt is when this memory was last modified
	UpdatedAt time.Time
}

// LTMQuery represents a query to retrieve memories from LTM.
type LTMQuery struct {
	// ExactMatch is used for key-based exact matching (SQL, KV stores)
	ExactMatch map[string]interface{}
	
	// TextMatch is used for text-based search (SQL, potentially Vector stores)
	Text string
	
	// EmbeddingMatch is used for semantic search (Vector stores)
	Embedding []float32
	
	// Filters is used for metadata filtering
	Filters map[string]interface{}
	
	// Limit is the maximum number of results to return
	Limit int
}

// LTMStore is the interface that all long-term memory store adapters must implement.
type LTMStore interface {
	// Store persists a memory record to the store.
	// It enforces entity isolation by storing the EntityID with the record.
	Store(ctx context.Context, record MemoryRecord) (string, error)
	
	// Retrieve fetches memory records matching the query.
	// It enforces entity isolation using the Context in ctx.
	Retrieve(ctx context.Context, query LTMQuery) ([]MemoryRecord, error)
	
	// Update modifies an existing memory record.
	// It enforces entity isolation, preventing updates to records from other entities.
	Update(ctx context.Context, record MemoryRecord) error
	
	// Delete removes a memory record.
	// It enforces entity isolation, preventing deletion of records from other entities.
	Delete(ctx context.Context, id string) error
}

// VectorCapableLTMStore extends the base LTMStore interface with vector capabilities.
// Adapters that implement this interface can perform vector-based semantic search operations.
type VectorCapableLTMStore interface {
	LTMStore
	
	// SupportsVectorSearch indicates that this LTM store supports vector operations.
	SupportsVectorSearch() bool
}
