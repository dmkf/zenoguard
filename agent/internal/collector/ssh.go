package collector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"zenoguard-agent/internal/logger"
)

// SSHLogin represents an SSH login entry
type SSHLogin struct {
	User            string `json:"user"`
	IP              string `json:"ip"`
	Time            string `json:"time"`
	Method          string `json:"method"`    // password, publickey, etc.
	Success         bool   `json:"success"`   // true for accepted, false for failed
	Port            int    `json:"port"`      // SSH port
	Protocol        string `json:"protocol"`  // usually ssh2
	SessionDuration int64  `json:"session_duration"` // seconds
	IsActive        bool   `json:"is_active"`         // currently logged in
}

// SSHCollector collects SSH login information
type SSHCollector struct {
	BaseCollector
	logPaths []string
}

// NewSSHCollector creates a new SSH collector
func NewSSHCollector() *SSHCollector {
	paths := []string{
		"/var/log/auth.log",      // Debian/Ubuntu
		"/var/log/secure",        // CentOS/RHEL/Amazon Linux
		"/var/log/messages",      // Some systems
	}

	// Add macOS-specific paths
	if runtime.GOOS == "darwin" {
		paths = append([]string{
			"/var/log/system.log",  // macOS system log
			"/var/log/ipfw.log",    // macOS firewall log (may contain SSH info)
		}, paths...)
	}

	return &SSHCollector{
		BaseCollector: BaseCollector{name: "ssh"},
		logPaths:      paths,
	}
}

// Collect collects SSH login information
func (c *SSHCollector) Collect() (interface{}, error) {
	logger.Info("Collecting SSH login information")

	logins := make([]SSHLogin, 0)

	// Try each log path
	for _, logPath := range c.logPaths {
		if _, err := os.Stat(logPath); err == nil {
			fileLogins, err := c.parseLogFile(logPath)
			if err != nil {
				logger.Warn("Failed to parse " + logPath + ": " + err.Error())
				continue
			}
			logins = append(logins, fileLogins...)
			logger.Info("Found " + fmt.Sprint(len(fileLogins)) + " SSH log entries in " + logPath)
		}
	}

	// Collect active sessions and calculate durations
	logins = c.enrichWithActiveSessions(logins)

	// If no log entries found but we have active sessions, return those
	// This handles systems like macOS where SSH logs are in binary format
	if len(logins) == 0 {
		logger.Info("No SSH log entries found, checking for active sessions only")
		activeSessions := c.collectActiveSessions()
		if len(activeSessions) > 0 {
			logger.Info("Returning " + fmt.Sprint(len(activeSessions)) + " active sessions")
			return activeSessions, nil
		}
	}

	logger.Info("Total SSH log entries collected: " + fmt.Sprint(len(logins)))
	return logins, nil
}

