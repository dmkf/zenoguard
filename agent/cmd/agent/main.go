package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"zenoguard-agent/internal/config"
	"zenoguard-agent/internal/daemon"
	"zenoguard-agent/internal/logger"
	"zenoguard-agent/internal/reporter"
)

var (
	version = "1.0.0"
	commit  = "unknown"
	date    = "unknown"
)

const (
	defaultLogPath = "/var/log/zenoguard/agent.log"
)

func main() {
	// Command-line flags
	serverURL := flag.String("server", "", "Server URL (e.g., https://monitor.example.com)")
	token := flag.String("token", "", "Authentication token")
	configFlag := flag.Bool("config", false, "Configure the agent")
	daemonFlag := flag.Bool("daemon", false, "Run as daemon")
	stopFlag := flag.Bool("stop", false, "Stop the daemon")
	statusFlag := flag.Bool("status", false, "Show daemon status")
	showVersion := flag.Bool("version", false, "Show version information")
	logPath := flag.String("log", defaultLogPath, "Log file path")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("ZenoGuard Agent v%s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Initialize logger
	logLevelEnum := parseLogLevel(*logLevel)
	if err := logger.Init(*logPath, logLevelEnum); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Handle stop command
	if *stopFlag {
		if err := daemon.Stop(); err != nil {
			logger.Error("Failed to stop daemon: " + err.Error())
			os.Exit(1)
		}
		logger.Info("Daemon stopped successfully")
		os.Exit(0)
	}

	// Handle status command
	if *statusFlag {
		running, pid, err := daemon.Status()
		if err != nil {
			logger.Error("Failed to get status: " + err.Error())
			os.Exit(1)
		}
		if running {
			fmt.Printf("ZenoGuard Agent is running (PID: %d)\n", pid)
			os.Exit(0)
		} else {
			fmt.Println("ZenoGuard Agent is not running")
			os.Exit(1)
		}
	}

	// Initialize config directory
	if err := config.InitConfigDir(); err != nil {
		logger.Fatal("Failed to initialize config directory: " + err.Error())
	}

	// Check config security
	if err := config.EnsureSecurePermissions(); err != nil {
		logger.Warn("Config security issue: " + err.Error())
	}

	// Handle config command
	if *configFlag {
		if err := configureAgent(*serverURL, *token); err != nil {
			logger.Fatal("Configuration failed: " + err.Error())
		}
		logger.Info("Configuration saved successfully")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration: " + err.Error())
	}

	// Check if config is empty (first run)
	if !config.ConfigExists() || (cfg.ServerURL == "" && cfg.Token == "") {
		// If neither config file nor environment variables are set, require configuration
		if *serverURL == "" && *token == "" && (cfg.ServerURL == "" || cfg.Token == "") {
			fmt.Println("ZenoGuard Agent is not configured.")
			fmt.Println("Please configure using:")
			fmt.Println("  zenoguard-agent -config -server <URL> -token <TOKEN>")
			fmt.Println("")
			fmt.Println("Example:")
			fmt.Println("  zenoguard-agent -config -server https://monitor.example.com -token abc123...")
			fmt.Println("")
			fmt.Println("Or set environment variables:")
			fmt.Println("  ZENOGUARD_SERVER_URL=<URL>")
			fmt.Println("  ZENOGUARD_TOKEN=<TOKEN>")
			fmt.Println("  ZENOGUARD_HOSTNAME=<hostname>")
			fmt.Println("  ZENOGUARD_REPORT_INTERVAL=<seconds>")
			os.Exit(1)
		}

		// Save configuration from command line (if provided)
		if *serverURL != "" {
			cfg.ServerURL = *serverURL
		}
		if *token != "" {
			cfg.Token = *token
		}

		// Save config if we have settings now
		if cfg.ServerURL != "" && cfg.Token != "" {
			if err := config.SaveConfig(cfg); err != nil {
				logger.Fatal("Failed to save configuration: " + err.Error())
			}
			logger.Info("Initial configuration saved")
		}
	}

	// Override config with command-line arguments if provided
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
	}
	if *token != "" {
		cfg.Token = *token
	}

	// Validate configuration
	if cfg.ServerURL == "" || cfg.Token == "" {
		logger.Fatal("Invalid configuration: server URL and token are required")
	}

	// Daemonize if requested
	if *daemonFlag {
		if err := daemon.Daemonize(); err != nil {
			logger.Fatal("Failed to daemonize: " + err.Error())
		}
		// If we reach here, daemonization failed
		os.Exit(1)
	}

	// Check if already running (skip if we're the daemon child process)
	// When running as daemon, main.go is executed twice: once by parent, once by child
	// The parent writes the PID file and exits, the child continues
	// So if the PID file contains our own PID, we're the child and should continue
	if running, pid, _ := daemon.Status(); running && pid != os.Getpid() {
		logger.Info(fmt.Sprintf("Agent is already running (PID: %d)", pid))
		os.Exit(0)
	}

	// Create reporter
	reporterCfg := &reporter.Config{
		ServerURL:      cfg.ServerURL,
		Token:          cfg.Token,
		ReportInterval: cfg.ReportInterval,
	}
	rep := reporter.NewReporter(reporterCfg)

	// Set up signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		logger.Info("Received signal: " + sig.String())
		rep.Stop()
		daemon.RemovePIDFile()
		logger.Close()
		os.Exit(0)
	}()

	// Note: PID file is already written by the parent during daemonization
	// No need to write it again here

	logger.Info("Starting ZenoGuard Agent v" + version)
	logger.Info("Server: " + cfg.ServerURL)

	// Start reporting
	if err := rep.Start(); err != nil {
		logger.Error("Reporter error: " + err.Error())
		daemon.RemovePIDFile()
		os.Exit(1)
	}
}

// configureAgent runs the interactive configuration
func configureAgent(serverURL, token string) error {
	if serverURL == "" || token == "" {
		return fmt.Errorf("server URL and token are required")
	}

	// Validate URL format
	if len(serverURL) < 10 || ! (serverURL[:8] == "https://" || serverURL[:7] == "http://") {
		return fmt.Errorf("invalid server URL format (should start with http:// or https://)")
	}

	// Validate token
	if len(token) < 10 {
		return fmt.Errorf("invalid token (too short)")
	}

	cfg := &config.Config{
		ServerURL:      serverURL,
		Token:          token,
		ReportInterval: 60,
	}

	return config.SaveConfig(cfg)
}

// parseLogLevel parses log level string
func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "debug":
		return logger.DEBUG
	case "info":
		return logger.INFO
	case "warn":
		return logger.WARN
	case "error":
		return logger.ERROR
	default:
		return logger.INFO
	}
}
