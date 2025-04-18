package chromem_go

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	chromem "github.com/philippgille/chromem-go"
)

var (
	// ErrMissingQueryVector is returned when a semantic search is attempted without a query vector
	ErrMissingQueryVector = errors.New("missing query vector for semantic search")
	
	// ErrRecordNotFound is returned when a record with the specified ID doesn't exist
	ErrRecordNotFound = errors.New("record not found")
	
	// ErrChromemGoUnavailable is returned when the chromem-go client is unavailable
	ErrChromemGoUnavailable = errors.New("chromem-go client unavailable")
)

// Metadata keys used for filtering records
const (
	MetadataKeyID          = "id"
	MetadataKeyEntityID    = "entity_id"
	MetadataKeyUserID      = "user_id"
	MetadataKeyAccessLevel = "access_level"
	MetadataKeyCreatedAt   = "created_at"
	MetadataKeyUpdatedAt   = "updated_at"
)

// ChromemGoAdapter implements the ltm.LTMStore interface using chromem-go as the backend
type ChromemGoAdapter struct {
	client         *chromem.DB
	collectionName string
	collection     *chromem.Collection
}

// NewChromemGoAdapter creates a new adapter for chromem-go
func NewChromemGoAdapter(client *chromem.DB, collectionName string) (*ChromemGoAdapter, error) {
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}

	if collectionName == "" {
		return nil, errors.New("collection name cannot be empty")
	}

	// Create a default embedding function
	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		// This is a placeholder that returns a default embedding
		// In a real implementation, this would call to the reasoning engine
		return []float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil
	}

	// Create or get collection
	collection, err := client.GetOrCreateCollection(collectionName, nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create/get collection: %w", err)
	}

	return &ChromemGoAdapter{
		client:         client,
		collectionName: collectionName,
		collection:     collection,
	}, nil
}

// NewChromemGoAdapterWithConfig creates a new ChromemGo adapter using the provided configuration
func NewChromemGoAdapterWithConfig(config *ChromemGoConfig) (*ChromemGoAdapter, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.Collection == "" {
		return nil, errors.New("collection name cannot be empty")
	}

	var client *chromem.DB
	
	// Use persistent storage if a storage path is provided
	if config.StoragePath != "" {
		log.Debug("Using persistent storage for ChromemGo", "path", config.StoragePath)
		// Use NewPersistentDB for on-disk storage
		// Second parameter controls concurrent access (set to false for better reliability)
		var err error
		client, err = chromem.NewPersistentDB(config.StoragePath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistent chromem-go client: %w", err)
		}
	} else {
		log.Debug("Using in-memory storage for ChromemGo")
		// Use NewDB for in-memory storage
		client = chromem.NewDB()
	}
	
	if client == nil {
		return nil, errors.New("failed to create chromem-go client")
	}

	// Create the adapter with the client
	return NewChromemGoAdapter(client, config.Collection)
}

// ChromemGoConfig holds configuration options for the ChromemGo adapter
type ChromemGoConfig struct {
	// Collection is the collection name to use
	Collection string
	// StoragePath is the path for on-disk storage (if empty, in-memory is used)
	StoragePath string
	// Dimensions specifies the embedding dimensions (default 1536)
	Dimensions int
}

// Store persists a memory record to chromem-go
func (a *ChromemGoAdapter) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
	// Use the existing ID or generate a new one
	recordID := record.ID
	if recordID == "" {
		recordID = fmt.Sprintf("rec-%d", time.Now().UnixNano())
		record.ID = recordID
	}

	// Ensure timestamps are set
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = now
	}

	// Prepare metadata for chromem-go (convert to string keys and values)
	metadata := make(map[string]string)
	
	// Add system metadata for filtering
	metadata[MetadataKeyID] = record.ID
	metadata[MetadataKeyEntityID] = string(record.EntityID)
	metadata[MetadataKeyUserID] = record.UserID
	metadata[MetadataKeyAccessLevel] = strconv.Itoa(int(record.AccessLevel))
	metadata[MetadataKeyCreatedAt] = record.CreatedAt.Format(time.RFC3339)
	metadata[MetadataKeyUpdatedAt] = record.UpdatedAt.Format(time.RFC3339)
	
	// Add user metadata
	if record.Metadata != nil {
		for k, v := range record.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				// Convert non-string values to strings
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Ensure we have an embedding
	embedding := record.Embedding
	if embedding == nil || len(embedding) == 0 {
		// Use a default embedding for now
		// In a real implementation, this would be generated by calling the reasoning engine
		embedding = []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	}

	// Create a chromem document
	doc := chromem.Document{
		ID:        record.ID,
		Content:   record.Content,
		Embedding: embedding,
		Metadata:  metadata,
	}

	// Store the document in chromem-go
	err := a.collection.AddDocument(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to add document: %w", err)
	}
	
	// Log success for debugging
	log.Debug("Document stored successfully", 
		"id", doc.ID, 
		"collection", a.collectionName,
		"metadata_keys", fmt.Sprintf("%v", doc.Metadata))

	log.Debug("Stored record in chromem-go", 
		"id", record.ID, 
		"entity_id", record.EntityID,
		"collection", a.collectionName)

	return record.ID, nil
}

