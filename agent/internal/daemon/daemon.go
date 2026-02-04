package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"zenoguard-agent/internal/logger"
)

const (
	pidFilePath = "/var/run/zenoguard.pid"
)

// Daemonize converts the current process into a daemon
func Daemonize() error {
	// Check if already running
	if isRunning() {
		return fmt.Errorf("zenoguard agent is already running (PID file exists)")
	}

	logger.Info("Daemonizing process...")

	// Build args for child process, removing -daemon flag
	args := buildChildArgs()

	// Fork the process
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Start the daemon
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Write PID file
	if err := writePIDFile(cmd.Process.Pid); err != nil {
		// Kill the daemon if we can't write PID file
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	logger.Info(fmt.Sprintf("Daemon started with PID: %d", cmd.Process.Pid))
	logger.Info("Parent process exiting...")

	// Exit parent process
	os.Exit(0)
	return nil
}

// buildChildArgs builds command line arguments for the child process
// It removes the -daemon flag to prevent infinite daemonization
func buildChildArgs() []string {
	args := os.Args[1:] // Skip program name
	filtered := make([]string, 0, len(args))

	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		// Skip -daemon and its value if it has one
		if arg == "-daemon" || arg == "--daemon" {
			continue
		}

		// Check if next arg should be skipped (e.g., -daemon value)
		if (arg == "-d" || arg == "--daemon-flag") && i+1 < len(args) {
			skipNext = true
			continue
		}

		filtered = append(filtered, arg)
	}

	// Prepend program name
	return append([]string{os.Args[0]}, filtered...)
}

// isRunning checks if the daemon is already running
func isRunning() bool {
	// Read PID file
	pidData, err := os.ReadFile(pidFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}

	// Parse PID
	pidStr := string(pidData)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false
	}

	// Check if it's our own PID (we're the daemon child process)
	if pid == os.Getpid() {
		return false
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process not running, remove stale PID file
		os.Remove(pidFilePath)
		return false
	}

	return true
}

// writePIDFile writes the PID file
func writePIDFile(pid int) error {
	return os.WriteFile(pidFilePath, []byte(strconv.Itoa(pid)), 0644)
}

// RemovePIDFile removes the PID file
func RemovePIDFile() error {
	if _, err := os.Stat(pidFilePath); err == nil {
		return os.Remove(pidFilePath)
	}
	return nil
}

// GetPID returns the PID from the PID file
func GetPID() (int, error) {
	pidData, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(pidData))
}

// SetupSignalHandler sets up signal handlers for graceful shutdown
func SetupSignalHandler(callback func()) chan struct{} {
	stopChan := make(chan struct{})

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal: " + sig.String())

		// Execute callback
		if callback != nil {
			callback()
		}

		// Clean up
		RemovePIDFile()
		logger.Close()

		close(stopChan)
	}()

	return stopChan
}

// Stop stops the daemon
func Stop() error {
	pid, err := GetPID()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	logger.Info("Sent SIGTERM to daemon (PID: " + fmt.Sprint(pid) + ")")
	return nil
}

// Status returns the status of the daemon
func Status() (bool, int, error) {
	pidData, err := os.ReadFile(pidFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return false, 0, err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0, err
	}

	// Check if process is running
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Stale PID file
		os.Remove(pidFilePath)
		return false, 0, nil
	}

	return true, pid, nil
}
