package pgvector

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
)

var (
	// ErrMissingQueryVector is returned when a semantic search is attempted without a query vector
	ErrMissingQueryVector = errors.New("missing query vector for semantic search")
	
	// ErrRecordNotFound is returned when a record with the specified ID doesn't exist
	ErrRecordNotFound = errors.New("record not found")
	
	// ErrPgvectorUnavailable is returned when the pgvector client is unavailable
	ErrPgvectorUnavailable = errors.New("pgvector client unavailable")
)

// PgvectorAdapter implements the ltm.VectorCapableLTMStore interface using PostgreSQL with pgvector extension
type PgvectorAdapter struct {
	db            *pgxpool.Pool
	tableName     string
	dimensionSize int
	// Distance metric: cosine (default), euclidean, dot
	distanceMetric string 
}

// DB returns the underlying database connection pool (used for testing)
func (a *PgvectorAdapter) DB() *pgxpool.Pool {
	return a.db
}

// PgvectorConfig contains the configuration for a Pgvector adapter
type PgvectorConfig struct {
	// ConnectionString is the PostgreSQL connection string
	ConnectionString string
	
	// TableName is the name of the table to use
	TableName string
	
	// DimensionSize is the size of vector embeddings
	DimensionSize int
	
	// DistanceMetric is the distance metric to use (cosine, euclidean, dot)
	DistanceMetric string
}

// NewPgvectorAdapter creates a new adapter for PostgreSQL with pgvector extension
func NewPgvectorAdapter(ctx context.Context, config PgvectorConfig) (*PgvectorAdapter, error) {
	if config.ConnectionString == "" {
		return nil, errors.New("connection string cannot be empty")
	}

	if config.TableName == "" {
		config.TableName = "memory_vectors"
	}

	if config.DimensionSize <= 0 {
		config.DimensionSize = 1536 // Default dimension size for OpenAI embeddings
	}

	// Default to cosine similarity if not specified
	if config.DistanceMetric == "" {
		config.DistanceMetric = "cosine"
	} else {
		config.DistanceMetric = strings.ToLower(config.DistanceMetric)
		if config.DistanceMetric != "cosine" && config.DistanceMetric != "euclidean" && config.DistanceMetric != "dot" {
			return nil, fmt.Errorf("unsupported distance metric: %s (must be cosine, euclidean, or dot)", config.DistanceMetric)
		}
	}

	// Connect to PostgreSQL
	db, err := pgxpool.New(ctx, config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Check connection
	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Create adapter
	adapter := &PgvectorAdapter{
		db:            db,
		tableName:     config.TableName,
		dimensionSize: config.DimensionSize,
		distanceMetric: config.DistanceMetric,
	}

	// Initialize table
	if err := adapter.initializeTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize pgvector table: %w", err)
	}

	return adapter, nil
}