// parseLogFile parses an SSH log file
func (c *SSHCollector) parseLogFile(logPath string) ([]SSHLogin, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	logins := make([]SSHLogin, 0)
	scanner := bufio.NewScanner(file)

	// Regular expressions for different log formats
	// Ubuntu/Debian format: Jan 30 10:00:00 hostname sshd[1234]: Accepted password for root from 1.2.3.4 port 22 ssh2
	acceptedPattern := regexp.MustCompile(
		`(\w+\s+\d+\s+\d+:\d+:\d+).*sshd\[\d+\]:\s+(Accepted|Failed)\s+(\w+)\s+for\s+(\w+)\s+from\s+([\d.]+)\s+port\s+(\d+)`,
	)

	// CentOS/RHEL format: Jan 30 10:00:00 hostname sshd[1234]: Failed password for root from 1.2.3.4 port 22 ssh2
	// Also handle publickey auth
	publickeyPattern := regexp.MustCompile(
		`(\w+\s+\d+\s+\d+:\d+:\d+).*sshd\[\d+\]:\s+(Accepted|Failed)\s+publickey\s+for\s+(\w+)\s+from\s+([\d.]+)\s+port\s+(\d+)`,
	)

	// Invalid user pattern
	invalidUserPattern := regexp.MustCompile(
		`(\w+\s+\d+\s+\d+:\d+:\d+).*sshd\[\d+\]:\s+(Invalid user)\s+(\w+)\s+from\s+([\d.]+)\s+port\s+(\d+)`,
	)

	// Get current year for adding to log times
	currentYear := time.Now().Year()

	// Parse line by line (only last 1000 lines for performance)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 1000 {
			lines = lines[1:]
		}
	}

	// Process from oldest to newest
	for _, line := range lines {
		// Try accepted/failed pattern
		matches := acceptedPattern.FindStringSubmatch(line)
		if len(matches) >= 7 {
			login := c.parseLoginFromMatches(matches, true)
			// Add year to time
			login.Time = fmt.Sprintf("%d %s", currentYear, login.Time)
			logins = append(logins, login)
			continue
		}

		// Try publickey pattern
		matches = publickeyPattern.FindStringSubmatch(line)
		if len(matches) >= 6 {
			login := SSHLogin{
				Time:     matches[1],
				User:     matches[4],
				IP:       matches[5],
				Method:   "publickey",
				Success:  strings.Contains(matches[2], "Accepted"),
				Port:     c.parseInt(matches[6]),
				Protocol: "ssh2",
			}
			// Add year to time
			login.Time = fmt.Sprintf("%d %s", currentYear, login.Time)
			logins = append(logins, login)
			continue
		}

		// Try invalid user pattern
		matches = invalidUserPattern.FindStringSubmatch(line)
		if len(matches) >= 6 {
			login := SSHLogin{
				Time:     matches[1],
				User:     matches[3], // Invalid username
				IP:       matches[4],
				Method:   "password",
				Success:  false,
				Port:     c.parseInt(matches[5]),
				Protocol: "ssh2",
			}
			// Add year to time
			login.Time = fmt.Sprintf("%d %s", currentYear, login.Time)
			logins = append(logins, login)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Filter logins from last 15 minutes (to avoid sending too much data)
	filteredLogins := c.filterRecentLogins(logins, 15*time.Minute)

	return filteredLogins, nil
}

// parseLoginFromMatches parses login from regex matches
func (c *SSHCollector) parseLoginFromMatches(matches []string, hasMethod bool) SSHLogin {
	login := SSHLogin{
		Time:     matches[1],
		User:     matches[4],
		IP:       matches[5],
		Method:   "password",
		Success:  strings.Contains(matches[2], "Accepted"),
		Port:     c.parseInt(matches[6]),
		Protocol: "ssh2",
	}

	if hasMethod && len(matches) > 3 {
		login.Method = matches[3]
	}

	return login
}

// filterRecentLogins filters logins to only include recent ones
func (c *SSHCollector) filterRecentLogins(logins []SSHLogin, duration time.Duration) []SSHLogin {
	recentLogins := make([]SSHLogin, 0)
	cutoffTime := time.Now().Add(-duration)

	for _, login := range logins {
		// Parse time from log format (e.g., "Jan 30 10:00:00")
		// Add current year since log doesn't include it
		t, err := time.Parse("2006 Jan 2 15:04:05",
			fmt.Sprintf("%d %s", time.Now().Year(), login.Time))

		if err == nil && t.After(cutoffTime) {
			recentLogins = append(recentLogins, login)
		}
	}

	return recentLogins
}

// parseInt safely parses an integer
func (c *SSHCollector) parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// getLogPath returns the first available log path
func (c *SSHCollector) getLogPath() string {
	for _, path := range c.logPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// GetLogPath returns the log path being used
func (c *SSHCollector) GetLogPath() string {
	return c.getLogPath()
}

// GetLogSize returns the size of the log file in bytes
func (c *SSHCollector) GetLogSize() int64 {
	logPath := c.getLogPath()
	if logPath == "" {
		return 0
	}

	info, err := os.Stat(logPath)
	if err != nil {
		return 0
	}

	return info.Size()
}

// enrichWithActiveSessions enriches logins with active session info and durations
func (c *SSHCollector) enrichWithActiveSessions(logins []SSHLogin) []SSHLogin {
	// Collect active sessions using `who` command
	activeSessions := c.collectActiveSessions()

	// Create a map for quick lookup
	activeSessionMap := make(map[string]bool)
	for _, session := range activeSessions {
		key := session.User + "@" + session.IP
		activeSessionMap[key] = true
	}

	// Enrich logins with active status and duration
	for i := range logins {
		if !logins[i].Success {
			continue
		}

		// Check if this login is currently active
		key := logins[i].User + "@" + logins[i].IP
		if activeSessionMap[key] {
			logins[i].IsActive = true
			// Calculate duration from login time to now
			logins[i].SessionDuration = c.calculateDurationFromLogin(logins[i].Time)
		} else {
			logins[i].IsActive = false
			// For completed sessions, we'd need to parse logout events
			// This is more complex and would require additional log parsing
			logins[i].SessionDuration = 0
		}
	}

	return logins
}

// collectActiveSessions collects currently active SSH sessions using `who` command
func (c *SSHCollector) collectActiveSessions() []SSHLogin {
	sessions := make([]SSHLogin, 0)

	// Try `who -u` command first
	output, err := c.execCommand("who", "-u")
	if err != nil {
		logger.Warn("Failed to run 'who -u': " + err.Error())
		// Try `w` command as fallback
		output, err = c.execCommand("w")
		if err != nil {
			logger.Warn("Failed to run 'w': " + err.Error())
			return sessions
		}
	}

	// Parse output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "USER") {
			// Skip empty lines and header
			continue
		}

		session := c.parseWhoLine(line)
		if session.User != "" {
			logger.Debug("Parsed SSH session: user=%s ip=%s time=%s duration=%d active=%v",
				session.User, session.IP, session.Time, session.SessionDuration, session.IsActive)
			sessions = append(sessions, session)
		}
	}

	logger.Info("Found " + fmt.Sprint(len(sessions)) + " active sessions")
	return sessions
}