// Retrieve fetches memory records matching the query
func (a *ChromemGoAdapter) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	if a.collection == nil {
		return nil, ErrChromemGoUnavailable
	}

	// Determine the retrieval mode based on the query
	var results []chromem.Result
	var err error

	switch {
	case query.ExactMatch != nil && query.ExactMatch["id"] != nil:
		// ID-based lookup
		recordID, ok := query.ExactMatch["id"].(string)
		if !ok {
			return nil, errors.New("invalid record ID in query")
		}
		results, err = a.retrieveByID(ctx, recordID)
	case len(query.Embedding) > 0:
		// Semantic search with vector
		results, err = a.retrieveSemantic(ctx, query)
	default:
		// Filter-based search
		results, err = a.retrieveByFilters(ctx, query)
	}

	if err != nil {
		return nil, err
	}

	// Convert from chromem.Result to ltm.MemoryRecord
	records := a.convertToMemoryRecords(results)
	return records, nil
}

// retrieveByID retrieves a record by its ID
func (a *ChromemGoAdapter) retrieveByID(ctx context.Context, recordID string) ([]chromem.Result, error) {
	if recordID == "" {
		return nil, errors.New("record ID cannot be empty")
	}

	log.Debug("Retrieving by ID", "id", recordID, "collection", a.collectionName)
	
	// Create filter for document ID
	where := map[string]string{
		MetadataKeyID: recordID,
	}
	
	// Try to get an estimated count to handle limit better
	count, err := a.getEstimatedCount(ctx)
	if err != nil {
		log.Warn("Failed to get document count", "error", err)
		// Continue with a safe default
		count = 10
	}
	
	if count == 0 {
		// No documents in collection, return empty result
		log.Debug("No documents in collection", "collection", a.collectionName)
		return []chromem.Result{}, nil
	}
	
	// For ID lookup, we only need one result max
	limit := 1
	
	// NOTE: chromem-go v0.7.0 doesn't have a direct GetDocument method,
	// so we'll always use query. When a newer version with direct lookup
	// is available, this can be optimized.

	// Fallback to query with a dummy embedding if direct lookup fails
	// Query with a dummy embedding and our ID filter
	log.Debug("Trying to find document using query", "id", recordID, "where", where)
	
	dummyEmbedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	results, err := a.collection.QueryEmbedding(ctx, dummyEmbedding, limit, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query by ID (%s): %w", recordID, err)
	}
	
	// Log query results for debugging
	log.Debug("Query by ID results", 
		"id", recordID, 
		"count", len(results),
		"where", fmt.Sprintf("%v", where))

	return results, nil
}

