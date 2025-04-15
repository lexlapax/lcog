--[[  
  MMU Retrieval Filter Script
  
  This script demonstrates custom filtering/ranking of retrieved LTM records.
  It provides hooks that will be called by the MMU during retrieval operations.
]]

-- Called before LTMStore.Retrieve is executed
function before_retrieve(ctx, query)
  -- Log the retrieval operation
  log_info("MMU before_retrieve called with query: " .. (query.text or ""))
  
  -- Optionally modify the query
  -- query.limit = 10
  
  -- Return the possibly modified query
  return query
end

-- Called after LTMStore.Retrieve returns results
function after_retrieve(ctx, results)
  -- Log the number of results
  log_info("MMU after_retrieve processing " .. #results .. " results")
  
  -- You can filter or re-rank the results here
  -- This example just returns them as-is
  return results
end