// parseWhoLine parses a line from `who` or `w` command output
func (c *SSHCollector) parseWhoLine(line string) SSHLogin {
	fields := strings.Fields(line)
	logger.Debug("Parsing who line: %s", line)

	if len(fields) < 2 {
		return SSHLogin{}
	}

	session := SSHLogin{
		User:     fields[0],
		Success:  true,
		IsActive: true,
		Protocol: "ssh2",
	}

	var ipFound bool

	// Extract IP from the last field
	if len(fields) > 0 {
		lastField := fields[len(fields)-1]
		logger.Debug("Last field: %s", lastField)
		// Check if it's an IP in parentheses
		if strings.HasPrefix(lastField, "(") && strings.HasSuffix(lastField, ")") {
			ip := strings.Trim(lastField, "()")
			// Only report if it looks like an IP address (contains dots and numbers)
			if strings.Contains(ip, ".") || strings.Contains(ip, ":") {
				session.IP = ip
				ipFound = true
				logger.Debug("Found IP: %s", ip)
			}
		}
	}

	// Only process if this is a remote SSH session (has IP)
	if !ipFound {
		// Return empty login for local sessions
		logger.Debug("No IP found, skipping local session")
		return SSHLogin{}
	}

	// Parse who -u output format (Linux):
	// username pts/0 2026-01-31 10:00 (1.2.3.4) 00:05 1234
	// OR
	// username pts/0 2026-01-31 10:00 00:05 1234 (1.2.3.4)

	// Extract login time (fields[2] and possibly fields[3])
	// Format: fields[2] = "2026-01-31", fields[3] = "12:37" (time) or "00:07" (idle)
	if len(fields) >= 4 {
		// Try to parse login time
		loginTime := fields[2]
		logger.Debug("Field[2] (date): %s", loginTime)
		// fields[3] is usually time (HH:MM), append it
		if len(fields) > 3 && strings.Contains(fields[3], ":") {
			loginTime = loginTime + " " + fields[3]
			logger.Debug("Field[3] (time): %s, combined: %s", fields[3], loginTime)
		}

		loginTimeObj := c.formatLoginTime(loginTime)
		logger.Debug("Parsed loginTimeObj: %v (zero=%v)", loginTimeObj, loginTimeObj.IsZero())
		if !loginTimeObj.IsZero() {
			session.Time = loginTimeObj.Format("2006-01-02 15:04:05")
			session.SessionDuration = c.calculateDurationFromTime(loginTimeObj)
			logger.Debug("Session time: %s, duration: %d", session.Time, session.SessionDuration)
		}
	}

	return session
}

// isTimeString checks if a string looks like an idle time (e.g., "00:05", ".")
func (c *SSHCollector) isTimeString(s string) bool {
	// Check for idle time patterns: "00:05", "01:30", "."
	return strings.HasPrefix(s, ".") || (len(s) == 5 && strings.Contains(s, ":"))
}

