package reporter

import (
	"fmt"
	"time"

	"zenoguard-agent/internal/collector"
	"zenoguard-agent/internal/logger"
)

const (
	// MaxRetries is the maximum number of retry attempts
	MaxRetries = 5
	// InitialRetryDelay is the initial delay before first retry
	InitialRetryDelay = 5 * time.Second
	// MaxRetryDelay is the maximum delay between retries
	MaxRetryDelay = 60 * time.Second
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier = 2
)

// Reporter handles data collection and reporting
type Reporter struct {
	client         *Client
	config         *Config
	collectors     []collector.Collector
	reportInterval time.Duration
	stopChan       chan struct{}
	intervalUpdate chan time.Duration
}

// Config represents reporter configuration
type Config struct {
	ServerURL      string
	Token          string
	ReportInterval int // seconds
}

// NewReporter creates a new reporter
func NewReporter(config *Config) *Reporter {
	// Create HTTP client
	client := NewClient(config.ServerURL, config.Token)

	// Initialize collectors
	collectors := []collector.Collector{
		collector.NewSSHCollector(),
		collector.NewSystemCollector(),
		collector.NewNetworkCollector(),
		collector.NewHostInfoCollector(),
	}

	return &Reporter{
		client:         client,
		config:         config,
		collectors:     collectors,
		stopChan:       make(chan struct{}),
		intervalUpdate: make(chan time.Duration, 1),
	}
}

// Start starts the reporting loop
func (r *Reporter) Start() error {
	logger.Info("Starting reporter with interval: " + fmt.Sprint(r.config.ReportInterval) + " seconds")

	// Convert interval to duration
	interval := time.Duration(r.config.ReportInterval) * time.Second

	// Start background sampler (every 5 minutes)
	sampleTicker := time.NewTicker(5 * time.Minute)
	defer sampleTicker.Stop()
	go func() {
		for {
			select {
			case <-sampleTicker.C:
				r.collectNetworkSample()
			case <-r.stopChan:
				return
			}
		}
	}()

	// Set up report ticker
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Report immediately on start
	if err := r.report(); err != nil {
		logger.Error("Initial report failed: " + err.Error())
	}

	// Report loop with dynamic interval support
	for {
		select {
		case newInterval := <-r.intervalUpdate:
			// Update interval
			logger.Info(fmt.Sprintf("Updating ticker interval: %v -> %v", interval, newInterval))
			interval = newInterval
			ticker.Reset(interval)
		case <-ticker.C:
			if err := r.report(); err != nil {
				logger.Error("Report failed: " + err.Error())
			}
		case <-r.stopChan:
			logger.Info("Reporter stopped")
			return nil
		}
	}
}

// Stop stops the reporter
func (r *Reporter) Stop() {
	logger.Info("Stopping reporter...")
	close(r.stopChan)
}

// collectNetworkSample collects a network traffic sample
func (r *Reporter) collectNetworkSample() {
	for _, col := range r.collectors {
		if nc, ok := col.(*collector.NetworkCollector); ok {
			nc.Collect() // This will add a sample to the buffer
			logger.Info("Collected network traffic sample (5-minute interval)")
			return
		}
	}
}

// report performs a single report with retry logic
func (r *Reporter) report() error {
	var lastErr error
	delay := InitialRetryDelay

	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Info(fmt.Sprintf("Retry attempt %d/%d after %v", attempt+1, MaxRetries, delay))
			time.Sleep(delay)
		}

		// Collect data
		data, err := r.collectData()
		if err != nil {
			lastErr = err
			logger.Error("Failed to collect data: " + err.Error())
			continue
		}

		// Send report
		response, err := r.client.Report(data)
		if err != nil {
			lastErr = err

			// Check if token is invalid
			if fmt.Sprintf("%v", err) == "unauthorized: invalid token" {
				logger.Error("Invalid token - will retry for 1 minute then exit")
				time.Sleep(1 * time.Minute)
				logger.Fatal("Invalid token, exiting")
			}

			logger.Error("Failed to send report: " + err.Error())

			// Exponential backoff
			delay = time.Duration(float64(delay) * BackoffMultiplier)
			if delay > MaxRetryDelay {
				delay = MaxRetryDelay
			}
			continue
		}

		// Update report interval if server returned a new value
		if response != nil && response.ReportInterval > 0 {
			newInterval := response.ReportInterval
			if newInterval != r.config.ReportInterval {
				logger.Info(fmt.Sprintf("Server requested new interval: %d -> %d seconds",
					r.config.ReportInterval, newInterval))
				r.config.ReportInterval = newInterval
				// Send update to ticker (non-blocking)
				select {
				case r.intervalUpdate <- time.Duration(newInterval) * time.Second:
					logger.Info("Interval update signal sent")
				default:
					logger.Warn("Interval update channel full, skipping")
				}
			}
		}

		// Clear network samples after successful report
		for _, col := range r.collectors {
			if nc, ok := col.(*collector.NetworkCollector); ok {
				nc.ClearSamples()
				logger.Info("Cleared network traffic samples after successful report")
				break
			}
		}

		// Success
		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// collectData collects data from all collectors
