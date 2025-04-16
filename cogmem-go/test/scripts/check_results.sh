#!/bin/bash
# check_results.sh - Helper script to extract expected results and compare with actual output
# Usage: ./check_results.sh <test_script> <output_file>

if [ $# -ne 2 ]; then
    echo "Usage: $0 <test_script> <output_file>"
    exit 1
fi

TEST_SCRIPT=$1
OUTPUT_FILE=$2

if [ ! -f "$TEST_SCRIPT" ]; then
    echo "Test script $TEST_SCRIPT not found!"
    exit 1
fi

if [ ! -f "$OUTPUT_FILE" ]; then
    echo "Output file $OUTPUT_FILE not found!"
    exit 1
fi

# Extract expected results
EXPECTED_FILE=$(mktemp)
grep -A 1 "# Expect:" "$TEST_SCRIPT" | grep -v "# Expect:" | sed 's/^# //' > "$EXPECTED_FILE"

echo "=== Expected vs Actual Results ==="
echo "Expected results extracted to: $EXPECTED_FILE"
echo "Actual output in: $OUTPUT_FILE"
echo ""
echo "Please compare the files to verify test results."
echo "You can use a diff tool or manually review both files."
echo ""
echo "=== Summary of Expected Results ==="
cat "$EXPECTED_FILE"