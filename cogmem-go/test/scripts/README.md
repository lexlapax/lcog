# CogMem Example Client Script-Based Tests

This directory contains script-based tests for the CogMem example client. These tests validate the functionality of the example-client command-line application with different storage backends.

## Test Scripts

Each test script contains a series of commands that are piped to the example-client, and comments indicating the expected results:

- `mock_test.txt`: Tests with the mock LTM backend (no external dependencies)
- `boltdb_test.txt`: Tests with the BoltDB KV backend
- `sqlite_test.txt`: Tests with the SQLite SQL backend
- `postgres_test.txt`: Tests with PostgreSQL and pgvector (requires database and API keys)
- `chromemgo_test.txt`: Tests with ChromemGo vector database (requires API keys for embeddings)

## Helper Scripts

- `run_tests.sh`: Main script to run the tests with specified backends
- `check_results.sh`: Helper script to extract expected results from test scripts

## Running the Tests

The example-client has been updated with a `-s` flag to read commands from stdin and exit when done, which makes it easier to run script-based tests.

### From the Makefile

The tests can be run directly from the repository root using the Makefile:

```
# Run the default test (mock backend)
make test-cmd-script

# Run tests with specific backends
make test-cmd-script-mock
make test-cmd-script-boltdb
make test-cmd-script-sqlite
make test-cmd-script-postgres  # Requires POSTGRES_URL and OPENAI_API_KEY env vars
make test-cmd-script-chromemgo  # Requires OPENAI_API_KEY env var

# Run all tests
make test-cmd-script-all
```

### Manual Execution

You can also run the scripts manually:

```
# From the test/scripts directory
./run_tests.sh mock
./run_tests.sh boltdb
./run_tests.sh sqlite
./run_tests.sh postgres
./run_tests.sh chromemgo
./run_tests.sh all
```

## Requirements

- For PostgreSQL tests: 
  - Set `POSTGRES_URL` environment variable
  - Set `OPENAI_API_KEY` environment variable
  - PostgreSQL with pgvector extension installed

- For ChromemGo tests:
  - Set `OPENAI_API_KEY` environment variable

## Output

The test output is stored in the `output/` directory:
- `<backend>_output.txt`: Standard output from the test
- `<backend>_errors.txt`: Error output from the test

## Checking Results

After running tests, you can manually review the output files or use the check_results.sh script:

```
./check_results.sh <test_script> <output_file>
```

For example:

```
./check_results.sh mock_test.txt output/mock_output.txt
```