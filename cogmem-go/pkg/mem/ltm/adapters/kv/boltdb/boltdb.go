package boltdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	bolt "go.etcd.io/bbolt"
)

// BoltStore implements the LTMStore interface using a BoltDB database.
type BoltStore struct {
	db *bolt.DB
}

// NewBoltStore creates a new BoltStore with the given database connection.
func NewBoltStore(db *bolt.DB) *BoltStore {
	store := &BoltStore{
		db: db,
	}
	
	log.Debug("Initialized BoltDB LTM store adapter", 
		"db_path", db.Path(),
		"read_only", db.IsReadOnly(),
	)
	
	return store
}

// Initialize creates the required buckets if they don't exist.
// This is called internally by the Store method, but can be called
// explicitly to ensure buckets are created at startup.
func (b *BoltStore) Initialize(ctx context.Context) error {
	log.DebugContext(ctx, "Initializing BoltDB store buckets")
	
	err := b.db.Update(func(tx *bolt.Tx) error {
		// Create the main bucket to hold entity buckets
		_, err := tx.CreateBucketIfNotExists([]byte("entities"))
		return err
	})
	
	if err != nil {
		log.ErrorContext(ctx, "Failed to initialize BoltDB buckets", "error", err)
		return err
	}
	
	log.DebugContext(ctx, "Successfully initialized BoltDB store buckets")
	return nil
}

// getEntityBucket gets or creates a bucket for the specified entity.
func (b *BoltStore) getEntityBucket(tx *bolt.Tx, entityID entity.EntityID) (*bolt.Bucket, error) {
	// Get the main entities bucket
	entities, err := tx.CreateBucketIfNotExists([]byte("entities"))
	if err != nil {
		return nil, fmt.Errorf("failed to create entities bucket: %w", err)
	}

	// Get or create a bucket for this entity
	entityBucket, err := entities.CreateBucketIfNotExists([]byte(entityID))
	if err != nil {
		return nil, fmt.Errorf("failed to create entity bucket for %s: %w", entityID, err)
	}

	return entityBucket, nil
}

// Store persists a memory record to the BoltDB database.
func (b *BoltStore) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return "", entity.ErrMissingEntityContext
	}

	// Fill in the entity ID if not already provided
	if record.EntityID == "" {
		record.EntityID = entityCtx.EntityID
	} else if record.EntityID != entityCtx.EntityID {
		// Validate that the record entity ID matches the context entity ID
		return "", fmt.Errorf("record entity ID must match context entity ID")
	}

	// Fill in user ID if available and not already provided
	if record.UserID == "" && entityCtx.UserID != "" {
		record.UserID = entityCtx.UserID
	}

	// Generate a unique ID if not provided
	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	// Store the record in a transaction
	err := b.db.Update(func(tx *bolt.Tx) error {
		// Get the entity bucket
		entityBucket, err := b.getEntityBucket(tx, entityCtx.EntityID)
		if err != nil {
			return err
		}

		// Marshal the record to JSON
		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to marshal record: %w", err)
		}

		// Store the record
		return entityBucket.Put([]byte(record.ID), data)
	})

	if err != nil {
		return "", fmt.Errorf("failed to store record: %w", err)
	}

	return record.ID, nil
}

// Retrieve fetches memory records matching the query from the BoltDB database.
func (b *BoltStore) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	var records []ltm.MemoryRecord

	// Retrieve records in a read-only transaction
	err := b.db.View(func(tx *bolt.Tx) error {
		// Get the entities bucket
		entities := tx.Bucket([]byte("entities"))
		if entities == nil {
			// No entities bucket exists yet, return empty result
			return nil
		}

		// Get the entity bucket
		entityBucket := entities.Bucket([]byte(entityCtx.EntityID))
		if entityBucket == nil {
			// No bucket for this entity, return empty result
			return nil
		}

		// If querying for a specific ID
		if query.ExactMatch != nil {
			if id, ok := query.ExactMatch["ID"]; ok {
				idStr, ok := id.(string)
				if !ok {
					return fmt.Errorf("ID must be a string")
				}

				// Get the record by ID
				data := entityBucket.Get([]byte(idStr))
				if data == nil {
					// Record not found
					return nil
				}

				// Unmarshal the record
				var record ltm.MemoryRecord
				if err := json.Unmarshal(data, &record); err != nil {
					return fmt.Errorf("failed to unmarshal record: %w", err)
				}

				// Apply access level filtering
				if isAccessible(record, entityCtx) {
					records = append(records, record)
				}
				return nil
			}
		}

		// Full scan with filtering
		limit := 100
		if query.Limit > 0 {
			limit = query.Limit
		}

		// Create a slice to store all records before filtering and sorting
		var allRecords []ltm.MemoryRecord

		// Iterate through all records in the entity bucket
		err := entityBucket.ForEach(func(k, v []byte) error {
			var record ltm.MemoryRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return fmt.Errorf("failed to unmarshal record: %w", err)
			}

			// Apply access level filtering
			if !isAccessible(record, entityCtx) {
				return nil
			}

			// Text search filtering
			if query.Text != "" && !containsText(record.Content, query.Text) {
				return nil
			}

			// Apply metadata filtering
			if query.Filters != nil && len(query.Filters) > 0 {
				if !matchesFilters(record, query.Filters) {
					return nil
				}
			}

			allRecords = append(allRecords, record)
			return nil
		})

		if err != nil {
			return err
		}

		// Sort records by created_at (newest first)
		sortRecordsByCreatedAt(allRecords)

		// Apply limit
		recordCount := len(allRecords)
		if recordCount > limit {
			records = allRecords[:limit]
		} else {
			records = allRecords
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve records: %w", err)
	}

	return records, nil
}

