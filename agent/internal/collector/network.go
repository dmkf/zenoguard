package collector

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"zenoguard-agent/internal/logger"
)

// NetworkInterface represents network interface statistics
type NetworkInterface struct {
	Name      string  `json:"name"`
	InBytes   uint64  `json:"in_bytes"`
	OutBytes  uint64  `json:"out_bytes"`
	InPackets uint64  `json:"in_packets"`
	OutPackets uint64 `json:"out_packets"`
	InErrors  uint64  `json:"in_errors"`
	OutErrors uint64 `json:"out_errors"`
}

// TrafficSample represents a single traffic sample
type TrafficSample struct {
	Timestamp        time.Time `json:"timestamp"`
	InBytes          uint64    `json:"in_bytes"`
	OutBytes         uint64    `json:"out_bytes"`
	TotalBytes       uint64    `json:"total_bytes"`
	TimeDeltaSeconds float64  `json:"time_delta_seconds"`
}

// NetworkTraffic represents network traffic with multiple samples
type NetworkTraffic struct {
	Interface    string          `json:"interface"`
	Samples      []TrafficSample `json:"samples"`          // Multiple 5-minute samples
	TotalInBytes uint64          `json:"total_in_bytes"`    // Total across all samples
	TotalOutBytes uint64          `json:"total_out_bytes"`   // Total across all samples
	SampleCount   int             `json:"sample_count"`      // Number of samples
}

// NetworkCollector collects network traffic information
type NetworkCollector struct {
	BaseCollector
	devPath            string
	publicIface        string
	lastTotalInBytes   uint64        // Last total in bytes (for delta calculation)
	lastTotalOutBytes  uint64        // Last total out bytes (for delta calculation)
	lastTimestamp      time.Time     // Last collection time
	sampleInterval     time.Duration // Sample interval (5 minutes)
	samples            []TrafficSample // Collected samples
	lastSampleTime     time.Time     // Last sample time
	mu                 sync.Mutex    // Protect samples slice
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		BaseCollector:  BaseCollector{name: "network"},
		devPath:        "/proc/net/dev",
		sampleInterval: 5 * time.Minute, // 5 minutes
	}
}

// Collect collects network traffic information
func (c *NetworkCollector) Collect() (interface{}, error) {
	logger.Info("Collecting network traffic information")

	// macOS uses different method
	if runtime.GOOS == "darwin" {
		return c.collectDarwin()
	}

	// Linux: read from /proc/net/dev
	data, err := os.ReadFile(c.devPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", c.devPath, err)
	}

	// Find the public interface and get current totals
	publicIface, currentStats, err := c.parseNetworkStats(string(data))
	if err != nil {
		return nil, err
	}

	// Collect sample
	c.collectSample(publicIface, currentStats.InBytes, currentStats.OutBytes)

	// Check if we have any samples to report
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.samples) == 0 {
		// No samples yet (first report), return nil to exclude network traffic
		logger.Info("No network traffic samples available yet, skipping")
		return nil, nil
	}

	result := &NetworkTraffic{
		Interface:    publicIface,
		Samples:      make([]TrafficSample, len(c.samples)),
		TotalInBytes: 0,
		TotalOutBytes: 0,
		SampleCount:   len(c.samples),
	}

	// Copy samples and calculate totals
	// Calculate average rate (bytes per second) over the 5-minute period
	var totalInRate, totalOutRate uint64
	totalTimeSeconds := 0.0

	for i, sample := range c.samples {
		result.Samples[i] = sample

		// Calculate rate for this sample: bytes / time_delta
		if sample.TimeDeltaSeconds > 0 {
			inRate := uint64(float64(sample.InBytes) / sample.TimeDeltaSeconds)
			outRate := uint64(float64(sample.OutBytes) / sample.TimeDeltaSeconds)

			// Add to totals (will be averaged later)
			totalInRate += inRate * uint64(sample.TimeDeltaSeconds)
			totalOutRate += outRate * uint64(sample.TimeDeltaSeconds)
			totalTimeSeconds += sample.TimeDeltaSeconds
		}
	}

	// Calculate average rate over the entire period
	if totalTimeSeconds > 0 {
		result.TotalInBytes = totalInRate / uint64(totalTimeSeconds)  // Average in bytes/sec
		result.TotalOutBytes = totalOutRate / uint64(totalTimeSeconds) // Average out bytes/sec
	}

	logger.Info(fmt.Sprintf("Network traffic: avg in rate=%d bytes/sec, avg out rate=%d bytes/sec (period=%.1fs)",
		result.TotalInBytes, result.TotalOutBytes, totalTimeSeconds))

	return result, nil
}