// initializeTable creates the necessary table and index for vector storage if they don't exist
func (a *PgvectorAdapter) initializeTable(ctx context.Context) error {
	// Check if pgvector extension is installed
	var extensionExists bool
	err := a.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&extensionExists)
	if err != nil {
		return fmt.Errorf("failed to check for pgvector extension: %w", err)
	}

	if !extensionExists {
		// Try to create the extension
		_, err = a.db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
		if err != nil {
			return fmt.Errorf("failed to create pgvector extension: %w", err)
		}
		log.Info("Created pgvector extension")
	}

	// Create the table if it doesn't exist
	_, err = a.db.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			entity_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			access_level INTEGER NOT NULL,
			content TEXT NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}',
			embedding VECTOR(%d) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL
		)
	`, a.tableName, a.dimensionSize))
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create indices for efficient querying
	indices := []struct {
		name string
		sql  string
	}{
		{
			name: "idx_entity_id",
			sql:  fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_entity_id_idx ON %s (entity_id)", a.tableName, a.tableName),
		},
		{
			name: "idx_user_id",
			sql:  fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_user_id_idx ON %s (user_id)", a.tableName, a.tableName),
		},
		{
			name: "idx_access_level",
			sql:  fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_access_level_idx ON %s (access_level)", a.tableName, a.tableName),
		},
		{
			name: "idx_created_at",
			sql:  fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_created_at_idx ON %s (created_at)", a.tableName, a.tableName),
		},
		{
			name: "idx_updated_at",
			sql:  fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_updated_at_idx ON %s (updated_at)", a.tableName, a.tableName),
		},
	}

	// Create the vector index based on the configured distance metric
	var vectorIndexSQL string
	switch a.distanceMetric {
	case "cosine":
		vectorIndexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)", a.tableName, a.tableName)
	case "euclidean":
		vectorIndexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_l2_ops) WITH (lists = 100)", a.tableName, a.tableName)
	case "dot":
		vectorIndexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_ip_ops) WITH (lists = 100)", a.tableName, a.tableName)
	}

	indices = append(indices, struct {
		name string
		sql  string
	}{
		name: "idx_embedding",
		sql:  vectorIndexSQL,
	})

	// Create each index
	for _, idx := range indices {
		_, err = a.db.Exec(ctx, idx.sql)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	return nil
}

// Close closes the database connection pool
func (a *PgvectorAdapter) Close() {
	if a.db != nil {
		a.db.Close()
	}
}

// Store persists a memory record to PostgreSQL with vector embedding
func (a *PgvectorAdapter) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
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

	// Check if we have an embedding
	if record.Embedding == nil || len(record.Embedding) == 0 {
		return "", errors.New("record must have an embedding for vector storage")
	}

	if len(record.Embedding) != a.dimensionSize {
		return "", fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(record.Embedding), a.dimensionSize)
	}

	// Convert embedding to a format suitable for pgvector
	embeddingStr := embedToString(record.Embedding)

	// Begin a transaction
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Insert or update the record
	_, err = tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			id, entity_id, user_id, access_level, content, metadata, embedding, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7::vector, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			entity_id = $2,
			user_id = $3,
			access_level = $4,
			content = $5,
			metadata = $6,
			embedding = $7::vector,
			updated_at = $9
	`, a.tableName),
		record.ID,
		string(record.EntityID),
		record.UserID,
		int(record.AccessLevel),
		record.Content,
		record.Metadata,
		embeddingStr,
		record.CreatedAt,
		record.UpdatedAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to store record: %w", err)
	}

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug("Stored record in pgvector",
		"id", record.ID,
		"entity_id", record.EntityID,
		"table", a.tableName)

	return record.ID, nil
}

// Retrieve fetches memory records matching the query
func (a *PgvectorAdapter) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	if a.db == nil {
		return nil, ErrPgvectorUnavailable
	}

	// Extract entity context for logging
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	log.Debug("PgVector retrieve operation",
		"entity_id", entityCtx.EntityID,
		"has_embedding", len(query.Embedding) > 0,
		"has_exact_match", query.ExactMatch != nil,
		"has_filters", query.Filters != nil,
		"text", query.Text)

	// Determine the retrieval mode based on the query
	var rows pgx.Rows
	var err error

	switch {
	case query.ExactMatch != nil && query.ExactMatch["id"] != nil:
		// ID-based lookup
		recordID, ok := query.ExactMatch["id"].(string)
		if !ok {
			return nil, errors.New("invalid record ID in query")
		}
		log.Debug("PgVector ID-based lookup", "record_id", recordID, "entity_id", entityCtx.EntityID)
		rows, err = a.retrieveByID(ctx, recordID)
	case len(query.Embedding) > 0:
		// Semantic search with vector
		log.Debug("PgVector semantic search", 
			"entity_id", entityCtx.EntityID, 
			"embedding_size", len(query.Embedding))
		rows, err = a.retrieveSemantic(ctx, query)
	default:
		// Filter-based search
		log.Debug("PgVector filter-based search", 
			"entity_id", entityCtx.EntityID,
			"filters", fmt.Sprintf("%v", query.Filters))
		rows, err = a.retrieveByFilters(ctx, query)
	}

	if err != nil {
		log.Error("PgVector retrieve error", "error", err, "entity_id", entityCtx.EntityID)
		return nil, err
	}
	defer rows.Close()

	// Convert from database rows to ltm.MemoryRecord
	records, err := a.convertRowsToMemoryRecords(ctx, rows)
	if err != nil {
		return nil, err
	}

	log.Debug("PgVector retrieve complete", 
		"entity_id", entityCtx.EntityID,
		"records_found", len(records))

	return records, nil
}

// retrieveByID retrieves a record by its ID
func (a *PgvectorAdapter) retrieveByID(ctx context.Context, recordID string) (pgx.Rows, error) {
	// Extract entity context for isolation
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	query := fmt.Sprintf(`
		SELECT id, entity_id, user_id, access_level, content, metadata, embedding, created_at, updated_at 
		FROM %s 
		WHERE id = $1 AND entity_id = $2
	`, a.tableName)

	rows, err := a.db.Query(ctx, query, recordID, string(entityCtx.EntityID))
	if err != nil {
		return nil, fmt.Errorf("failed to query by ID: %w", err)
	}

	return rows, nil
}

