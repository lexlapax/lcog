package mock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
)

// MockStore is an in-memory implementation of the LTMStore interface
// used for testing and development.
type MockStore struct {
	// Map of records indexed by entity ID and record ID
	// records[EntityID][RecordID] = MemoryRecord
	records map[entity.EntityID]map[string]ltm.MemoryRecord
	
	// Mutex for safe concurrent access
	mutex sync.RWMutex
}

// NewMockStore creates a new instance of the MockStore.
func NewMockStore() *MockStore {
	store := &MockStore{
		records: make(map[entity.EntityID]map[string]ltm.MemoryRecord),
	}
	
	log.Debug("Initialized LTM mock store adapter")
	return store
}

// Store implements the LTMStore interface.
func (m *MockStore) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "Missing entity context when storing memory record")
		return "", entity.ErrMissingEntityContext
	}
	
	// Fill in the entity ID if not already provided
	if record.EntityID == "" {
		record.EntityID = entityCtx.EntityID
		log.DebugContext(ctx, "Setting EntityID from context", "entity_id", entityCtx.EntityID)
	} else if record.EntityID != entityCtx.EntityID {
		// Validate that the record entity ID matches the context entity ID
		log.ErrorContext(ctx, "Record entity ID doesn't match context entity ID",
			"record_entity_id", record.EntityID,
			"context_entity_id", entityCtx.EntityID,
		)
		return "", fmt.Errorf("record entity ID must match context entity ID")
	}
	
	// Fill in user ID if available and not already provided
	if record.UserID == "" && entityCtx.UserID != "" {
		record.UserID = entityCtx.UserID
		log.DebugContext(ctx, "Setting UserID from context", "user_id", entityCtx.UserID)
	}
	
	// Generate a unique ID if not provided
	if record.ID == "" {
		record.ID = uuid.New().String()
		log.DebugContext(ctx, "Generated new record ID", "record_id", record.ID)
	}
	
	// Set timestamps
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now
	
	// Initialize metadata if nil
	if record.Metadata == nil {
		record.Metadata = make(map[string]interface{})
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Initialize entity records map if it doesn't exist
	if _, exists := m.records[record.EntityID]; !exists {
		log.DebugContext(ctx, "Creating new entity record map", "entity_id", record.EntityID)
		m.records[record.EntityID] = make(map[string]ltm.MemoryRecord)
	}
	
	// Store the record
	m.records[record.EntityID][record.ID] = record
	
	log.DebugContext(ctx, "Stored memory record in mock store", 
		"record_id", record.ID, 
		"entity_id", record.EntityID,
		"content_length", len(record.Content),
	)
	
	return record.ID, nil
}

// Retrieve implements the LTMStore interface.
func (m *MockStore) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "Missing entity context when retrieving memory records")
		return nil, entity.ErrMissingEntityContext
	}
	
	log.DebugContext(ctx, "Retrieving memory records", 
		"entity_id", entityCtx.EntityID,
		"query_text", query.Text,
		"has_exact_match", query.ExactMatch != nil,
		"limit", query.Limit,
	)
	
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Get all records for the entity
	entityRecords, exists := m.records[entityCtx.EntityID]
	if !exists {
		log.DebugContext(ctx, "No records found for entity", "entity_id", entityCtx.EntityID)
		return []ltm.MemoryRecord{}, nil
	}
	
	// Filter records based on query and access level
	var results []ltm.MemoryRecord
	
	// If limit is not specified or invalid, use a default
	limit := query.Limit
	if limit <= 0 {
		limit = 100 // Default limit
		log.DebugContext(ctx, "Using default limit for query", "default_limit", limit)
	}
	
	recordsScanned := 0
	recordsSkippedByAccess := 0
	recordsSkippedByQuery := 0
	
	for _, record := range entityRecords {
		recordsScanned++
		
		// First check access level
		if record.AccessLevel == entity.PrivateToUser {
			// If private, check if user ID matches
			if record.UserID != entityCtx.UserID {
				recordsSkippedByAccess++
				continue
			}
		}
		
		// Check if record matches the query
		if !m.recordMatchesQuery(record, query) {
			recordsSkippedByQuery++
			continue
		}
		
		// Add to results
		results = append(results, record)
		
		// Stop if we've reached the limit
		if len(results) >= limit {
			log.DebugContext(ctx, "Reached query limit", "limit", limit)
			break
		}
	}
	
	log.DebugContext(ctx, "Retrieved memory records", 
		"entity_id", entityCtx.EntityID,
		"result_count", len(results),
		"records_scanned", recordsScanned,
		"skipped_by_access", recordsSkippedByAccess,
		"skipped_by_query", recordsSkippedByQuery,
	)
	
	return results, nil
}

