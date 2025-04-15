--[[  
  Reflection Analysis Script
  
  This script provides hooks that are called during the reflection process.
  The hooks allow for customizing how reflection analysis is performed.
]]

-- Called before performing reflection analysis
-- Parameter: memories - Array of memory records
-- Returns: Boolean - If true, skip the analysis
function before_reflection_analysis(memories)
  cogmem.log("debug", "Before reflection analysis hook called. Memories to analyze: " .. #memories)

  -- Example logic: Skip analysis if there are fewer than 3 memories
  if #memories < 3 then
    cogmem.log("info", "Skipping reflection analysis due to insufficient memory count")
    return true
  end

  -- More advanced logic could be added here
  -- For example, categorize memories, filter out certain types, etc.

  return false -- Proceed with analysis
end

-- Called after insights are generated
-- Parameter: insights - Array of generated insights
-- Returns: nil
function after_insight_generation(insights)
  cogmem.log("debug", "After insight generation hook called. Insights generated: " .. #insights)

  -- Example: Log high confidence insights
  for i, insight in ipairs(insights) do
    if insight.confidence > 0.8 then
      cogmem.log("info", "High confidence insight: " .. insight.description)
    end
  end

  -- More advanced logic could be added here
  -- For example, tag insights, correlate with external knowledge, etc.
end

-- Called before consolidating insights into LTM
-- Parameter: insights - Array of insights to be consolidated
-- Returns: Potentially modified array of insights
function before_consolidation(insights)
  cogmem.log("debug", "Before consolidation hook called. Insights to consolidate: " .. #insights)
  
  -- Example: Filter out low confidence insights
  local filtered_insights = {}
  for i, insight in ipairs(insights) do
    if insight.confidence >= 0.6 then
      table.insert(filtered_insights, insight)
    else
      cogmem.log("debug", "Filtered out low confidence insight: " .. insight.description)
    end
  end
  
  -- Example: Add additional metadata to insights
  for i, insight in ipairs(filtered_insights) do
    if not insight.metadata then
      insight.metadata = {}
    end
    
    insight.metadata.processed_by_lua = true
    insight.metadata.processing_timestamp = cogmem.now()
  end
  
  return filtered_insights
end
