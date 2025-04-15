package reflection

// Lua hook function names for reflection operations
const (
	// Called before performing reflection analysis
	// Parameters: memories []ltm.MemoryRecord
	// Return: bool (if true, skip analysis)
	beforeReflectionAnalysisFuncName = "before_reflection_analysis"
	
	// Called after insights are generated
	// Parameters: insights []*Insight
	// Return: nil
	afterInsightGenerationFuncName = "after_insight_generation"
	
	// Called before consolidating insights into LTM
	// Parameters: insights []*Insight
	// Return: []*Insight (potentially modified insights)
	beforeConsolidationFuncName = "before_consolidation"
)