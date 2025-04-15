package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
)

// PostgresStore implements the LTMStore interface using a PostgreSQL database.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore with the given connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		pool: pool,
	}
}

// Store persists a memory record to the PostgreSQL database.
func (p *PostgresStore) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
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

	// Convert metadata to JSONB
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Insert record into database and get the generated ID
	var id string
	err = p.pool.QueryRow(ctx,
		`INSERT INTO memory_records (
			entity_id, user_id, access_level, content, metadata
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		record.EntityID, record.UserID, record.AccessLevel, record.Content, metadataJSON,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to store record: %w", err)
	}

	return id, nil
}

// Retrieve fetches memory records matching the query from the PostgreSQL database.
func (p *PostgresStore) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	// Build the query dynamically based on the provided filters
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, entity_id, user_id, access_level, content, metadata, created_at, updated_at
		FROM memory_records
		WHERE entity_id = $1
	`)

	// Build parameter list, starting with entity ID
	params := []interface{}{entityCtx.EntityID}
	paramIndex := 2 // Start at $2 since $1 is entity_id

	// Handle access level filtering
	if entityCtx.UserID != "" {
		// User provided: can see shared records and private records for this user
		queryBuilder.WriteString(` AND (access_level = $`)
		queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
		queryBuilder.WriteString(` OR (access_level = $`)
		paramIndex++
		queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
		queryBuilder.WriteString(` AND user_id = $`)
		paramIndex++
		queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
		queryBuilder.WriteString(`))`)
		
		params = append(params, entity.SharedWithinEntity, entity.PrivateToUser, entityCtx.UserID)
		paramIndex++
	} else {
		// No user provided: can only see shared records
		queryBuilder.WriteString(` AND access_level = $`)
		queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
		params = append(params, entity.SharedWithinEntity)
		paramIndex++
	}

	// Handle text search
	if query.Text != "" {
		queryBuilder.WriteString(` AND content ILIKE $`)
		queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
		params = append(params, "%"+query.Text+"%")
		paramIndex++
	}

	// Handle exact match for ID
	if query.ExactMatch != nil {
		if id, ok := query.ExactMatch["ID"]; ok {
			queryBuilder.WriteString(` AND id = $`)
			queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
			params = append(params, id)
			paramIndex++
		}
	}

	// Handle limit
	limit := 100 // Default limit
	if query.Limit > 0 {
		limit = query.Limit
	}
	queryBuilder.WriteString(` ORDER BY created_at DESC LIMIT $`)
	queryBuilder.WriteString(fmt.Sprintf("%d", paramIndex))
	params = append(params, limit)

	// Execute the query
	rows, err := p.pool.Query(ctx, queryBuilder.String(), params...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve records: %w", err)
	}
	defer rows.Close()

	// Parse results
	var records []ltm.MemoryRecord
	for rows.Next() {
		var record ltm.MemoryRecord
		var metadataJSON []byte

		err := rows.Scan(
			&record.ID,
			&record.EntityID,
			&record.UserID,
			&record.AccessLevel,
			&record.Content,
			&metadataJSON,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		// Parse metadata from JSON
		if len(metadataJSON) > 0 {
			record.Metadata = make(map[string]interface{})
			if err := json.Unmarshal(metadataJSON, &record.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	// Apply additional filtering for metadata
	// This is a simple implementation that does filtering in memory
	// For better performance, this could be moved to the SQL query
	if query.Filters != nil && len(query.Filters) > 0 {
		var filteredRecords []ltm.MemoryRecord
		for _, record := range records {
			if matchesFilters(record, query.Filters) {
				filteredRecords = append(filteredRecords, record)
			}
		}
		records = filteredRecords
	}

	return records, nil
}

// Update modifies an existing memory record in the PostgreSQL database.
func (p *PostgresStore) Update(ctx context.Context, record ltm.MemoryRecord) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}

	// Require ID
	if record.ID == "" {
		return errors.New("record ID is required for update")
	}

	// Convert metadata to JSONB
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update the record, ensuring it belongs to the correct entity
	commandTag, err := p.pool.Exec(ctx,
		`UPDATE memory_records
		SET content = $3, metadata = $4, updated_at = NOW()
		WHERE id = $1 AND entity_id = $2`,
		record.ID, entityCtx.EntityID, record.Content, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	// Check if a record was actually updated
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("record with ID %s not found or belongs to another entity", record.ID)
	}

	return nil
}

// Delete removes a memory record from the PostgreSQL database.
func (p *PostgresStore) Delete(ctx context.Context, id string) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}

	// Delete the record, ensuring it belongs to the correct entity
	commandTag, err := p.pool.Exec(ctx,
		`DELETE FROM memory_records
		WHERE id = $1 AND entity_id = $2`,
		id, entityCtx.EntityID,
	)

	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Check if a record was actually deleted
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("record with ID %s not found or belongs to another entity", id)
	}

	return nil
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