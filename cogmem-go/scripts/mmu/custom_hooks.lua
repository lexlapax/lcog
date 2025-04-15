--[[  
  Custom MMU Hooks Example
  
  This script demonstrates more advanced hooks for the MMU.
]]

-- Called before LTMStore.Retrieve is executed
function before_retrieve(query)
  cogmem.log("info", "Custom before_retrieve hook - Modifying query")
  
  -- Add a prefix to text queries for demonstration
  if query.text and query.text ~= "" then
    query.text = "[modified] " .. query.text
  end
  
  -- Force a smaller result limit
  query.limit = 5
  
  -- Add a metadata filter for demonstration
  if not query.filters then
    query.filters = {}
  end
  query.filters["important"] = true
  
  cogmem.log("info", "Modified query: " .. cogmem.json_encode(query))
  return query
end

-- Called after LTMStore.Retrieve returns results
function after_retrieve(results)
  cogmem.log("info", "Custom after_retrieve hook - Processing " .. #results .. " results")
  
  -- Filter and transform results for demonstration
  local filtered_results = {}
  
  for i, result in ipairs(results) do
    -- Add a prefix to each result's content
    result.content = "[processed] " .. result.content
    
    -- Add a custom field to metadata
    if not result.metadata then
      result.metadata = {}
    end
    result.metadata["processed_by"] = "custom_hooks.lua"
    result.metadata["processed_at"] = cogmem.now()
    
    table.insert(filtered_results, result)
  end
  
  return filtered_results
end