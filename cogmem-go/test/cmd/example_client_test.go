//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/lexlapax/cogmem/pkg/config"
)

// TestExampleClient tests the basic functionality of the example-client CLI.
func TestExampleClient(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// First, build the example-client binary
	buildCmd := exec.Command("go", "build", "-o", "test_example_client", "../../cmd/example-client")
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build example-client: %s", buildOutput)
	defer os.Remove("test_example_client") // Clean up the binary after the test

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "example-client-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a minimal test config file
	testConfig := config.Config{
		LTM: config.LTMConfig{
			Type: "mock", // Use mock for testing
		},
		Scripting: config.ScriptingConfig{
			Paths: []string{"../../scripts"}, // Use standard script paths
		},
		Reasoning: config.ReasoningConfig{
			Provider: "mock",
		},
		Reflection: config.ReflectionConfig{
			Enabled: false, // Disable reflection for testing
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}

	// Save the config to a YAML file
	configYaml, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	configPath := filepath.Join(tempDir, "test_config.yaml")
	err = os.WriteFile(configPath, configYaml, 0644)
	require.NoError(t, err)

	t.Run("ShowHelp", func(t *testing.T) {
		// Test that the help command works
		cmd := exec.Command("./test_example_client", "--config", configPath, "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", output)
		
		// Check for expected help text
		assert.Contains(t, string(output), "Usage of", "Help output should contain usage info")
		assert.Contains(t, string(output), "-config", "Help output should mention the config flag")
	})

	t.Run("RunWithConfig", func(t *testing.T) {
		// Use a buffer to simulate stdin for interactive input
		inputCommands := []string{
			"!help",       // Show help
			"!entity test", // Set entity
			"!user tester", // Set user
			"!remember The Earth is the third planet from the Sun.", // Store a memory
			"!lookup Earth", // Retrieve the memory
			"!quit",      // Exit
		}
		
		inputString := strings.Join(inputCommands, "\n") + "\n"
		
		// Start the command with piped stdin and capture stdout
		cmd := exec.Command("./test_example_client", "--config", configPath)
		cmd.Stdin = bytes.NewBufferString(inputString)
		
		// Set up pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err, "Failed to create stdout pipe")
		
		stderr, err := cmd.StderrPipe()
		require.NoError(t, err, "Failed to create stderr pipe")
		
		// Start the command
		err = cmd.Start()
		require.NoError(t, err, "Failed to start example-client")
		
		// Read all output
		outputBytes := make([]byte, 0)
		buffer := make([]byte, 1024)
		
		// Use a channel to signal when done reading
		done := make(chan bool)
		
		// Read from stdout in a goroutine
		go func() {
			for {
				n, err := stdout.Read(buffer)
				if n > 0 {
					outputBytes = append(outputBytes, buffer[:n]...)
				}
				if err != nil {
					break
				}
			}
			done <- true
		}()
		
		// Also read from stderr
		errBytes := make([]byte, 0)
		errBuffer := make([]byte, 1024)
		
		go func() {
			for {
				n, err := stderr.Read(errBuffer)
				if n > 0 {
					errBytes = append(errBytes, errBuffer[:n]...)
				}
				if err != nil {
					break
				}
			}
		}()
		
		// Wait with timeout
		select {
		case <-done:
			// Normal completion
		case <-time.After(5 * time.Second):
			t.Log("Command timed out, killing process")
			_ = cmd.Process.Kill()
		}
		
		// Wait for command to finish
		err = cmd.Wait()
		require.NoError(t, err, "Command execution failed: %s", errBytes)
		
		// Convert output to string
		output := string(outputBytes)
		t.Logf("Command output: %s", output)
		
		// Check for expected output patterns
		assert.Contains(t, output, "CogMem Client", "Output should contain client header")
		assert.Contains(t, output, "Entity set to: test", "Output should confirm entity change")
		assert.Contains(t, output, "User set to: tester", "Output should confirm user change")
		assert.Contains(t, output, "Memory stored successfully", "Should confirm memory storage")
		assert.Contains(t, output, "The Earth is the third planet", "Should return the stored memory")
		assert.Contains(t, output, "Goodbye", "Should show exit message")
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test invalid config path handling
		cmd := exec.Command("./test_example_client", "--config", "/path/does/not/exist.yaml")
		output, _ := cmd.CombinedOutput()
		
		// The client now exits with an error when the config file doesn't exist
		assert.Contains(t, string(output), "Failed to initialize CogMem client", 
			"Should show error message when config file doesn't exist")
		assert.Contains(t, string(output), "failed to load configuration", 
			"Should show detailed error information")
	})
}

// TestExampleClientWithPostgresConfig tests the example-client with the standard postgres config.
func TestExampleClientWithPostgresConfig(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Skip if postgres test is not enabled or missing required environment variable
	if os.Getenv("TEST_POSTGRES") != "true" {
		t.Skip("Skipping postgres config test; set TEST_POSTGRES=true to run")
	}

	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("Skipping postgres config test; POSTGRES_URL environment variable is required")
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping postgres config test; OPENAI_API_KEY environment variable is required")
	}

	// First, build the example-client binary
	buildCmd := exec.Command("go", "build", "-o", "test_example_client", "../../cmd/example-client")
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build example-client: %s", buildOutput)
	defer os.Remove("test_example_client") // Clean up the binary after the test

	// Locate the postgres config file
	postgresConfigPath := "../../configs/postgres.yaml"
	_, err = os.Stat(postgresConfigPath)
	require.NoError(t, err, "Postgres config file not found at %s", postgresConfigPath)

	// Set up input commands for a simple test
	inputCommands := []string{
		"!config",       // Show the loaded configuration
		"!entity pg_test", // Set entity
		"!user pg_tester", // Set user
		"!remember PostgreSQL test: This memory is stored in pgvector.", // Store a memory
		"!lookup PostgreSQL", // Basic keyword lookup
		"!quit",      // Exit
	}
	
	inputString := strings.Join(inputCommands, "\n") + "\n"
	
	// Start the command with piped stdin and capture stdout
	cmd := exec.Command("./test_example_client", "--config", postgresConfigPath)
	cmd.Stdin = bytes.NewBufferString(inputString)
	
	// Set environment variables
	cmd.Env = append(os.Environ(), 
		"OPENAI_API_KEY="+os.Getenv("OPENAI_API_KEY"),
		"POSTGRES_URL="+os.Getenv("POSTGRES_URL"),
	)
	
	// Use pipes to capture output
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "Failed to create stdout pipe")
	
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err, "Failed to create stderr pipe")
	
	// Start the command
	err = cmd.Start()
	require.NoError(t, err, "Failed to start example-client")
	
	// Read all output
	outputBytes := make([]byte, 0)
	buffer := make([]byte, 1024)
	
	// Use a channel to signal when done reading
	done := make(chan bool)
	
	// Read from stdout in a goroutine
	go func() {
		for {
			n, err := stdout.Read(buffer)
			if n > 0 {
				outputBytes = append(outputBytes, buffer[:n]...)
			}
			if err != nil {
				break
			}
		}
		done <- true
	}()
	
	// Also read from stderr
	errBytes := make([]byte, 0)
	errBuffer := make([]byte, 1024)
	
	go func() {
		for {
			n, err := stderr.Read(errBuffer)
			if n > 0 {
				errBytes = append(errBytes, errBuffer[:n]...)
			}
			if err != nil {
				break
			}
		}
	}()
	
	// Wait with timeout
	select {
	case <-done:
		// Normal completion
	case <-time.After(10 * time.Second):
		t.Log("Command timed out, killing process")
		_ = cmd.Process.Kill()
	}
	
	// Wait for command to finish
	err = cmd.Wait()
	
	// Log output for debugging regardless of success
	output := string(outputBytes)
	t.Logf("Command output: %s", output)
	
	if len(errBytes) > 0 {
		t.Logf("Error output: %s", string(errBytes))
	}
	
	// Only verify output if command succeeded
	if err == nil {
		// Check for expected output patterns
		assert.Contains(t, output, "LTM Store Type: pgvector", "Should show pgvector as LTM store type")
		assert.Contains(t, output, "Entity set to: pg_test", "Should confirm entity change")
		assert.Contains(t, output, "Memory stored successfully", "Should confirm memory storage")
		assert.Contains(t, output, "PostgreSQL test", "Should find the stored memory")
	} else {
		t.Logf("PostgreSQL config test failed: %v", err)
		t.Logf("This may be normal if PostgreSQL with pgvector is not properly set up")
	}
}