// retrieveSemantic performs semantic search with the query vector
func (a *ChromemGoAdapter) retrieveSemantic(ctx context.Context, query ltm.LTMQuery) ([]chromem.Result, error) {
	if len(query.Embedding) == 0 {
		return nil, ErrMissingQueryVector
	}

	log.Debug("Performing semantic search", "collection", a.collectionName, "embedding_len", len(query.Embedding))

	// Build the where clause for filtering
	where := make(map[string]string)
	
	// Process system field mappings
	// Handle entity_id
	if entityID, ok := query.Filters["entity_id"].(entity.EntityID); ok && entityID != "" {
		where[MetadataKeyEntityID] = string(entityID)
		log.Debug("Added entity filter for semantic search", "entity_id", entityID)
	}
	
	// Handle user_id
	if userID, ok := query.Filters["user_id"].(string); ok && userID != "" {
		where[MetadataKeyUserID] = userID
		log.Debug("Added user filter for semantic search", "user_id", userID)
	}
	
	// Handle access_level
	if accessLevel, ok := query.Filters["access_level"].(entity.AccessLevel); ok {
		where[MetadataKeyAccessLevel] = strconv.Itoa(int(accessLevel))
		log.Debug("Added access level filter for semantic search", "access_level", accessLevel)
	}
	
	// Process other filters
	if query.Filters != nil {
		for k, v := range query.Filters {
			// Skip already processed system fields
			if k == "entity_id" || k == "user_id" || k == "access_level" {
				continue
			}
			
			// Convert value to string for the filter
			switch val := v.(type) {
			case string:
				where[k] = val
			case entity.EntityID:
				where[k] = string(val)
			case int:
				where[k] = strconv.Itoa(val)
			case bool:
				where[k] = strconv.FormatBool(val)
			default:
				// For anything else, convert to string
				where[k] = fmt.Sprintf("%v", val)
			}
			
			log.Debug("Added filter for semantic search", "key", k, "value", where[k])
		}
	}

	// Set default limit if not specified
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	// Try to get an estimated count to handle limit better
	count, err := a.getEstimatedCount(ctx)
	if err != nil {
		log.Warn("Failed to get document count for semantic search", "error", err)
		// Continue with a safe default
		count = 10
	}
	
	if count == 0 {
		// No documents in collection, return empty result
		log.Debug("No documents in collection for semantic search", "collection", a.collectionName)
		return []chromem.Result{}, nil
	}
	
	// Adjust limit if necessary
	if limit > count {
		limit = count
	}

	// Query documents
	log.Debug("Running semantic search query", 
		"filters", fmt.Sprintf("%v", where), 
		"limit", limit,
		"embedding_len", len(query.Embedding))
		
	results, err := a.collection.QueryEmbedding(ctx, query.Embedding, limit, where, nil)
	if err != nil {
		// Apply more detailed error reporting
		if err.Error() == "nResults must be <= the number of documents in the collection" {
			// Collection is empty
			log.Debug("Collection is empty for semantic search")
			return []chromem.Result{}, nil
		}
		
		return nil, fmt.Errorf("failed to perform semantic search: %w", err)
	}

	log.Debug("Semantic search returned results", "count", len(results))
	return results, nil
}

