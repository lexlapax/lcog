# Reflection Module in CogMem

The Reflection module enables a CogMem-based agent to analyze its own memory records, derive insights, and improve its future behavior. This document covers the configuration, usage, and extension of the reflection capabilities.

## Overview

The Reflection module:

1. Periodically analyzes an agent's memory records
2. Identifies patterns, preferences, and connections
3. Stores insights back into long-term memory
4. Enables future reasoning to draw upon these insights

## Configuration

Enable and configure the Reflection module in your config file:

```yaml
reflection:
  enabled: true
  scripts_path: "scripts/reflection"
  analysis_frequency: 100  # Analyze after every 100 memories
  analysis_model: "gpt-3.5-turbo"  # Optional, defaults to the main reasoning model
  analysis_temperature: 0.3  # Lower temperature for more focused analysis
  max_memories_to_analyze: 50  # Number of recent memories to include in analysis
```

## Integration with CogMemClient

The Reflection module is automatically triggered by the CogMemClient based on the configuration parameters. When using the CogMemClient:

```go
// Create a new CogMemClient with the reflection module enabled
client, err := cogmem.NewCogMemClient(ctx, config, mmu, reasoningEngine, reflectionModule)
if err != nil {
    return err
}

// Process inputs as normal - reflection will be triggered automatically
response, err := client.Process(ctx, input)
```

## Insights

Insights generated during reflection are structured as follows:

```go
type Insight struct {
    ID          string
    Type        string  // "pattern", "preference", "connection", etc.
    Content     string  // Description of the insight
    Confidence  float64 // 0.0-1.0 confidence score
    References  []string // IDs of memory records that led to this insight
    CreatedAt   time.Time
}
```

Insights are stored in the LTM with special metadata that identifies them as insights rather than regular memories.

## Lua Scripting Hooks

The reflection process can be customized using Lua scripts:

### Available Hooks

- `before_reflection_analysis(memories)`: Pre-process or filter memories before analysis
- `after_insight_generation(insights, memories)`: Process or filter insights after generation
- `rank_insights(insights)`: Prioritize insights based on custom criteria

### Example Script

```lua
-- scripts/reflection/analysis.lua

function before_reflection_analysis(memories)
    -- Filter out low-importance memories
    local filtered = {}
    for i, memory in ipairs(memories) do
        if memory.metadata and (memory.metadata.importance or 0) > 0.5 then
            table.insert(filtered, memory)
        end
    end
    return filtered
end

function after_insight_generation(insights, memories)
    -- Add additional metadata to insights
    for i, insight in ipairs(insights) do
        insight.metadata = insight.metadata or {}
        insight.metadata.processed_by = "custom_reflection_script"
    end
    return insights
end
```

## Future Enhancements

Planned enhancements for the Reflection module include:

1. More sophisticated analysis approaches
2. Hierarchical insights with relationships
3. Self-correction based on user feedback
4. Integration with external knowledge bases
5. Hypothesis testing by the agent