package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// HstoreStore implements the LTMStore interface using PostgreSQL with hstore extension.
type HstoreStore struct {
	db *sqlx.DB
}

// NewHstoreStore creates a new PostgreSQL HstoreStore with the given database connection.
func NewHstoreStore(db *sqlx.DB) *HstoreStore {
	store := &HstoreStore{
		db: db,
	}

	log.Debug("Initialized PostgreSQL Hstore LTM store adapter")
	return store
}

// Initialize creates the required tables if they don't exist.
func (h *HstoreStore) Initialize(ctx context.Context) error {
	log.DebugContext(ctx, "Initializing PostgreSQL Hstore store tables")

	// Create the hstore extension if it doesn't exist
	_, err := h.db.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS hstore;
	`)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create hstore extension", "error", err)
		return fmt.Errorf("failed to create hstore extension: %w", err)
	}

	// Create the memory_records table if it doesn't exist
	_, err = h.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS memory_records (
			id TEXT PRIMARY KEY,
			entity_id TEXT NOT NULL,
			user_id TEXT,
			access_level INTEGER NOT NULL,
			content TEXT NOT NULL,
			metadata HSTORE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL
		);
	`)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create memory_records table", "error", err)
		return fmt.Errorf("failed to create memory_records table: %w", err)
	}

	// Create index on entity_id for faster lookups
	_, err = h.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS memory_records_entity_id_idx ON memory_records (entity_id);
	`)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create entity_id index", "error", err)
		return fmt.Errorf("failed to create entity_id index: %w", err)
	}

	log.DebugContext(ctx, "Successfully initialized PostgreSQL Hstore store tables")
	return nil
}

// Store persists a memory record to the PostgreSQL database.
func (h *HstoreStore) Store(ctx context.Context, record ltm.MemoryRecord) (string, error) {
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

	// Convert metadata to hstore format
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert JSON to hstore string format
	var hstoreStr string
	if len(record.Metadata) > 0 {
		hstoreStr, err = jsonToHstoreStr(metadataJSON)
		if err != nil {
			return "", fmt.Errorf("failed to convert metadata to hstore: %w", err)
		}
	}

	// Store the record in the database
	query := `
		INSERT INTO memory_records (
			id, entity_id, user_id, access_level, content, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6::hstore, $7, $8
		) ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	var id string
	err = h.db.QueryRowContext(ctx, query,
		record.ID,
		string(record.EntityID),
		record.UserID,
		int(record.AccessLevel),
		record.Content,
		hstoreStr,
		record.CreatedAt,
		record.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to store record: %w", err)
	}

	return id, nil
}

// Retrieve fetches memory records matching the query from the PostgreSQL database.
func (h *HstoreStore) Retrieve(ctx context.Context, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	// Build the SQL query
	sqlQuery := `
		SELECT id, entity_id, user_id, access_level, content, metadata, created_at, updated_at
		FROM memory_records
		WHERE entity_id = $1
	`

	args := []interface{}{string(entityCtx.EntityID)}
	argIdx := 2

	// Add user ID filtering for private records if user ID is provided
	if entityCtx.UserID != "" {
		sqlQuery += fmt.Sprintf(` AND (access_level = %d OR (access_level = %d AND user_id = $%d))`, 
			entity.SharedWithinEntity, entity.PrivateToUser, argIdx)
		args = append(args, entityCtx.UserID)
		argIdx++
	} else {
		sqlQuery += fmt.Sprintf(` AND access_level = %d`, entity.SharedWithinEntity)
	}

	// Add exact match filtering if provided
	if query.ExactMatch != nil {
		if id, ok := query.ExactMatch["ID"]; ok {
			idStr, ok := id.(string)
			if !ok {
				return nil, fmt.Errorf("ID must be a string")
			}
			sqlQuery += fmt.Sprintf(` AND id = $%d`, argIdx)
			args = append(args, idStr)
			argIdx++
		}
	}

	// Add text search filtering if provided
	if query.Text != "" {
		sqlQuery += fmt.Sprintf(` AND content ILIKE $%d`, argIdx)
		args = append(args, "%"+query.Text+"%")
		argIdx++
	}

	// Add metadata filtering if provided
	if query.Filters != nil && len(query.Filters) > 0 {
		for key, value := range query.Filters {
			valueStr, ok := value.(string)
			if !ok {
				valueBytes, err := json.Marshal(value)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal metadata filter value: %w", err)
				}
				valueStr = string(valueBytes)
			}
			sqlQuery += fmt.Sprintf(` AND metadata -> $%d = $%d`, argIdx, argIdx+1)
			args = append(args, key, valueStr)
			argIdx += 2
		}
	}

	// Order by created_at descending (newest first)
	sqlQuery += ` ORDER BY created_at DESC`

	// Add limit if provided
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, query.Limit)
	} else {
		sqlQuery += ` LIMIT 100`
	}

	// Execute the query
	rows, err := h.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve records: %w", err)
	}
	defer rows.Close()

	// Process results
	var records []ltm.MemoryRecord
	for rows.Next() {
		var record ltm.MemoryRecord
		var entityIDStr string
		var accessLevelInt int
		var metadataHstore string

		err := rows.Scan(
			&record.ID,
			&entityIDStr,
			&record.UserID,
			&accessLevelInt,
			&record.Content,
			&metadataHstore,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		// Convert string to EntityID
		record.EntityID = entity.EntityID(entityIDStr)
		
		// Convert int to AccessLevel
		record.AccessLevel = entity.AccessLevel(accessLevelInt)

		// Convert hstore string to metadata map
		if metadataHstore != "" {
			metadataMap, err := hstoreStrToMap(metadataHstore)
			if err != nil {
				return nil, fmt.Errorf("failed to parse metadata: %w", err)
			}
			record.Metadata = metadataMap
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating records: %w", err)
	}

	return records, nil
}

// Update modifies an existing memory record in the PostgreSQL database.
func (h *HstoreStore) Update(ctx context.Context, record ltm.MemoryRecord) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}

	// Require ID
	if record.ID == "" {
		return errors.New("record ID is required for update")
	}

	// First check if the record exists and belongs to the entity
	var existingEntityID string
	err := h.db.QueryRowContext(ctx, `
		SELECT entity_id FROM memory_records WHERE id = $1
	`, record.ID).Scan(&existingEntityID)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("record with ID %s not found", record.ID)
		}
		return fmt.Errorf("failed to check record existence: %w", err)
	}

	// Ensure the record belongs to the entity in context
	if existingEntityID != string(entityCtx.EntityID) {
		return fmt.Errorf("record belongs to another entity")
	}

	// Convert metadata to hstore format
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert JSON to hstore string format
	var hstoreStr string
	if len(record.Metadata) > 0 {
		hstoreStr, err = jsonToHstoreStr(metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to convert metadata to hstore: %w", err)
		}
	}

	// Update the record
	now := time.Now().UTC()
	_, err = h.db.ExecContext(ctx, `
		UPDATE memory_records SET
			content = $1,
			metadata = $2::hstore,
			updated_at = $3
		WHERE id = $4 AND entity_id = $5
	`, record.Content, hstoreStr, now, record.ID, string(entityCtx.EntityID))

	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	return nil
}

