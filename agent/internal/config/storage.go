package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"runtime"
	"syscall"

	"zenoguard-agent/internal/logger"
)

const (
	configPerm = 0600
)

// getConfigPath returns the config file path
// On non-Linux systems, use user home directory
// On Linux, use system directory for root, user directory for non-root
func getConfigPath() string {
	if runtime.GOOS != "linux" {
		// Use user home directory for macOS and other systems
		if u, err := user.Current(); err == nil {
			return u.HomeDir + "/.zenoguard/config.json"
		}
		// Fallback to temp directory
		return os.TempDir() + "/zenoguard/config.json"
	}

	// Linux: check if we're root
	if os.Geteuid() == 0 {
		// Running as root, use system directory
		return "/etc/zenoguard/config.json"
	}

	// Not running as root, use user home directory
	if u, err := user.Current(); err == nil {
		return u.HomeDir + "/.zenoguard/config.json"
	}

	// Last resort: temp directory
	return os.TempDir() + "/zenoguard/config.json"
}

var configPath = getConfigPath()

// getConfigDir returns the config directory path
func getConfigDir() string {
	if runtime.GOOS != "linux" {
		// Use user home directory for macOS and other systems
		if u, err := user.Current(); err == nil {
			return u.HomeDir + "/.zenoguard"
		}
		return os.TempDir() + "/zenoguard"
	}

	// Linux: check if we're root
	if os.Geteuid() == 0 {
		// Running as root, use system directory
		return "/etc/zenoguard"
	}

	// Not running as root, use user home directory
	if u, err := user.Current(); err == nil {
		return u.HomeDir + "/.zenoguard"
	}

	// Last resort: temp directory
	return os.TempDir() + "/zenoguard"
}

// generateKeyFromMachine generates an encryption key from machine characteristics
func generateKeyFromMachine() []byte {
	// Get machine identifiers
	hostname, _ := os.Hostname()
	interfaces, _ := net.Interfaces()

	// Combine identifiers
	data := hostname
	for _, iface := range interfaces {
		if len(iface.HardwareAddr) > 0 {
			data += iface.HardwareAddr.String()
			break // Use first available MAC address
		}
	}

	// If no MAC, add username
	if data == hostname {
		if u, err := user.Current(); err == nil {
			data += u.Username
		}
	}

	// Add OS info
	data += runtime.GOOS + runtime.GOARCH

	// Generate SHA256 hash (32 bytes for AES-256)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// encrypt encrypts data using AES-256-GCM
func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// LoadConfig loads and decrypts the configuration
func LoadConfig() (*Config, error) {
	logger.Info("Loading configuration from " + configPath)

	// Read encrypted config file
	ciphertext, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("Config file does not exist, returning default config")
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Generate decryption key
	key := generateKeyFromMachine()

	// Decrypt
	plaintext, err := decrypt(ciphertext, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	logger.Info("Configuration loaded successfully")
	return &config, nil
}

// SaveConfig encrypts and saves the configuration
func SaveConfig(config *Config) error {
	logger.Info("Saving configuration to " + configPath)

	// Ensure directory exists
	dir := getConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	plaintext, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Generate encryption key
	key := generateKeyFromMachine()

	// Encrypt
	ciphertext, err := encrypt(plaintext, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	// Write with secure permissions
	if err := os.WriteFile(configPath, ciphertext, configPerm); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logger.Info("Configuration saved successfully")
	return nil
}

// ConfigExists checks if config file exists
func ConfigExists() bool {
	info, err := os.Stat(configPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// EnsureSecurePermissions ensures config file has secure permissions
func EnsureSecurePermissions() error {
	if !ConfigExists() {
		return nil
	}

	// Get current file info
	info, err := os.Stat(configPath)
	if err != nil {
		return err
	}

	// Check if permissions are too open
	mode := info.Mode().Perm()
	if mode&077 != 0 {
		logger.Info("Fixing insecure config file permissions")
		return os.Chmod(configPath, configPerm)
	}

	return nil
}

// InitConfigDir initializes the config directory with proper permissions
func InitConfigDir() error {
	dir := getConfigDir()

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory with parent directories
			logger.Info("Creating config directory: " + dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			logger.Info("Config directory created successfully")
			return nil
		}
		return fmt.Errorf("failed to stat config directory: %w", err)
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		return fmt.Errorf("%s exists but is not a directory", dir)
	}

	logger.Info("Using existing config directory: " + dir)
	return nil
}

// CheckConfigSecurity checks if the configuration is secure
func CheckConfigSecurity() error {
	if !ConfigExists() {
		return nil
	}

	info, err := os.Stat(configPath)
	if err != nil {
		return err
	}

	// Check permissions
	mode := info.Mode()
	if mode.Perm() != configPerm {
		return fmt.Errorf("config file has insecure permissions: %v", mode.Perm())
	}

	// On Linux, check ownership
	if runtime.GOOS == "linux" {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}

		// Only root should read/write
		if stat.Uid != 0 {
			return fmt.Errorf("config file should be owned by root")
		}
	}

	return nil
}
