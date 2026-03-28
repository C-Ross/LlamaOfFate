package openai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearLLMEnvVars temporarily clears LLM credential environment variables and returns a cleanup function
func clearLLMEnvVars(t *testing.T) func() {
	t.Helper()
	originalEndpoint := os.Getenv("AZURE_API_ENDPOINT")
	originalKey := os.Getenv("AZURE_API_KEY")

	_ = os.Unsetenv("AZURE_API_ENDPOINT")
	_ = os.Unsetenv("AZURE_API_KEY")

	return func() {
		if originalEndpoint != "" {
			_ = os.Setenv("AZURE_API_ENDPOINT", originalEndpoint)
		}
		if originalKey != "" {
			_ = os.Setenv("AZURE_API_KEY", originalKey)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	// Clear any environment variables that might interfere
	cleanup := clearLLMEnvVars(t)
	defer cleanup()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "llm-config.yaml")

	configContent := `api_endpoint: "https://test.inference.ai.azure.com"
api_key: "test-api-key-12345"
model_name: "Meta-Llama-3.1-70B-Instruct"
timeout: 45
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Load the config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "https://test.inference.ai.azure.com", config.APIEndpoint)
	assert.Equal(t, "test-api-key-12345", config.APIKey)
	assert.Equal(t, "Meta-Llama-3.1-70B-Instruct", config.ModelName)
	assert.Equal(t, 45, config.Timeout)
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Clear any environment variables that might interfere
	cleanup := clearLLMEnvVars(t)
	defer cleanup()

	// Create a minimal config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "llm-config.yaml")

	configContent := `api_endpoint: "https://test.inference.ai.azure.com"
api_key: "test-api-key-12345"
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Load the config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "https://test.inference.ai.azure.com", config.APIEndpoint)
	assert.Equal(t, "test-api-key-12345", config.APIKey)
	assert.Equal(t, "Meta-Llama-3.1-405B-Instruct", config.ModelName) // Default
	assert.Equal(t, 30, config.Timeout)                               // Default
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "llm-config.yaml")

	configContent := `api_endpoint: "https://file-endpoint.inference.ai.azure.com"
api_key: "file-api-key"
model_name: "Meta-Llama-3.1-70B-Instruct"
timeout: 45
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set environment variables
	originalEndpoint := os.Getenv("AZURE_API_ENDPOINT")
	originalKey := os.Getenv("AZURE_API_KEY")
	defer func() {
		if originalEndpoint != "" {
			_ = os.Setenv("AZURE_API_ENDPOINT", originalEndpoint)
		} else {
			_ = os.Unsetenv("AZURE_API_ENDPOINT")
		}
		if originalKey != "" {
			_ = os.Setenv("AZURE_API_KEY", originalKey)
		} else {
			_ = os.Unsetenv("AZURE_API_KEY")
		}
	}()

	_ = os.Setenv("AZURE_API_ENDPOINT", "https://env-endpoint.inference.ai.azure.com")
	_ = os.Setenv("AZURE_API_KEY", "env-api-key")

	// Load the config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Environment variables should override file values
	assert.Equal(t, "https://env-endpoint.inference.ai.azure.com", config.APIEndpoint)
	assert.Equal(t, "env-api-key", config.APIKey)
	// Other values should remain from file
	assert.Equal(t, "Meta-Llama-3.1-70B-Instruct", config.ModelName)
	assert.Equal(t, 45, config.Timeout)
}

func TestLoadConfigWithPartialEnvironmentVariables(t *testing.T) {
	// Clear any existing environment variables first
	cleanup := clearLLMEnvVars(t)
	defer cleanup()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "llm-config.yaml")

	configContent := `api_endpoint: "https://file-endpoint.inference.ai.azure.com"
api_key: "file-api-key"
model_name: "Meta-Llama-3.1-70B-Instruct"
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set only API_KEY environment variable (endpoint should come from file)
	_ = os.Setenv("AZURE_API_KEY", "env-api-key")

	// Load the config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Only API key should be overridden
	assert.Equal(t, "https://file-endpoint.inference.ai.azure.com", config.APIEndpoint)
	assert.Equal(t, "env-api-key", config.APIKey)
	assert.Equal(t, "Meta-Llama-3.1-70B-Instruct", config.ModelName)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "LLM config not found")
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	invalidContent := `api_endpoint: "https://test.inference.ai.azure.com"
api_key: "test-api-key-12345
model_name: Meta-Llama-3.1-70B-Instruct # Missing closing quote above
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0600)
	require.NoError(t, err)

	// Try to load the config
	config, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestLoadConfigRelativePath(t *testing.T) {
	// Clear any environment variables that might interfere
	cleanup := clearLLMEnvVars(t)
	defer cleanup()

	// Change to temp directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	configContent := `api_endpoint: "https://test.inference.ai.azure.com"
api_key: "test-api-key-12345"
model_name: "Meta-Llama-3.1-70B-Instruct"
`

	err = os.WriteFile(filepath.Join(tempDir, "llm-config.yaml"), []byte(configContent), 0600)
	require.NoError(t, err)

	// Load with relative path
	config, err := LoadConfig("llm-config.yaml")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "https://test.inference.ai.azure.com", config.APIEndpoint)
}

func TestLoadConfigEmptyEndpointValidation(t *testing.T) {
	cleanup := clearLLMEnvVars(t)
	defer cleanup()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "llm-config.yaml")
	configContent := `api_endpoint: ""
api_key: "test-api-key"
model_name: "Meta-Llama-3.1-70B-Instruct"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	config, err := LoadConfig(configPath)
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "api_endpoint is empty")
}

func TestLoadConfigEmptyAPIKeyValidation(t *testing.T) {
	cleanup := clearLLMEnvVars(t)
	defer cleanup()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "llm-config.yaml")
	configContent := `api_endpoint: "https://test.inference.ai.azure.com"
api_key: ""
model_name: "Meta-Llama-3.1-70B-Instruct"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	config, err := LoadConfig(configPath)
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "api_key is empty")
}
