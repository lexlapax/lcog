-- Create memory_records table

CREATE TABLE IF NOT EXISTS memory_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id TEXT NOT NULL,
    user_id TEXT,
    access_level INTEGER NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Create index on entity_id for fast filtering
    CONSTRAINT memory_records_entity_id_idx UNIQUE (id, entity_id)
);

-- Add index on entity_id for performance
CREATE INDEX IF NOT EXISTS memory_records_entity_id_idx ON memory_records(entity_id);

-- Add index on metadata for JSON querying
CREATE INDEX IF NOT EXISTS memory_records_metadata_idx ON memory_records USING GIN (metadata);
