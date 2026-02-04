package collector

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"zenoguard-agent/internal/logger"
)

// SystemLoad represents system load information
type SystemLoad struct {
	Load1  float64 `json:"load1"`  // 1-minute average
	Load5  float64 `json:"load5"`  // 5-minute average
	Load15 float64 `json:"load15"` // 15-minute average
}

// SystemCollector collects system load information
type SystemCollector struct {
	BaseCollector
	loadPath string
}

// NewSystemCollector creates a new system collector
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{
		BaseCollector: BaseCollector{name: "system"},
		loadPath:      "/proc/loadavg",
	}
}

// Collect collects system load information
func (c *SystemCollector) Collect() (interface{}, error) {
	logger.Info("Collecting system load information")

	// macOS uses different method
	if runtime.GOOS == "darwin" {
		return c.collectDarwin()
	}

	// Linux: read from /proc/loadavg
	data, err := exec.Command("cat", c.loadPath).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", c.loadPath, err)
	}

	// Parse /proc/loadavg format: "0.50 0.80 0.60 1/123 4567"
	parts := strings.Fields(string(data))
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid loadavg format: %s", string(data))
	}

	load1, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load1: %w", err)
	}

	load5, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load5: %w", err)
	}

	load15, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load15: %w", err)
	}

	load := SystemLoad{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}

	logger.Info(fmt.Sprintf("System load: %.2f %.2f %.2f", load1, load5, load15))
	return load, nil
}

// collectDarwin collects system load on macOS using sysctl
func (c *SystemCollector) collectDarwin() (interface{}, error) {
	// Use sysctl to get load averages
	// Output format: "0.50 0.80 0.60"
	data, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute sysctl: %w", err)
	}

	// Parse the output, it returns: { 1 min 5 min 15 min }
	output := strings.TrimSpace(string(data))
	output = strings.TrimPrefix(output, "{")
	output = strings.TrimSuffix(output, "}")

	parts := strings.Fields(output)
	if len(parts) < 3 {
		// Try alternative format: just three numbers separated by spaces
		data, err = exec.Command("uptime").Output()
		if err != nil {
			return nil, fmt.Errorf("failed to execute uptime: %w", err)
		}
		// Parse uptime output: "load average: 0.50, 0.80, 0.60"
		uptimeOutput := string(data)
		for _, line := range strings.Split(uptimeOutput, "\n") {
			if strings.Contains(line, "load average") {
				// Extract the load averages
				parts := strings.Split(line, "load average:")[1]
				parts = strings.TrimSpace(parts)
				loadParts := strings.Split(parts, ",")
				if len(loadParts) == 3 {
					load1, _ := strconv.ParseFloat(strings.TrimSpace(loadParts[0]), 64)
					load5, _ := strconv.ParseFloat(strings.TrimSpace(loadParts[1]), 64)
					load15, _ := strconv.ParseFloat(strings.TrimSpace(loadParts[2]), 64)

					load := SystemLoad{
						Load1:  load1,
						Load5:  load5,
						Load15: load15,
					}

					logger.Info(fmt.Sprintf("System load: %.2f %.2f %.2f", load1, load5, load15))
					return load, nil
				}
			}
		}
		return nil, fmt.Errorf("invalid loadavg format from uptime")
	}

	load1, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load1: %w", err)
	}

	load5, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load5: %w", err)
	}

	load15, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load15: %w", err)
	}

	load := SystemLoad{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}

	logger.Info(fmt.Sprintf("System load: %.2f %.2f %.2f", load1, load5, load15))
	return load, nil
}
