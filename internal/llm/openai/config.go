package openai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const appConfigDirName = "LlamaOfFate"

// LoadConfig loads LLM configuration from a YAML file and applies environment variable overrides.
// Environment variables take precedence over values in the YAML file:
// - AZURE_API_ENDPOINT: Overrides api_endpoint
// - AZURE_API_KEY: Overrides api_key
//
// For relative paths, this lookup order is used:
// 1. Current working directory
// 2. Executable directory
// 3. OS user config directory (e.g. ~/.config/LlamaOfFate or %AppData%\\LlamaOfFate)
func LoadConfig(configPath string) (*Config, error) {
	resolvedPath, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("read LLM config %s: %w", resolvedPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse LLM config %s: %w", resolvedPath, err)
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

	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid LLM config %s: %w", resolvedPath, err)
	}

	return &config, nil
}

// ValidateConfig performs basic required-field validation.
func ValidateConfig(config Config) error {
	if strings.TrimSpace(config.APIEndpoint) == "" {
		return fmt.Errorf("api_endpoint is empty; set it in config or AZURE_API_ENDPOINT")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("api_key is empty; set it in config or AZURE_API_KEY")
	}
	return nil
}

func resolveConfigPath(configPath string) (string, error) {
	if filepath.IsAbs(configPath) {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("LLM config not found at %s", configPath)
	}

	candidates := []string{}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, configPath))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, configPath))
	}

	if userCfgDir, err := os.UserConfigDir(); err == nil {
		root := filepath.Join(userCfgDir, appConfigDirName)
		candidates = append(candidates,
			filepath.Join(root, configPath),
			filepath.Join(root, filepath.Base(configPath)),
		)
	}

	seen := map[string]struct{}{}
	attempted := []string{}
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		attempted = append(attempted, clean)
		if _, err := os.Stat(clean); err == nil {
			return clean, nil
		}
	}

	if len(attempted) == 0 {
		return "", fmt.Errorf("LLM config not found; no candidate paths generated for %q", configPath)
	}
	return "", fmt.Errorf("LLM config %q not found; checked: %s", configPath, strings.Join(attempted, ", "))
}
