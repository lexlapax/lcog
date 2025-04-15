package mmu

import (
	"context"
	"fmt"

	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

const (
	// beforeRetrieveFuncName is the name of the Lua function to call before LTM retrieval
	beforeRetrieveFuncName = "before_retrieve"

	// afterRetrieveFuncName is the name of the Lua function to call after LTM retrieval
	afterRetrieveFuncName = "after_retrieve"

	// beforeEncodeFuncName is the name of the Lua function to call before LTM encoding
	beforeEncodeFuncName = "before_encode"

	// afterEncodeFuncName is the name of the Lua function to call after LTM encoding
	afterEncodeFuncName = "after_encode"
)

// callBeforeRetrieveHook calls the before_retrieve Lua hook if available
func callBeforeRetrieveHook(
	ctx context.Context,
	engine scripting.Engine,
	query ltm.LTMQuery,
) (ltm.LTMQuery, error) {
	if engine == nil {
		return query, nil
	}

	// Convert the query to a map for passing to Lua
	queryMap := map[string]interface{}{
		"text":   query.Text,
		"limit":  query.Limit,
		"filters": query.Filters,
	}

	if query.ExactMatch != nil {
		queryMap["exact_match"] = query.ExactMatch
	}

	// Try to call the hook function
	result, err := engine.ExecuteFunction(ctx, beforeRetrieveFuncName, queryMap)
	if err != nil {
		// If the function doesn't exist, that's ok - just continue
		if err.Error() == fmt.Sprintf("%v: %s", scripting.ErrFunctionNotFound, beforeRetrieveFuncName) {
			return query, nil
		}
		// Log the error but don't fail the operation
		log.WarnContext(ctx, "Error calling Lua hook", 
			"hook", beforeRetrieveFuncName, 
			"error", err)
		return query, nil
	}

	// If the function returned nil or not a map, just use the original query
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return query, nil
	}

	// Update the query with the results from Lua
	if text, ok := resultMap["text"].(string); ok {
		query.Text = text
	}

	if limit, ok := resultMap["limit"].(float64); ok {
		query.Limit = int(limit)
	}

	if filters, ok := resultMap["filters"].(map[string]interface{}); ok {
		query.Filters = filters
	}

	if exactMatch, ok := resultMap["exact_match"].(map[string]interface{}); ok {
		query.ExactMatch = exactMatch
	}

	return query, nil
}

// callAfterRetrieveHook calls the after_retrieve Lua hook if available
func callAfterRetrieveHook(
	ctx context.Context,
	engine scripting.Engine,
	results []ltm.MemoryRecord,
) ([]ltm.MemoryRecord, error) {
	// Always attempt to call the hook, even if there are no results
	// This helps in testing and allows hooks that might add results
	if engine == nil {
		return results, nil
	}

	// For simplicity in Phase 1, just pass the number of results and a preview
	// This is safer than trying to pass complex structures to Lua
	summary := fmt.Sprintf("Found %d results", len(results))
	if len(results) > 0 {
		summary = fmt.Sprintf("Found %d results, first one: %s", len(results), results[0].Content)
	}

	// Try to call the hook function with a simple string instead of complex structures
	result, err := engine.ExecuteFunction(ctx, afterRetrieveFuncName, summary)
	if err != nil {
		// If the function doesn't exist, that's ok - just continue
		if err.Error() == fmt.Sprintf("%v: %s", scripting.ErrFunctionNotFound, afterRetrieveFuncName) {
			return results, nil
		}
		// Log the error but don't fail the operation
		log.WarnContext(ctx, "Error calling Lua hook", 
			"hook", afterRetrieveFuncName, 
			"error", err)
		return results, nil
	}

	// In Phase 1, we just acknowledge the script was called
	// but we don't modify the results based on the return value
	if resultStr, ok := result.(string); ok {
		log.DebugContext(ctx, "Lua hook processed results", 
			"hook", afterRetrieveFuncName,
			"result", resultStr)
	}

	return results, nil
}