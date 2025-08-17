package azure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "azure-config.yaml")

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
	// Create a minimal config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "azure-config.yaml")

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

func TestLoadConfigFileNotFound(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "no such file or directory")
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

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "azure-config.yaml")

	config := Config{
		APIEndpoint: "https://my-endpoint.inference.ai.azure.com",
		APIKey:      "my-secret-api-key",
		ModelName:   "Meta-Llama-3.1-8B-Instruct",
		Timeout:     60,
	}

	// Save the config
	err := SaveConfig(config, configPath)
	require.NoError(t, err)

	// Verify the file was created
	assert.FileExists(t, configPath)

	// Verify file permissions are restrictive
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Load the config back and verify
	loadedConfig, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, config.APIEndpoint, loadedConfig.APIEndpoint)
	assert.Equal(t, config.APIKey, loadedConfig.APIKey)
	assert.Equal(t, config.ModelName, loadedConfig.ModelName)
	assert.Equal(t, config.Timeout, loadedConfig.Timeout)
}

func TestSaveConfigCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subdir", "azure-config.yaml")

	config := Config{
		APIEndpoint: "https://test.inference.ai.azure.com",
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
		Timeout:     30,
	}

	// Save the config (should create the subdirectory)
	err := SaveConfig(config, configPath)
	require.NoError(t, err)

	// Verify the file was created
	assert.FileExists(t, configPath)

	// Verify the directory was created
	assert.DirExists(t, filepath.Dir(configPath))
}

func TestLoadConfigRelativePath(t *testing.T) {
	// Change to temp directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		os.Chdir(originalWd)
	}()

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create config with relative path
	configContent := `api_endpoint: "https://test.inference.ai.azure.com"
api_key: "test-api-key-12345"
model_name: "Meta-Llama-3.1-70B-Instruct"
`

	err = os.WriteFile("azure-config.yaml", []byte(configContent), 0600)
	require.NoError(t, err)

	// Load the config using relative path
	config, err := LoadConfig("azure-config.yaml")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "https://test.inference.ai.azure.com", config.APIEndpoint)
	assert.Equal(t, "test-api-key-12345", config.APIKey)
	assert.Equal(t, "Meta-Llama-3.1-70B-Instruct", config.ModelName)
}