// retrieveSemantic performs semantic search with the query vector
func (a *PgvectorAdapter) retrieveSemantic(ctx context.Context, query ltm.LTMQuery) (pgx.Rows, error) {
	if len(query.Embedding) == 0 {
		return nil, ErrMissingQueryVector
	}

	if len(query.Embedding) != a.dimensionSize {
		return nil, fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(query.Embedding), a.dimensionSize)
	}

	// Extract entity context for isolation
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	// Ensure entity_id filter is present for proper isolation
	if query.Filters == nil {
		query.Filters = make(map[string]interface{})
	}
	
	// Only set entity_id if it's not already in the filters
	if _, hasEntityID := query.Filters["entity_id"]; !hasEntityID {
		query.Filters["entity_id"] = entityCtx.EntityID
	}

	// Convert embedding to a format suitable for pgvector
	embeddingStr := embedToString(query.Embedding)

	// Build the WHERE clause for filtering
	whereClause, args := a.buildWhereClause(query)
	args = append(args, embeddingStr) // Add embedding as the last argument

	// Set default limit if not specified
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	// Choose the distance function based on the configured metric
	var distanceFunc string
	switch a.distanceMetric {
	case "cosine":
		distanceFunc = "embedding <=> $%d"
	case "euclidean":
		distanceFunc = "embedding <-> $%d"
	case "dot":
		distanceFunc = "embedding <#> $%d"
	}

	// Build the query
	sqlQuery := fmt.Sprintf(`
		SELECT id, entity_id, user_id, access_level, content, metadata, embedding, created_at, updated_at
		FROM %s
		WHERE %s
		ORDER BY %s
		LIMIT %d
	`, a.tableName, whereClause, fmt.Sprintf(distanceFunc, len(args)), limit)

	// Execute the query
	rows, err := a.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to perform semantic search: %w", err)
	}

	return rows, nil
}

// retrieveByFilters retrieves records using metadata filters
func (a *PgvectorAdapter) retrieveByFilters(ctx context.Context, query ltm.LTMQuery) (pgx.Rows, error) {
	// Extract entity context for isolation
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	// Ensure entity_id filter is present for proper isolation
	if query.Filters == nil {
		query.Filters = make(map[string]interface{})
	}
	
	// Only set entity_id if it's not already in the filters
	if _, hasEntityID := query.Filters["entity_id"]; !hasEntityID {
		query.Filters["entity_id"] = entityCtx.EntityID
	}

	// Build the WHERE clause for filtering
	whereClause, args := a.buildWhereClause(query)

	// Set default limit if not specified
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	var sqlQuery string

	// If text search is specified, use it
	if query.Text != "" {
		// Add text search parameter to args
		args = append(args, "%"+query.Text+"%")
		sqlQuery = fmt.Sprintf(`
			SELECT id, entity_id, user_id, access_level, content, metadata, embedding, created_at, updated_at
			FROM %s
			WHERE %s AND content ILIKE $%d
			ORDER BY updated_at DESC
			LIMIT %d
		`, a.tableName, whereClause, len(args), limit)
	} else {
		sqlQuery = fmt.Sprintf(`
			SELECT id, entity_id, user_id, access_level, content, metadata, embedding, created_at, updated_at
			FROM %s
			WHERE %s
			ORDER BY updated_at DESC
			LIMIT %d
		`, a.tableName, whereClause, limit)
	}

	// Execute the query
	rows, err := a.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query by filters: %w", err)
	}

	return rows, nil
}

// buildWhereClause constructs a WHERE clause for SQL queries based on the query parameters
func (a *PgvectorAdapter) buildWhereClause(query ltm.LTMQuery) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	// Start with a true condition to make it easier to add AND clauses
	conditions = append(conditions, "TRUE")
	
	// Process exact match criteria
	if query.ExactMatch != nil {
		for k, v := range query.ExactMatch {
			if k == "id" {
				continue // Skip ID as it's handled separately
			}

			paramIndex := len(args) + 1
			conditions = append(conditions, fmt.Sprintf("%s = $%d", k, paramIndex))

			switch val := v.(type) {
			case string:
				args = append(args, val)
			case entity.EntityID:
				args = append(args, string(val))
			case int:
				args = append(args, val)
			case bool:
				args = append(args, val)
			case entity.AccessLevel:
				args = append(args, int(val))
			default:
				args = append(args, fmt.Sprintf("%v", val))
			}
		}
	}

	// Process general filters
	if query.Filters != nil {
		for k, v := range query.Filters {
			paramIndex := len(args) + 1

			switch k {
			case "entity_id":
				conditions = append(conditions, fmt.Sprintf("entity_id = $%d", paramIndex))
				if entityID, ok := v.(entity.EntityID); ok {
					args = append(args, string(entityID))
				} else {
					args = append(args, fmt.Sprintf("%v", v))
				}
			case "user_id":
				conditions = append(conditions, fmt.Sprintf("user_id = $%d", paramIndex))
				args = append(args, fmt.Sprintf("%v", v))
			case "access_level":
				conditions = append(conditions, fmt.Sprintf("access_level = $%d", paramIndex))
				if accessLevel, ok := v.(entity.AccessLevel); ok {
					args = append(args, int(accessLevel))
				} else {
					args = append(args, v)
				}
			default:
				// For other metadata fields, use JSONB query
				conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", k, paramIndex))
				args = append(args, fmt.Sprintf("%v", v))
			}
		}
	}

	return strings.Join(conditions, " AND "), args
}