// Update implements the LTMStore interface.
func (m *MockStore) Update(ctx context.Context, record ltm.MemoryRecord) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}
	
	// Require ID
	if record.ID == "" {
		return errors.New("record ID is required for update")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if this entity has any records
	entityRecords, exists := m.records[entityCtx.EntityID]
	if !exists {
		return fmt.Errorf("record with ID %s not found", record.ID)
	}
	
	// Check if the record exists for this entity
	existingRecord, exists := entityRecords[record.ID]
	if !exists {
		return fmt.Errorf("record with ID %s not found", record.ID)
	}
	
	// Verify record belongs to the entity in the context
	if existingRecord.EntityID != entityCtx.EntityID {
		return errors.New("cannot update record belonging to another entity")
	}
	
	// Update timestamps
	record.CreatedAt = existingRecord.CreatedAt
	record.UpdatedAt = time.Now()
	
	// Ensure entity ID is preserved
	record.EntityID = entityCtx.EntityID
	
	// Update the record
	m.records[record.EntityID][record.ID] = record
	
	return nil
}

// Delete implements the LTMStore interface.
func (m *MockStore) Delete(ctx context.Context, id string) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if this entity has any records
	entityRecords, exists := m.records[entityCtx.EntityID]
	if !exists {
		return fmt.Errorf("record with ID %s not found", id)
	}
	
	// Check if the record exists for this entity
	record, exists := entityRecords[id]
	if !exists {
		return fmt.Errorf("record with ID %s not found", id)
	}
	
	// Verify record belongs to the entity in the context
	if record.EntityID != entityCtx.EntityID {
		return errors.New("cannot delete record belonging to another entity")
	}
	
	// Delete the record
	delete(m.records[entityCtx.EntityID], id)
	
	return nil
}

// recordMatchesQuery checks if a record matches the given query parameters.
func (m *MockStore) recordMatchesQuery(record ltm.MemoryRecord, query ltm.LTMQuery) bool {
	// Check exact match conditions
	if query.ExactMatch != nil {
		for key, value := range query.ExactMatch {
			// Special case for ID field
			if key == "ID" {
				if record.ID != value {
					return false
				}
				continue
			}
			
			// Handle metadata exact matches
			if record.Metadata != nil {
				if metaValue, exists := record.Metadata[key]; exists {
					if metaValue != value {
						return false
					}
				} else {
					return false
				}
			} else {
				return false
			}
		}
	}
	
	// Check text match
	if query.Text != "" && !strings.Contains(strings.ToLower(record.Content), strings.ToLower(query.Text)) {
		return false
	}
	
	// Check metadata filters
	if query.Filters != nil && len(query.Filters) > 0 {
		if record.Metadata == nil {
			return false
		}
		
		for key, value := range query.Filters {
			metaValue, exists := record.Metadata[key]
			if !exists {
				return false
			}
			
			// Simple equality check for now
			// Could be enhanced for more complex filtering
			if metaValue != value {
				return false
			}
		}
	}
	
	// If all checks pass or no checks were performed, the record matches
	return true
}