func (r *Reporter) collectData() (*ReportData, error) {
	logger.Info("Collecting data from all collectors")

	data := &ReportData{}

	// Collect from each collector
	for _, col := range r.collectors {
		result, err := col.Collect()
		if err != nil {
			logger.Warn("Collector " + col.Name() + " failed: " + err.Error())
			continue
		}

		// Skip nil results (collector has no data to report)
		if result == nil {
			logger.Info("Collector " + col.Name() + " has no data to report, skipping")
			continue
		}

		// Type switch to assign to appropriate field
		switch v := result.(type) {
		case []collector.SSHLogin:
			logger.Info("SSH collector returned " + fmt.Sprint(len(v)) + " logins")
			data.SSHLogins = convertSSHLogins(v)
			logger.Info("Converted to " + fmt.Sprint(len(data.SSHLogins)) + " report entries")
		case collector.SystemLoad:
			data.SystemLoad = SystemLoadReport{
				Load1:  v.Load1,
				Load5:  v.Load5,
				Load15: v.Load15,
			}
		case *collector.SystemLoad:
			data.SystemLoad = SystemLoadReport{
				Load1:  v.Load1,
				Load5:  v.Load5,
				Load15: v.Load15,
			}
		case *collector.NetworkTraffic:
			// Convert samples to report format
			samples := make([]TrafficSampleReport, len(v.Samples))
			for i, s := range v.Samples {
				samples[i] = TrafficSampleReport{
					Timestamp:        s.Timestamp.Format(time.RFC3339),
					InBytes:          s.InBytes,
					OutBytes:         s.OutBytes,
					TotalBytes:       s.TotalBytes,
					TimeDeltaSeconds: s.TimeDeltaSeconds,
				}
			}

			data.NetworkTraffic = NetworkTrafficReport{
				Interface:    v.Interface,
				Samples:      samples,
				TotalInBytes: v.TotalInBytes,
				TotalOutBytes: v.TotalOutBytes,
				SampleCount:   v.SampleCount,
			}
		case collector.HostInfo:
			data.Hostname = v.Hostname
			data.PublicIP = v.PublicIP
		case *collector.HostInfo:
			data.Hostname = v.Hostname
			data.PublicIP = v.PublicIP
		default:
			logger.Warn("Unknown collector result type from " + col.Name())
		}
	}

	logger.Info("Data collection completed")
	return data, nil
}

// convertSSHLogins converts SSH logins from collector format to report format
func convertSSHLogins(logins []collector.SSHLogin) []SSHLoginReport {
	report := make([]SSHLoginReport, len(logins))
	for i, login := range logins {
		report[i] = SSHLoginReport{
			User:            login.User,
			IP:              login.IP,
			Time:            login.Time,
			Method:          login.Method,
			Success:         login.Success,
			Port:            login.Port,
			Protocol:        login.Protocol,
			SessionDuration: login.SessionDuration,
			IsActive:        login.IsActive,
		}
	}
	return report
}

// UpdateConfig updates the reporter configuration
func (r *Reporter) UpdateConfig(config *Config) {
	r.config = config
	logger.Info("Reporter configuration updated")
}

// GetClient returns the HTTP client
func (r *Reporter) GetClient() *Client {
	return r.client
}

// TestConnection tests the connection to the server
func (r *Reporter) TestConnection() error {
	return r.client.TestConnection()
}