// calculateDurationFromLogin calculates duration in seconds from login time to now
func (c *SSHCollector) calculateDurationFromLogin(loginTime string) int64 {
	// Parse time from log format (e.g., "Jan 30 10:00:00")
	t, err := time.Parse("2006 Jan 2 15:04:05",
		fmt.Sprintf("%d %s", time.Now().Year(), loginTime))
	if err != nil {
		return 0
	}

	duration := time.Since(t)
	seconds := int64(duration.Seconds())

	// Handle year boundary (if login was from last year)
	if seconds < 0 {
		// Try with previous year
		t, err = time.Parse("2006 Jan 2 15:04:05",
			fmt.Sprintf("%d %s", time.Now().Year()-1, loginTime))
		if err == nil {
			duration = time.Since(t)
			seconds = int64(duration.Seconds())
		}
	}

	return seconds
}

// execCommand executes a command and returns its output
func (c *SSHCollector) execCommand(name string, args ...string) (string, error) {
	cmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	output, err := c.execShellCommand(cmd)
	if err != nil {
		return "", err
	}
	return output, nil
}

// execShellCommand executes a shell command
func (c *SSHCollector) execShellCommand(cmd string) (string, error) {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// formatLoginTime converts who command time format to Go time.Time
// Input: "2026-01-31 10:00" or "Jan 30 10:00:00" format
// Output: time.Time object (in local timezone)
func (c *SSHCollector) formatLoginTime(loginTime string) time.Time {
	// Try ISO format first: "2026-01-31 10:00"
	// Parse in local timezone since who output is in local time
	isoFormat := "2006-01-02 15:04"
	t, err := time.ParseInLocation(isoFormat, loginTime, time.Local)
	if err == nil {
		return t
	}

	// Try with seconds: "2026-01-31 10:00:00"
	isoFormatWithSec := "2006-01-02 15:04:05"
	t, err = time.ParseInLocation(isoFormatWithSec, loginTime, time.Local)
	if err == nil {
		return t
	}

	// Try log format with current year: "Jan 31 10:00:00"
	logFormat := "2006 Jan 2 15:04:05"
	t, err = time.ParseInLocation(logFormat,
		fmt.Sprintf("%d %s", time.Now().Year(), loginTime), time.Local)
	if err == nil {
		return t
	}

	// Return zero time if all parsing fails
	return time.Time{}
}

// calculateDurationFromLoginTime calculates duration in seconds from login time string
func (c *SSHCollector) calculateDurationFromLoginTime(loginTime string) int64 {
	if loginTime == "" {
		return 0
	}

	// Try format: "Jan 2 15:04:05"
	t, err := time.Parse("Jan 2 15:04:05", loginTime)
	if err != nil {
		// Try format with year: "2006 Jan 2 15:04:05"
		t, err = time.Parse("2006 Jan 2 15:04:05",
			fmt.Sprintf("%d %s", time.Now().Year(), loginTime))
		if err != nil {
			return 0
		}
	}

	duration := time.Since(t)
	seconds := int64(duration.Seconds())

	// Handle year boundary
	if seconds < 0 {
		t, err = time.Parse("2006 Jan 2 15:04:05",
			fmt.Sprintf("%d %s", time.Now().Year()-1, loginTime))
		if err == nil {
			duration = time.Since(t)
			seconds = int64(duration.Seconds())
		}
	}

	return seconds
}

// calculateDurationFromTime calculates duration in seconds from a time.Time object
func (c *SSHCollector) calculateDurationFromTime(t time.Time) int64 {
	if t.IsZero() {
		logger.Debug("calculateDurationFromTime: zero time")
		return 0
	}

	now := time.Now()
	logger.Debug("calculateDurationFromTime: now=%v, login=%v", now, t)
	duration := now.Sub(t)
	seconds := int64(duration.Seconds())
	logger.Debug("calculateDurationFromTime: duration=%v, seconds=%d", duration, seconds)

	// Handle future times (shouldn't happen, but just in case)
	if seconds < 0 {
		logger.Debug("calculateDurationFromTime: negative duration, returning 0")
		return 0
	}

	return seconds
}
