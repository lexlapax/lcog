-- name: CreateMemoryRecord :one
INSERT INTO memory_records (
    entity_id,
    user_id,
    access_level,
    content,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, entity_id, user_id, access_level, content, metadata, created_at, updated_at;

-- name: GetMemoryRecord :one
SELECT * FROM memory_records
WHERE id = $1 AND entity_id = $2
LIMIT 1;

-- name: ListMemoryRecordsByEntityID :many
SELECT * FROM memory_records
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListMemoryRecordsByEntityIDAndAccessLevel :many
SELECT * FROM memory_records
WHERE entity_id = $1 AND access_level = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: ListMemoryRecordsByEntityIDAndUserID :many
SELECT * FROM memory_records
WHERE entity_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: ListMemoryRecordsByEntityIDWithTextSearch :many
SELECT * FROM memory_records
WHERE entity_id = $1 AND content ILIKE '%' || $2 || '%'
ORDER BY created_at DESC
LIMIT $3;

-- name: UpdateMemoryRecord :exec
UPDATE memory_records
SET 
    content = $3,
    metadata = $4,
    updated_at = NOW()
WHERE id = $1 AND entity_id = $2;

-- name: DeleteMemoryRecord :exec
DELETE FROM memory_records
WHERE id = $1 AND entity_id = $2;