// Update modifies an existing memory record in the BoltDB database.
func (b *BoltStore) Update(ctx context.Context, record ltm.MemoryRecord) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}

	// Require ID
	if record.ID == "" {
		return errors.New("record ID is required for update")
	}

	// Update the record in a transaction
	var recordExists bool
	err := b.db.Update(func(tx *bolt.Tx) error {
		// Get the entities bucket
		entities := tx.Bucket([]byte("entities"))
		if entities == nil {
			return fmt.Errorf("entities bucket does not exist")
		}

		// Get the entity bucket
		entityBucket := entities.Bucket([]byte(entityCtx.EntityID))
		if entityBucket == nil {
			return fmt.Errorf("entity bucket does not exist for %s", entityCtx.EntityID)
		}

		// Get the existing record
		data := entityBucket.Get([]byte(record.ID))
		if data == nil {
			// Record not found
			return nil
		}

		// Unmarshal the existing record
		var existingRecord ltm.MemoryRecord
		if err := json.Unmarshal(data, &existingRecord); err != nil {
			return fmt.Errorf("failed to unmarshal existing record: %w", err)
		}

		// Ensure the record belongs to the correct entity
		if existingRecord.EntityID != entityCtx.EntityID {
			return fmt.Errorf("record belongs to another entity")
		}

		recordExists = true

		// Update the record fields
		existingRecord.Content = record.Content
		existingRecord.Metadata = record.Metadata
		existingRecord.UpdatedAt = time.Now().UTC()

		// Marshal the updated record
		updatedData, err := json.Marshal(existingRecord)
		if err != nil {
			return fmt.Errorf("failed to marshal updated record: %w", err)
		}

		// Store the updated record
		return entityBucket.Put([]byte(record.ID), updatedData)
	})

	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	if !recordExists {
		return fmt.Errorf("record with ID %s not found or belongs to another entity", record.ID)
	}

	return nil
}

// Delete removes a memory record from the BoltDB database.
func (b *BoltStore) Delete(ctx context.Context, id string) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}

	// Delete the record in a transaction
	var recordExists bool
	err := b.db.Update(func(tx *bolt.Tx) error {
		// Get the entities bucket
		entities := tx.Bucket([]byte("entities"))
		if entities == nil {
			return nil // Nothing to delete
		}

		// Get the entity bucket
		entityBucket := entities.Bucket([]byte(entityCtx.EntityID))
		if entityBucket == nil {
			return nil // Nothing to delete
		}

		// Check if the record exists
		data := entityBucket.Get([]byte(id))
		if data == nil {
			return nil // Nothing to delete
		}

		// Unmarshal to verify entity ownership
		var record ltm.MemoryRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return fmt.Errorf("failed to unmarshal record: %w", err)
		}

		// Ensure the record belongs to the correct entity
		if record.EntityID != entityCtx.EntityID {
			return fmt.Errorf("record belongs to another entity")
		}

		recordExists = true

		// Delete the record
		return entityBucket.Delete([]byte(id))
	})

	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	if !recordExists {
		return fmt.Errorf("record with ID %s not found or belongs to another entity", id)
	}

	return nil
}

// Helper functions

// isAccessible checks if a record is accessible given the entity context.
func isAccessible(record ltm.MemoryRecord, entityCtx entity.Context) bool {
	// Record must belong to the entity in context
	if record.EntityID != entityCtx.EntityID {
		return false
	}

	// Check access level
	switch record.AccessLevel {
	case entity.SharedWithinEntity:
		// Shared records are accessible to all within the entity
		return true
	case entity.PrivateToUser:
		// Private records are only accessible to the user who created them
		return entityCtx.UserID != "" && record.UserID == entityCtx.UserID
	default:
		// Unsupported access level
		return false
	}
}

// containsText checks if the content contains the search text.
func containsText(content, search string) bool {
	// Simple substring search (case-insensitive)
	// For more advanced search, consider using a proper text search library
	return len(search) == 0 || containsSubstring(content, search)
}

// containsSubstring checks if a string contains a substring.
func containsSubstring(s, substr string) bool {
	// Use case-insensitive search for testing purposes
	return strings.Contains(
		strings.ToLower(s),
		strings.ToLower(substr),
	)
}

// matchesFilters checks if a record's metadata matches the provided filters.
func matchesFilters(record ltm.MemoryRecord, filters map[string]interface{}) bool {
	if record.Metadata == nil {
		return false
	}

	for key, value := range filters {
		metaValue, exists := record.Metadata[key]
		if !exists {
			return false
		}

		// Simple equality check
		if metaValue != value {
			return false
		}
	}

	return true
}

// sortRecordsByCreatedAt sorts records by created_at in descending order (newest first).
func sortRecordsByCreatedAt(records []ltm.MemoryRecord) {
	// Sort records by created_at (newest first)
	// Using a simple bubble sort for simplicity
	// For production code, consider using sort.Sort with a custom interface
	n := len(records)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if records[j].CreatedAt.Before(records[j+1].CreatedAt) {
				records[j], records[j+1] = records[j+1], records[j]
			}
		}
	}
}