// collectSample collects a traffic sample
func (c *NetworkCollector) collectSample(iface string, inBytes, outBytes uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Skip first sample (no delta available)
	if c.lastTimestamp.IsZero() || c.lastTotalInBytes == 0 {
		// Save current totals for next sample, but don't create a sample yet
		c.lastTotalInBytes = inBytes
		c.lastTotalOutBytes = outBytes
		c.lastTimestamp = now
		c.lastSampleTime = now
		c.publicIface = iface

		logger.Info(fmt.Sprintf("Network sample %s: initializing (total in=%d, out=%d)",
			iface, inBytes, outBytes))
		return
	}

	// Calculate delta since last collection
	timeDelta := now.Sub(c.lastTimestamp).Seconds()

	var sample TrafficSample
	sample.Timestamp = now

	if inBytes >= c.lastTotalInBytes {
		sample.InBytes = inBytes - c.lastTotalInBytes
	}

	if outBytes >= c.lastTotalOutBytes {
		sample.OutBytes = outBytes - c.lastTotalOutBytes
	}

	sample.TotalBytes = sample.InBytes + sample.OutBytes
	sample.TimeDeltaSeconds = timeDelta

	// Only add sample if there's actual traffic
	if sample.InBytes > 0 || sample.OutBytes > 0 {
		c.samples = append(c.samples, sample)
		logger.Info(fmt.Sprintf("Network sample %s: delta_in=%d, delta_out=%d, time=%.1fs",
			iface, sample.InBytes, sample.OutBytes, timeDelta))
	}

	// Save current totals for next sample
	c.lastTotalInBytes = inBytes
	c.lastTotalOutBytes = outBytes
	c.lastTimestamp = now
	c.lastSampleTime = now
	c.publicIface = iface
}

// ClearSamples clears all collected samples (called after report)
func (c *NetworkCollector) ClearSamples() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.samples = nil
}

// ShouldSample returns true if it's time to collect a new sample
func (c *NetworkCollector) ShouldSample() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastSampleTime.IsZero() {
		return true
	}

	return time.Since(c.lastSampleTime) >= c.sampleInterval
}

// parseNetworkStats parses /proc/net/dev
func (c *NetworkCollector) parseNetworkStats(data string) (string, *NetworkInterface, error) {
	lines := strings.Split(data, "\n")

	var publicIface string
	var maxBytes uint64
	interfaces := make(map[string]*NetworkInterface)

	// Skip header lines
	for i, line := range lines {
		if i < 2 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse interface line format: "eth0: 12345 678 9 0 ..."
		parts := strings.Fields(line)
		if len(parts) < 17 {
			continue
		}

		// Interface name is before colon
		colonIdx := strings.Index(parts[0], ":")
		if colonIdx == -1 {
			continue
		}

		name := parts[0][:colonIdx]

		// Skip localhost and virtual interfaces
		if name == "lo" || strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") {
			continue
		}

		// Parse statistics
		inBytes, _ := strconv.ParseUint(parts[1], 10, 64)
		outBytes, _ := strconv.ParseUint(parts[9], 10, 64)
		inPackets, _ := strconv.ParseUint(parts[2], 10, 64)
		outPackets, _ := strconv.ParseUint(parts[10], 10, 64)
		inErrors, _ := strconv.ParseUint(parts[3], 10, 64)
		outErrors, _ := strconv.ParseUint(parts[11], 10, 64)

		iface := &NetworkInterface{
			Name:      name,
			InBytes:   inBytes,
			OutBytes:  outBytes,
			InPackets: inPackets,
			OutPackets: outPackets,
			InErrors:  inErrors,
			OutErrors: outErrors,
		}

		interfaces[name] = iface

		// Find interface with most traffic (likely the public interface)
		totalBytes := inBytes + outBytes
		if totalBytes > maxBytes {
			maxBytes = totalBytes
			publicIface = name
		}
	}

	if publicIface == "" {
		// Fallback to first available interface
		for name := range interfaces {
			publicIface = name
			break
		}
	}

	if publicIface == "" {
		return "", nil, fmt.Errorf("no valid network interface found")
	}

	return publicIface, interfaces[publicIface], nil
}

