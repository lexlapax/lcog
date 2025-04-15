-- Create memory_vectors table for pgvector storage
CREATE TABLE IF NOT EXISTS memory_vectors (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    access_level INTEGER NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    embedding VECTOR(1536) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indices for efficient querying
CREATE INDEX IF NOT EXISTS memory_vectors_entity_id_idx ON memory_vectors (entity_id);
CREATE INDEX IF NOT EXISTS memory_vectors_user_id_idx ON memory_vectors (user_id);
CREATE INDEX IF NOT EXISTS memory_vectors_access_level_idx ON memory_vectors (access_level);
CREATE INDEX IF NOT EXISTS memory_vectors_created_at_idx ON memory_vectors (created_at);
CREATE INDEX IF NOT EXISTS memory_vectors_updated_at_idx ON memory_vectors (updated_at);

-- Create vector index for cosine similarity search
CREATE INDEX IF NOT EXISTS memory_vectors_embedding_idx ON memory_vectors USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);