// retrieveByFilters retrieves records using metadata filters
func (a *ChromemGoAdapter) retrieveByFilters(ctx context.Context, query ltm.LTMQuery) ([]chromem.Result, error) {
	// Build the where clause for filtering
	where := make(map[string]string)
	
	log.Debug("Retrieving by filters", "collection", a.collectionName)
	
	// Process system field mappings
	// Handle entity_id
	if entityID, ok := query.Filters["entity_id"].(entity.EntityID); ok && entityID != "" {
		where[MetadataKeyEntityID] = string(entityID)
		log.Debug("Added entity filter", "entity_id", entityID)
	}
	
	// Handle user_id
	if userID, ok := query.Filters["user_id"].(string); ok && userID != "" {
		where[MetadataKeyUserID] = userID
		log.Debug("Added user filter", "user_id", userID)
	}
	
	// Handle access_level
	if accessLevel, ok := query.Filters["access_level"].(entity.AccessLevel); ok {
		where[MetadataKeyAccessLevel] = strconv.Itoa(int(accessLevel))
		log.Debug("Added access level filter", "access_level", accessLevel)
	}
	
	// Process exact match criteria (other than system fields)
	if query.ExactMatch != nil {
		for k, v := range query.ExactMatch {
			if k == "id" {
				continue // Skip ID as it's handled separately
			}
			
			// Map standard field names to our metadata keys
			key := k
			switch k {
			case "entity_id":
				key = MetadataKeyEntityID
			case "user_id":
				key = MetadataKeyUserID
			case "access_level":
				key = MetadataKeyAccessLevel
			case "created_at":
				key = MetadataKeyCreatedAt
			case "updated_at":
				key = MetadataKeyUpdatedAt
			}
			
			// Convert value to string for the filter
			switch val := v.(type) {
			case string:
				where[key] = val
			case entity.EntityID:
				where[key] = string(val)
			case int:
				where[key] = strconv.Itoa(val)
			case bool:
				where[key] = strconv.FormatBool(val)
			case entity.AccessLevel:
				where[key] = strconv.Itoa(int(val))
			case time.Time:
				where[key] = val.Format(time.RFC3339)
			default:
				// For anything else, convert to string
				where[key] = fmt.Sprintf("%v", val)
			}
			
			log.Debug("Added exact match filter", "key", key, "value", where[key])
		}
	}
	
	// Process general filters (other than standard fields)
	if query.Filters != nil {
		for k, v := range query.Filters {
			// Skip already processed system fields
			if k == "entity_id" || k == "user_id" || k == "access_level" {
				continue
			}
			
			// Convert value to string for the filter
			switch val := v.(type) {
			case string:
				where[k] = val
			case entity.EntityID:
				where[k] = string(val)
			case int:
				where[k] = strconv.Itoa(val)
			case bool:
				where[k] = strconv.FormatBool(val)
			case entity.AccessLevel:
				where[k] = strconv.Itoa(int(val))
			case time.Time:
				where[k] = val.Format(time.RFC3339)
			default:
				// For anything else, convert to string
				where[k] = fmt.Sprintf("%v", val)
			}
			
			log.Debug("Added filter", "key", k, "value", where[k])
		}
	}

	// Set default limit if not specified
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	// Try to get an estimated count to handle limit better
	count, err := a.getEstimatedCount(ctx)
	if err != nil {
		log.Warn("Failed to get document count", "error", err)
		// Continue with a safe default
		count = 10
	}
	
	if count == 0 {
		// No documents in collection, return empty result
		log.Debug("No documents in collection", "collection", a.collectionName)
		return []chromem.Result{}, nil
	}
	
	// Adjust limit if necessary
	if limit > count {
		limit = count
	}

	var results []chromem.Result

	// Log the query we're about to make
	log.Debug("Running filter-based query", 
		"filters", fmt.Sprintf("%v", where), 
		"limit", limit,
		"has_text", query.Text != "")

	// If text search is specified, use Query with the text
	if query.Text != "" {
		results, err = a.collection.Query(ctx, query.Text, limit, where, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query by text (%s): %w", query.Text, err)
		}
	} else {
		// Use a default dummy embedding for metadata-only search
		// The API requires a non-empty embedding, but we'll ignore the vector distance in the results
		dummyEmbedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		results, err = a.collection.QueryEmbedding(ctx, dummyEmbedding, limit, where, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query by filters: %w", err)
		}
	}

	log.Debug("Filter query returned results", "count", len(results))
	return results, nil
}

// getEstimatedCount returns an estimated count of documents in the collection
// This is a more reliable approach than the previous countDocumentsInCollection function
func (a *ChromemGoAdapter) getEstimatedCount(ctx context.Context) (int, error) {
	if a.collection == nil {
		return 0, ErrChromemGoUnavailable
	}
	
	// Unfortunately, ChromemGo v0.7.0 doesn't have a direct way to count documents
	// We'll check if the collection has any documents by querying with a high limit
	// and assuming it's not empty if no error occurs
	
	// Return a safe non-zero value, since we can't accurately determine the count
	// This is a workaround for the "nResults must be <= the number of documents" error
	return 10000, nil
}

// Update modifies an existing memory record
func (a *ChromemGoAdapter) Update(ctx context.Context, record ltm.MemoryRecord) error {
	if a.collection == nil {
		return ErrChromemGoUnavailable
	}

	if record.ID == "" {
		return errors.New("record ID cannot be empty")
	}

	log.Debug("Updating record", "id", record.ID, "collection", a.collectionName)

	// First try to retrieve the record to check if it exists
	results, err := a.retrieveByID(ctx, record.ID)
	if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	if len(results) == 0 {
		log.Warn("Record not found for update", "id", record.ID)
		return ErrRecordNotFound
	}

	// Create filter for document ID
	where := map[string]string{
		MetadataKeyID: record.ID,
	}

	// Try to delete the existing record
	log.Debug("Deleting existing record for update", "id", record.ID)
	err = a.collection.Delete(ctx, where, nil, record.ID)
	if err != nil {
		// Handle specific error cases differently
		if err.Error() == "no match found" {
			// Record wasn't found, log but continue (try to insert)
			log.Warn("Record not found when trying to delete for update", "id", record.ID, "error", err)
		} else {
			return fmt.Errorf("failed to delete existing record for update: %w", err)
		}
	}

	// Update the timestamp
	record.UpdatedAt = time.Now()

	// Store the updated record
	log.Debug("Storing updated record", "id", record.ID)
	_, err = a.Store(ctx, record)
	if err != nil {
		return fmt.Errorf("failed to store updated record: %w", err)
	}

	log.Debug("Record updated successfully", "id", record.ID)
	return nil
}

