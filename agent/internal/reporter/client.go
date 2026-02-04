package reporter

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"zenoguard-agent/internal/logger"
)

// ReportData represents data to be reported
type ReportData struct {
	Hostname       string              `json:"hostname"`
	SSHLogins      []SSHLoginReport    `json:"ssh_logins"`
	SystemLoad     SystemLoadReport    `json:"system_load"`
	NetworkTraffic NetworkTrafficReport `json:"network_traffic"`
	PublicIP       string              `json:"public_ip"`
}

// SSHLoginReport represents SSH login info for reporting
type SSHLoginReport struct {
	User            string `json:"user"`
	IP              string `json:"ip"`
	Time            string `json:"time"`
	Method          string `json:"method"`
	Success         bool   `json:"success"`
	Port            int    `json:"port"`
	Protocol        string `json:"protocol"`
	SessionDuration int64  `json:"session_duration"` // seconds
	IsActive        bool   `json:"is_active"`        // currently logged in
}

// SystemLoadReport represents system load for reporting
type SystemLoadReport struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// NetworkTrafficReport represents network traffic for reporting
type NetworkTrafficReport struct {
	Interface    string                 `json:"interface"`
	Samples      []TrafficSampleReport `json:"samples"`
	TotalInBytes uint64                 `json:"total_in_bytes"`
	TotalOutBytes uint64                 `json:"total_out_bytes"`
	SampleCount   int                    `json:"sample_count"`
}

// TrafficSampleReport represents a traffic sample for reporting
type TrafficSampleReport struct {
	Timestamp        string  `json:"timestamp"`
	InBytes          uint64  `json:"in_bytes"`
	OutBytes         uint64  `json:"out_bytes"`
	TotalBytes       uint64  `json:"total_bytes"`
	TimeDeltaSeconds float64 `json:"time_delta_seconds"`
}

// ReportResponse represents server response
type ReportResponse struct {
	Success        bool `json:"success"`
	ReportInterval int  `json:"report_interval"`
}

// Client represents an HTTPS client for reporting
type Client struct {
	serverURL string
	token     string
	httpClient *http.Client
}

// NewClient creates a new HTTPS client
func NewClient(serverURL, token string) *Client {
	// Normalize serverURL: remove trailing /api/ if present
	serverURL = strings.TrimSuffix(serverURL, "/")
	if strings.HasSuffix(serverURL, "/api") {
		serverURL = strings.TrimSuffix(serverURL, "/api")
	}

	// Create HTTP client with timeout and TLS config
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// Allow self-signed certificates for flexibility
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS12,
			},
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
		},
	}

	return &Client{
		serverURL:  serverURL,
		token:      token,
		httpClient: client,
	}
}

// Report sends data to the server
func (c *Client) Report(data *ReportData) (*ReportResponse, error) {
	logger.Info("Reporting to server: " + c.serverURL)
	logger.Info(fmt.Sprintf("Sending %d SSH log entries", len(data.SSHLogins)))

	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal report data: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/agent/report", c.serverURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "ZenoGuard-Agent/1.0")

	// Send request
	logger.Debug("Sending request to: " + url)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	logger.Debug(fmt.Sprintf("Response status: %d", resp.StatusCode))

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response ReportResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Info(fmt.Sprintf("Report successful. New interval: %d seconds", response.ReportInterval))
	return &response, nil
}

// TestConnection tests the connection to the server
func (c *Client) TestConnection() error {
	logger.Info("Testing connection to server: " + c.serverURL)

	// Send a minimal report to test connection
	url := fmt.Sprintf("%s/api/agent/report", c.serverURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	// Unauthorized is expected for test request with empty data
	// Just check if we can reach the server
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
		logger.Info("Connection test successful (server is reachable)")
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// Close closes the HTTP client
func (c *Client) Close() {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}
