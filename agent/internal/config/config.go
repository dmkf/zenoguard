package config

import (
	"os"
	"strconv"
)

// Config holds the agent configuration
type Config struct {
	ServerURL      string `json:"server_url"`
	Token          string `json:"token"`
	ReportInterval int    `json:"report_interval"`
}

// DefaultConfig returns default configuration
// It checks environment variables first
func DefaultConfig() *Config {
	config := &Config{
		ReportInterval: 300, // Default 5 minutes (300 seconds)
	}

	// Load from environment variables
	if serverURL := os.Getenv("ZENOGUARD_SERVER_URL"); serverURL != "" {
		config.ServerURL = serverURL
	}
	if token := os.Getenv("ZENOGUARD_TOKEN"); token != "" {
		config.Token = token
	}
	if hostname := os.Getenv("ZENOGUARD_HOSTNAME"); hostname != "" {
		// Hostname is not stored in config but used for reporting
		// We'll handle it separately
	}
	if interval := os.Getenv("ZENOGUARD_REPORT_INTERVAL"); interval != "" {
		if iv, err := strconv.Atoi(interval); err == nil {
			config.ReportInterval = iv
		}
	}

	return config
}
