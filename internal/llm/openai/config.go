package openai

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads LLM configuration from a YAML file and applies environment variable overrides.
// Environment variables take precedence over values in the YAML file:
// - AZURE_API_ENDPOINT: Overrides api_endpoint
// - AZURE_API_KEY: Overrides api_key
func LoadConfig(configPath string) (*Config, error) {
	// Expand relative paths
	if !filepath.IsAbs(configPath) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(wd, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Override with environment variables if present
	if endpoint := os.Getenv("AZURE_API_ENDPOINT"); endpoint != "" {
		config.APIEndpoint = endpoint
	}
	if apiKey := os.Getenv("AZURE_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.ModelName == "" {
		config.ModelName = "Meta-Llama-3.1-405B-Instruct"
	}

	return &config, nil
}
