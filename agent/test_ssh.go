package main

import (
	"fmt"
	"zenoguard-agent/internal/collector"
)

func main() {
	sshCollector := collector.NewSSHCollector()
	fmt.Printf("Log paths: %v\n", sshCollector.GetLogPath())
	fmt.Printf("Log size: %d bytes\n", sshCollector.GetLogSize())

	data, err := sshCollector.Collect()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nSSH Data:\n")
	for _, login := range data.([]collector.SSHLogin) {
		fmt.Printf("User: %s, IP: %s, Time: %s, Active: %v, Duration: %d seconds\n",
			login.User, login.IP, login.Time, login.IsActive, login.SessionDuration)
	}
}