// Delete removes a memory record from the PostgreSQL database.
func (h *HstoreStore) Delete(ctx context.Context, id string) error {
	// Extract entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return entity.ErrMissingEntityContext
	}

	// First check if the record exists and belongs to the entity
	var existingEntityID string
	err := h.db.QueryRowContext(ctx, `
		SELECT entity_id FROM memory_records WHERE id = $1
	`, id).Scan(&existingEntityID)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("record with ID %s not found", id)
		}
		return fmt.Errorf("failed to check record existence: %w", err)
	}

	// Ensure the record belongs to the entity in context
	if existingEntityID != string(entityCtx.EntityID) {
		return fmt.Errorf("record belongs to another entity")
	}

	// Delete the record
	_, err = h.db.ExecContext(ctx, `
		DELETE FROM memory_records WHERE id = $1 AND entity_id = $2
	`, id, string(entityCtx.EntityID))

	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// Helper functions

// jsonToHstoreStr converts a JSON object to a PostgreSQL hstore string format.
// For example: {"key": "value", "num": 123} -> "key"=>"value", "num"=>"123"
func jsonToHstoreStr(jsonData []byte) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", err
	}

	var pairs []string
	for k, v := range data {
		var valStr string
		switch val := v.(type) {
		case string:
			valStr = val
		case nil:
			valStr = ""
		default:
			// Convert non-string values to JSON strings
			valBytes, err := json.Marshal(val)
			if err != nil {
				return "", err
			}
			valStr = string(valBytes)
		}
		pairs = append(pairs, fmt.Sprintf(`"%s"=>"%s"`, k, escapeHstoreValue(valStr)))
	}

	return strings.Join(pairs, ", "), nil
}

// escapeHstoreValue escapes quotes and backslashes in hstore values.
func escapeHstoreValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// hstoreStrToMap converts a PostgreSQL hstore string to a map.
// For example: "key"=>"value", "num"=>"123" -> {"key": "value", "num": "123"}
func hstoreStrToMap(hstoreStr string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if hstoreStr == "" {
		return result, nil
	}

	// Parse hstore string into key-value pairs
	// This is a simplified parser and may not handle all edge cases
	pairs := strings.Split(hstoreStr, ", ")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=>")
		if len(parts) != 2 {
			continue
		}

		key := strings.Trim(parts[0], `"`)
		val := strings.Trim(parts[1], `"`)

		// Try to parse different types of values
		if val == "true" {
			result[key] = true
		} else if val == "false" {
			result[key] = false
		} else if isNumeric(val) {
			// Try to parse as number
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				result[key] = f
			} else {
				result[key] = val
			}
		} else if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
			// Try to parse as a JSON object
			// First, unescape the escaped quotes
			unescapedVal := strings.ReplaceAll(val, `\\`, `\`)
			unescapedVal = strings.ReplaceAll(unescapedVal, `\"`, `"`)
			
			var jsonVal interface{}
			if err := json.Unmarshal([]byte(unescapedVal), &jsonVal); err == nil {
				result[key] = jsonVal
			} else {
				result[key] = val
			}
		} else if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			// Try to parse as a JSON array
			// First, unescape the escaped quotes
			unescapedVal := strings.ReplaceAll(val, `\\`, `\`)
			unescapedVal = strings.ReplaceAll(unescapedVal, `\"`, `"`)
			
			var jsonVal interface{}
			if err := json.Unmarshal([]byte(unescapedVal), &jsonVal); err == nil {
				result[key] = jsonVal
			} else {
				result[key] = val
			}
		} else {
			result[key] = val
		}
	}

	return result, nil
}

// isNumeric checks if a string is a valid JSON number.
func isNumeric(s string) bool {
	return strings.ContainsAny(s, "0123456789") &&
		!strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
}