// Update modifies an existing memory record
func (a *PgvectorAdapter) Update(ctx context.Context, record ltm.MemoryRecord) error {
	if record.ID == "" {
		return errors.New("record ID cannot be empty")
	}

	if len(record.Embedding) != a.dimensionSize {
		return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(record.Embedding), a.dimensionSize)
	}

	// Convert embedding to a format suitable for pgvector
	embeddingStr := embedToString(record.Embedding)

	// Check if the record exists
	var exists bool
	err := a.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)
	`, a.tableName), record.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	if !exists {
		return ErrRecordNotFound
	}

	// Update the timestamp
	record.UpdatedAt = time.Now()

	// Update the record
	_, err = a.db.Exec(ctx, fmt.Sprintf(`
		UPDATE %s SET
			entity_id = $1,
			user_id = $2,
			access_level = $3,
			content = $4,
			metadata = $5,
			embedding = $6::vector,
			updated_at = $7
		WHERE id = $8
	`, a.tableName),
		string(record.EntityID),
		record.UserID,
		int(record.AccessLevel),
		record.Content,
		record.Metadata,
		embeddingStr,
		record.UpdatedAt,
		record.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	log.Debug("Updated record in pgvector",
		"id", record.ID,
		"entity_id", record.EntityID,
		"table", a.tableName)

	return nil
}

// Delete removes a memory record
func (a *PgvectorAdapter) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("record ID cannot be empty")
	}

	result, err := a.db.Exec(ctx, fmt.Sprintf(`
		DELETE FROM %s WHERE id = $1
	`, a.tableName), id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	log.Debug("Deleted record from pgvector",
		"id", id,
		"table", a.tableName)

	return nil
}

// SupportsVectorSearch returns true as this adapter supports vector search
func (a *PgvectorAdapter) SupportsVectorSearch() bool {
	return true
}

// convertRowsToMemoryRecords converts database rows to MemoryRecord objects
func (a *PgvectorAdapter) convertRowsToMemoryRecords(ctx context.Context, rows pgx.Rows) ([]ltm.MemoryRecord, error) {
	var records []ltm.MemoryRecord

	for rows.Next() {
		var record ltm.MemoryRecord
		var entityIDStr string
		var accessLevel int
		var embeddingStr string

		err := rows.Scan(
			&record.ID,
			&entityIDStr,
			&record.UserID,
			&accessLevel,
			&record.Content,
			&record.Metadata,
			&embeddingStr,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		record.EntityID = entity.EntityID(entityIDStr)
		record.AccessLevel = entity.AccessLevel(accessLevel)
		record.Embedding = stringToEmbed(embeddingStr)

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}

// Helper function to convert []float32 to string for pgvector
func embedToString(embedding []float32) string {
	elements := make([]string, len(embedding))
	for i, v := range embedding {
		elements[i] = strconv.FormatFloat(float64(v), 'f', -1, 32)
	}
	return "[" + strings.Join(elements, ",") + "]"
}

// Helper function to convert pgvector string to []float32
func stringToEmbed(embeddingStr string) []float32 {
	// Remove brackets
	embeddingStr = strings.TrimPrefix(embeddingStr, "[")
	embeddingStr = strings.TrimSuffix(embeddingStr, "]")

	// Split by comma
	elements := strings.Split(embeddingStr, ",")
	embedding := make([]float32, len(elements))

	for i, element := range elements {
		val, err := strconv.ParseFloat(strings.TrimSpace(element), 32)
		if err != nil {
			// Log error and continue with 0
			log.Error("Failed to parse embedding element", "error", err, "element", element)
			val = 0
		}
		embedding[i] = float32(val)
	}

	return embedding
}