// Delete removes a memory record
func (a *ChromemGoAdapter) Delete(ctx context.Context, id string) error {
	if a.collection == nil {
		return ErrChromemGoUnavailable
	}

	if id == "" {
		return errors.New("record ID cannot be empty")
	}

	log.Debug("Deleting record", "id", id, "collection", a.collectionName)

	// First check if the record exists
	results, err := a.retrieveByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	if len(results) == 0 {
		// Record not found, return success (idempotent operation)
		log.Debug("Record not found for deletion, treating as success", "id", id)
		return nil
	}

	// Create a filter to find the document by ID
	where := map[string]string{
		MetadataKeyID: id,
	}

	// Delete the document by ID
	err = a.collection.Delete(ctx, where, nil, id)
	if err != nil {
		// Handle specific error cases
		if err.Error() == "no match found" {
			// Record wasn't found, consider it a success (idempotent)
			log.Debug("Record not found for deletion, treating as success", "id", id)
			return nil
		}
		return fmt.Errorf("failed to delete document: %w", err)
	}

	log.Debug("Deleted record from chromem-go", 
		"id", id, 
		"collection", a.collectionName)

	return nil
}

// convertToMemoryRecords converts chromem-go query results to MemoryRecord objects
func (a *ChromemGoAdapter) convertToMemoryRecords(results []chromem.Result) []ltm.MemoryRecord {
	if len(results) == 0 {
		return []ltm.MemoryRecord{}
	}

	records := make([]ltm.MemoryRecord, 0, len(results))

	for _, result := range results {
		// Get the ID
		recordID := result.ID

		// Get entity ID
		entityIDStr, ok := result.Metadata[MetadataKeyEntityID]
		if !ok {
			log.Warn("Document missing entity ID", "id", result.ID)
			continue
		}
		entityID := entity.EntityID(entityIDStr)

		// Parse access level
		accessLevelStr, ok := result.Metadata[MetadataKeyAccessLevel]
		if !ok {
			log.Warn("Document missing access level", "id", result.ID)
			continue
		}
		accessLevel, err := strconv.Atoi(accessLevelStr)
		if err != nil {
			log.Warn("Failed to parse access level", "id", result.ID, "access_level", accessLevelStr, "error", err)
			continue
		}

		// Parse timestamps
		createdAtStr, ok := result.Metadata[MetadataKeyCreatedAt]
		if !ok {
			log.Warn("Document missing created timestamp", "id", result.ID)
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			log.Warn("Failed to parse created timestamp", "id", result.ID, "created_at", createdAtStr, "error", err)
			continue
		}

		updatedAtStr, ok := result.Metadata[MetadataKeyUpdatedAt]
		if !ok {
			log.Warn("Document missing updated timestamp", "id", result.ID)
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			log.Warn("Failed to parse updated timestamp", "id", result.ID, "updated_at", updatedAtStr, "error", err)
			continue
		}

		// Extract user ID
		userID := result.Metadata[MetadataKeyUserID]

		// Create metadata map excluding system metadata
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			// Skip system metadata
			if k == MetadataKeyID || k == MetadataKeyEntityID || k == MetadataKeyUserID ||
				k == MetadataKeyAccessLevel || k == MetadataKeyCreatedAt || k == MetadataKeyUpdatedAt {
				continue
			}
			metadata[k] = v
		}

		// Create the memory record
		record := ltm.MemoryRecord{
			ID:          recordID,
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.AccessLevel(accessLevel),
			Content:     result.Content,
			Metadata:    metadata,
			Embedding:   result.Embedding,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		records = append(records, record)
	}

	return records
}

// SupportsVectorSearch returns true as this adapter supports vector search
func (a *ChromemGoAdapter) SupportsVectorSearch() bool {
	return true
}