package azure

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads Azure ML configuration from a YAML file
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

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.ModelName == "" {
		config.ModelName = "Meta-Llama-3.1-405B-Instruct"
	}

	return &config, nil
}

// SaveConfig saves Azure ML configuration to a YAML file
func SaveConfig(config Config, configPath string) error {
	// Expand relative paths
	if !filepath.IsAbs(configPath) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		configPath = filepath.Join(wd, configPath)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600) // Restrictive permissions for API keys
}