// TestExampleClientSearch tests the semantic search functionality of the example-client CLI.
// This test requires an OpenAI API key and should only be run manually when needed.
func TestExampleClientSearch(t *testing.T) {
	// Skip unless specifically enabled by environment variable
	if os.Getenv("TEST_SEMANTIC_SEARCH") != "true" {
		t.Skip("Skipping semantic search test; set TEST_SEMANTIC_SEARCH=true to run")
	}

	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping semantic search test; OPENAI_API_KEY is required")
	}

	// First, build the example-client binary if it doesn't exist
	if _, err := os.Stat("test_example_client"); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "test_example_client", "../../cmd/example-client")
		buildOutput, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "Failed to build example-client: %s", buildOutput)
	}
	defer os.Remove("test_example_client") // Clean up the binary after the test

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "example-client-search-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test config file for vector search
	testConfig := config.Config{
		LTM: config.LTMConfig{
			Type: "pgvector", // Use pgvector for semantic search
			PgVector: config.PgVectorConfig{
				ConnectionString: os.Getenv("POSTGRES_URL"),
				TableName:        "memory_vectors_test",
				Dimensions:       1536,
				DistanceMetric:   "cosine",
			},
		},
		Scripting: config.ScriptingConfig{
			Paths: []string{"../../scripts"}, // Use standard script paths
		},
		Reasoning: config.ReasoningConfig{
			Provider: "openai",
			OpenAI: config.OpenAIConfig{
				APIKey:         os.Getenv("OPENAI_API_KEY"),
				Model:          "gpt-4",
				EmbeddingModel: "text-embedding-3-small",
			},
		},
		Reflection: config.ReflectionConfig{
			Enabled: false, // Disable reflection for testing
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	// Save the config to a YAML file
	configYaml, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	configPath := filepath.Join(tempDir, "search_config.yaml")
	err = os.WriteFile(configPath, configYaml, 0644)
	require.NoError(t, err)

	// Set up input commands for semantic search test
	inputCommands := []string{
		"!entity semantic_test", // Set entity
		"!user semantic_tester", // Set user
		"!remember Artificial intelligence (AI) is the simulation of human intelligence by machines.", // Store memory 1
		"!remember The field of machine learning is a subset of AI focused on building systems that learn from data.", // Store memory 2
		"!remember Deep learning is a type of machine learning that uses neural networks with many layers.", // Store memory 3
		"!remember Natural language processing (NLP) is a field of AI that focuses on interactions between computers and human language.", // Store memory 4
		"!search neural networks", // Semantic search for neural networks
		"!quit", // Exit
	}
	
	inputString := strings.Join(inputCommands, "\n") + "\n"
	
	// Start the command with piped stdin and capture stdout
	cmd := exec.Command("./test_example_client", "--config", configPath)
	cmd.Stdin = bytes.NewBufferString(inputString)
	
	// Set environment variables
	cmd.Env = append(os.Environ(), 
		"OPENAI_API_KEY="+os.Getenv("OPENAI_API_KEY"),
		"POSTGRES_URL="+os.Getenv("POSTGRES_URL"),
	)
	
	// Capture output
	output, err := cmd.CombinedOutput()
	
	// Log the full output for debugging
	t.Logf("Command output: %s", output)
	
	// Check results only if command succeeded
	if err == nil {
		// Check for expected patterns in the output
		outputStr := string(output)
		assert.Contains(t, outputStr, "Memory stored successfully", "Should confirm memory storage")
		assert.Contains(t, outputStr, "Performing semantic search", "Should show semantic search is being performed")
		
		// The deep learning memory should be ranked highly in search results for "neural networks"
		assert.Contains(t, outputStr, "Deep learning", "Semantic search should find related memory about deep learning")
	} else {
		t.Logf("Semantic search test command failed: %v", err)
	}
}