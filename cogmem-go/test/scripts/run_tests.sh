#!/bin/bash
# run_tests.sh - Runner script for the example-client test scripts
# Usage: ./run_tests.sh [mock|boltdb|sqlite|postgres|chromemgo|all]

# Set default test
TEST_TYPE=${1:-mock}

# Path to example-client binary
CLIENT_BIN="../../bin/example-client"

# Build the client if not already built
if [ ! -f "$CLIENT_BIN" ]; then
    echo "Building example-client..."
    cd ../.. && make install
    cd - || exit 1
fi

# Create output directory
mkdir -p output

run_mock_test() {
    echo "Running mock backend test..."
    cat mock_test.txt | $CLIENT_BIN -s --config ../../configs/mock.yaml > output/mock_output.txt 2> output/mock_errors.txt
    if [ $? -eq 0 ]; then
        echo "Mock test completed successfully."
        ./check_results.sh mock_test.txt output/mock_output.txt
    else
        echo "Mock test failed. Check output/mock_errors.txt for details."
    fi
}

run_boltdb_test() {
    echo "Running BoltDB backend test..."
    cat boltdb_test.txt | $CLIENT_BIN -s --config ../../configs/boltdb.yaml > output/boltdb_output.txt 2> output/boltdb_errors.txt
    if [ $? -eq 0 ]; then
        echo "BoltDB test completed successfully."
        ./check_results.sh boltdb_test.txt output/boltdb_output.txt
    else
        echo "BoltDB test failed. Check output/boltdb_errors.txt for details."
    fi
}

run_sqlite_test() {
    echo "Running SQLite backend test..."
    cat sqlite_test.txt | $CLIENT_BIN -s --config ../../configs/sqlite.yaml > output/sqlite_output.txt 2> output/sqlite_errors.txt
    if [ $? -eq 0 ]; then
        echo "SQLite test completed successfully."
        ./check_results.sh sqlite_test.txt output/sqlite_output.txt
    else
        echo "SQLite test failed. Check output/sqlite_errors.txt for details."
    fi
}

run_postgres_test() {
    if [ -z "$POSTGRES_URL" ]; then
        echo "POSTGRES_URL environment variable not set. Using default (postgres://postgres:postgres@localhost:5432/cogmem)."
        export POSTGRES_URL="postgres://postgres:postgres@localhost:5432/cogmem?sslmode=disable"
    fi

    if [ -z "$OPENAI_API_KEY" ]; then
        echo "OPENAI_API_KEY environment variable not set. Using dummy key (will cause OpenAI API errors)."
        export OPENAI_API_KEY="sk-dummy-key-for-testing"
    fi
    
    # Create a temporary config file with the current POSTGRES_URL
    echo "Creating temporary PostgreSQL configuration..."
    TMP_CONFIG=$(mktemp)
    
    # Copy the original config and replace the POSTGRES_URL variable
    cat ../../configs/postgres.yaml | sed "s|\${POSTGRES_URL}|$POSTGRES_URL|g" > "$TMP_CONFIG"
    
    echo "Running PostgreSQL backend test with URL: $POSTGRES_URL"
    cat postgres_test.txt | $CLIENT_BIN -s --config "$TMP_CONFIG" > output/postgres_output.txt 2> output/postgres_errors.txt
    TEST_EXIT_CODE=$?
    
    # Clean up
    rm -f "$TMP_CONFIG"
    
    if [ $TEST_EXIT_CODE -eq 0 ]; then
        echo "PostgreSQL test completed successfully."
        ./check_results.sh postgres_test.txt output/postgres_output.txt
    else
        echo "PostgreSQL test failed. Check output/postgres_errors.txt for details."
    fi
}

run_chromemgo_test() {
    if [ -z "$OPENAI_API_KEY" ]; then
        echo "OPENAI_API_KEY environment variable not set. Using dummy key (will cause OpenAI API errors)."
        export OPENAI_API_KEY="sk-dummy-key-for-testing"
    fi

    # Create a temporary config file with the current OPENAI_API_KEY
    echo "Creating temporary ChromemGo configuration..."
    TMP_CONFIG=$(mktemp)
    
    # Copy the original config and replace the OPENAI_API_KEY variable
    cat ../../configs/chromemgo.yaml | sed "s|\${OPENAI_API_KEY}|$OPENAI_API_KEY|g" > "$TMP_CONFIG"
    
    echo "Running ChromemGo backend test with API key..."
    cat chromemgo_test.txt | $CLIENT_BIN -s --config "$TMP_CONFIG" > output/chromemgo_output.txt 2> output/chromemgo_errors.txt
    TEST_EXIT_CODE=$?
    
    # Clean up
    rm -f "$TMP_CONFIG"
    
    if [ $TEST_EXIT_CODE -eq 0 ]; then
        echo "ChromemGo test completed successfully."
        ./check_results.sh chromemgo_test.txt output/chromemgo_output.txt
    else
        echo "ChromemGo test failed. Check output/chromemgo_errors.txt for details."
    fi
}

# Run selected test(s)
case $TEST_TYPE in
    mock)
        run_mock_test
        ;;
    boltdb)
        run_boltdb_test
        ;;
    sqlite)
        run_sqlite_test
        ;;
    postgres)
        run_postgres_test
        ;;
    chromemgo)
        run_chromemgo_test
        ;;
    all)
        echo "=== Running Mock Backend Test ==="
        run_mock_test
        echo ""
        
        echo "=== Running BoltDB Backend Test ==="
        run_boltdb_test
        echo ""
        
        echo "=== Running SQLite Backend Test ==="
        run_sqlite_test
        echo ""
        
        echo "=== Running PostgreSQL Backend Test ==="
        run_postgres_test
        echo ""
        
        echo "=== Running ChromemGo Backend Test ==="
        run_chromemgo_test
        echo ""
        
        echo "=== All Tests Completed ==="
        ;;
    *)
        echo "Unknown test type: $TEST_TYPE"
        echo "Usage: $0 [mock|boltdb|sqlite|postgres|chromemgo|all]"
        exit 1
        ;;
esac