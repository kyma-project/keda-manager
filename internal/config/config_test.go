package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadLogConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `logLevel: debug
logFormat: text
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Read the config file
	cfg, err := LoadLogConfig(configFile)
	require.NoError(t, err)

	// Verify the values
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "text", cfg.LogFormat)
}

func TestLoadLogConfig_InvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `logLevel: debug
logFormat: text
invalid: [unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Read the config file - should error
	_, err = LoadLogConfig(configFile)
	require.Error(t, err)
}

func TestLoadLogConfig_NonExistent(t *testing.T) {
	// Try to read a non-existent file
	cfg, err := LoadLogConfig("/non/existent/config.yaml")
	require.Error(t, err)
	require.Empty(t, cfg.LogLevel)
}

func TestGetConfig_EnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("LOG_LEVEL", "warn")
	os.Setenv("LOG_FORMAT", "text")
	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
	}()

	// Get config from environment
	cfg, err := GetConfig("")
	require.NoError(t, err)

	// Verify the values from environment
	require.Equal(t, "warn", cfg.LogLevel)
	require.Equal(t, "text", cfg.LogFormat)
}