// collectDarwin collects network traffic on macOS using netstat
func (c *NetworkCollector) collectDarwin() (interface{}, error) {
	// Use netstat -i -b to get network interface statistics in bytes
	data, err := exec.Command("netstat", "-i", "-b").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute netstat: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	var publicIface string
	var maxBytes uint64
	interfaces := make(map[string]*NetworkInterface)

	// Parse each line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		name := fields[0]

		// Skip localhost and virtual interfaces
		if name == "lo0" || strings.HasPrefix(name, "lo") ||
			strings.HasPrefix(name, "gif") || strings.HasPrefix(name, "stf") {
			continue
		}

		// Parse netstat -Ib format
		inBytes, _ := strconv.ParseUint(fields[6], 10, 64)   // Ibytes
		outBytes, _ := strconv.ParseUint(fields[9], 10, 64)  // Obytes

		iface := &NetworkInterface{
			Name:      name,
			InBytes:   inBytes,
			OutBytes:  outBytes,
			InPackets: 0,
			OutPackets: 0,
			InErrors:   0,
			OutErrors:  0,
		}

		interfaces[name] = iface

		// Find interface with most traffic (likely the public interface)
		totalBytes := inBytes + outBytes
		if totalBytes > maxBytes {
			maxBytes = totalBytes
			publicIface = name
		}
	}

	// Fallback: try to get the default gateway interface
	if publicIface == "" {
		defaultRoute, err := exec.Command("route", "-n", "get", "default").Output()
		if err == nil {
			routeParts := strings.Fields(string(defaultRoute))
			if len(routeParts) > 0 {
				candidateIface := routeParts[len(routeParts)-1]
				// Verify this interface exists in our map
				if _, exists := interfaces[candidateIface]; exists {
					publicIface = candidateIface
				}
			}
		}
	}

	// Last fallback: use first non-loopback interface
	if publicIface == "" {
		for name := range interfaces {
			publicIface = name
			break
		}
	}

	if publicIface == "" {
		return nil, fmt.Errorf("no valid network interface found")
	}

	iface, exists := interfaces[publicIface]
	if !exists {
		return nil, fmt.Errorf("interface %s not found in interfaces map", publicIface)
	}

	// Collect sample
	c.collectSample(publicIface, iface.InBytes, iface.OutBytes)

	// Check if we have any samples to report
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.samples) == 0 {
		// No samples yet (first report), return nil to exclude network traffic
		logger.Info("No network traffic samples available yet, skipping")
		return nil, nil
	}

	result := &NetworkTraffic{
		Interface:    publicIface,
		Samples:      make([]TrafficSample, len(c.samples)),
		TotalInBytes: 0,
		TotalOutBytes: 0,
		SampleCount:   len(c.samples),
	}

	// Copy samples and calculate totals
	// Calculate average rate (bytes per second) over the 5-minute period
	var totalInRate, totalOutRate uint64
	totalTimeSeconds := 0.0

	for i, sample := range c.samples {
		result.Samples[i] = sample

		// Calculate rate for this sample: bytes / time_delta
		if sample.TimeDeltaSeconds > 0 {
			inRate := uint64(float64(sample.InBytes) / sample.TimeDeltaSeconds)
			outRate := uint64(float64(sample.OutBytes) / sample.TimeDeltaSeconds)

			// Add to totals (will be averaged later)
			totalInRate += inRate * uint64(sample.TimeDeltaSeconds)
			totalOutRate += outRate * uint64(sample.TimeDeltaSeconds)
			totalTimeSeconds += sample.TimeDeltaSeconds
		}
	}

	// Calculate average rate over the entire period
	if totalTimeSeconds > 0 {
		result.TotalInBytes = totalInRate / uint64(totalTimeSeconds)  // Average in bytes/sec
		result.TotalOutBytes = totalOutRate / uint64(totalTimeSeconds) // Average out bytes/sec
	}

	logger.Info(fmt.Sprintf("Network traffic: avg in rate=%d bytes/sec, avg out rate=%d bytes/sec (period=%.1fs)",
		result.TotalInBytes, result.TotalOutBytes, totalTimeSeconds))

	return result, nil
}
