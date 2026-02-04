package collector

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"zenoguard-agent/internal/logger"
)

// HostInfo represents host information
type HostInfo struct {
	Hostname  string `json:"hostname"`
	PublicIP  string `json:"public_ip"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Uptime    uint64 `json:"uptime"`
}

// HostInfoCollector collects host information
type HostInfoCollector struct {
	BaseCollector
	ipAPIEndpoint string
	client        *http.Client
}

// NewHostInfoCollector creates a new host info collector
func NewHostInfoCollector() *HostInfoCollector {
	return &HostInfoCollector{
		BaseCollector: BaseCollector{name: "hostinfo"},
		ipAPIEndpoint: "https://api.ipify.org",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Collect collects host information
func (c *HostInfoCollector) Collect() (interface{}, error) {
	logger.Info("Collecting host information")

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	publicIP, err := c.getPublicIP()
	if err != nil {
		logger.Warn("Failed to get public IP: " + err.Error())
		publicIP = ""
	}

	uptime, err := c.getUptime()
	if err != nil {
		logger.Warn("Failed to get uptime: " + err.Error())
		uptime = 0
	}

	info := HostInfo{
		Hostname:  hostname,
		PublicIP:  publicIP,
		OS:        getOS(),
		Arch:      getArch(),
		Uptime:    uptime,
	}

	logger.Info(fmt.Sprintf("Host info: hostname=%s, ip=%s", hostname, publicIP))
	return info, nil
}

// getPublicIP retrieves the public IP address
func (c *HostInfoCollector) getPublicIP() (string, error) {
	// Try multiple services for redundancy
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ifconfig.me",
		"https://checkip.amazonaws.com",
	}

	var lastErr error
	for _, service := range services {
		ip, err := c.fetchIP(service)
		if err == nil && ip != "" {
			return ip, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("all IP services failed: %w", lastErr)
}

// fetchIP fetches IP from a specific service
func (c *HostInfoCollector) fetchIP(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set user agent
	req.Header.Set("User-Agent", "ZenoGuard-Agent/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}

	// Read response
	buf := make([]byte, 100)
	n, err := resp.Body.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}

	ip := strings.TrimSpace(string(buf[:n]))

	// Validate IP address
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	return ip, nil
}

// getUptime retrieves system uptime in seconds
func (c *HostInfoCollector) getUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	// Parse uptime format: "12345.67 12345.67"
	// First number is uptime in seconds
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid uptime format")
	}

	var uptime float64
	fmt.Sscanf(parts[0], "%f", &uptime)

	return uint64(uptime), nil
}

// getOS returns the operating system name
func getOS() string {
	// Try /etc/os-release first
	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				// Extract value between quotes
				start := strings.Index(line, "\"")
				end := strings.LastIndex(line, "\"")
				if start != -1 && end > start {
					return line[start+1 : end]
				}
			}
		}
	}

	// Fallback to uname
	return "Linux"
}

// getArch returns the system architecture
func getArch() string {
	// Map Go architecture to common names
	archMap := map[string]string{
		"amd64": "x86_64",
		"386":   "i386",
		"arm64": "aarch64",
		"arm":   "armv7l",
	}

	// Get current architecture
	arch := ""
	if data, err := os.ReadFile("/proc/sys/kernel/arch"); err == nil {
		arch = strings.TrimSpace(string(data))
	}

	// Fallback to common mappings
	if arch == "" {
		// Read from uname
		return "x86_64" // Default fallback
	}

	// Normalize architecture name
	for goArch, commonArch := range archMap {
		if strings.Contains(strings.ToLower(arch), strings.ToLower(commonArch)) {
			return commonArch
		}
		if arch == goArch {
			return archMap[arch]
		}
	}

	return arch
}

// GetPrivateIPs returns all private IP addresses
func (c *HostInfoCollector) GetPrivateIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	ips := make([]string, 0)
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}

	return ips, nil
}
