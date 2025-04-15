# Retrieval-Augmented Generation (RAG) in CogMem

CogMem now supports vector-based semantic search capabilities, enabling Retrieval-Augmented Generation (RAG) patterns for AI agents. This document explains how to configure and use these capabilities.

## Overview

RAG combines the power of retrieval-based systems with generative AI. When an agent needs to answer a query:

1. Relevant information is retrieved from a vector database based on semantic similarity
2. The retrieved context is provided to the reasoning engine along with the query
3. The reasoning engine generates a response informed by both the query and the retrieved context

## Vector Storage Adapters

CogMem supports multiple vector storage backends:

### Chromem-go

A lightweight, embedded vector database implemented in Go.

**Configuration**:
```yaml
ltm:
  type: "chromemgo"
  chromemgo:
    collection_name: "memories"
    dimensions: 1536
    distance_metric: "cosine"  # cosine, euclidean, or dot
```

### PostgreSQL pgvector

PostgreSQL with the pgvector extension for vector operations.

**Configuration**:
```yaml
ltm:
  type: "pgvector"
  pgvector:
    connection_string: "${PGVECTOR_URL}"
    table_name: "memory_vectors"
    dimensions: 1536
    distance_metric: "cosine"  # cosine, euclidean, or dot
```

## Embedding Generation

Embeddings are generated using the configured Reasoning Engine. Currently, the OpenAI adapter is supported:

```yaml
reasoning:
  engine: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    embedding_model: "text-embedding-ada-002"
```

## Using Semantic Search

The Memory Management Unit (MMU) will automatically:

1. Generate embeddings for stored content when using a vector-capable backend
2. Use semantic search when a query vector or text is provided

Example usage:

```go
// Store content (embeddings generated automatically)
client.Process(ctx, &cogmem.StoreInput{
    Content: "The quick brown fox jumps over the lazy dog.",
})

// Semantic search
results, err := client.Process(ctx, &cogmem.QueryInput{
    Query: "fast animal jumping",  // Will be embedded and used for semantic search
    Options: &cogmem.QueryOptions{
        SemanticSearch: true,
        MaxResults: 5,
    },
})
```

## MMU Vector Operations

The MMU automatically handles vector operations based on the configured LTM store:

1. **Conditional Embedding**: Embeddings are only generated when the LTM store supports vectors
2. **Retrieval Strategy**: Automatically selects between keyword/ID lookup and semantic retrieval
3. **Lua Integration**: Hooks for custom processing of embeddings and search results

## Performance Considerations

- Vector operations can be computationally expensive
- The OpenAI API incurs costs for embedding generation
- Consider caching embeddings for frequently used content

## Next Steps

Future versions will support hybrid search strategies combining vector, keyword, and graph-based approaches.