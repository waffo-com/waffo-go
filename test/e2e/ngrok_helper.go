//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"
)

const (
	cloudflaredBin    = "/opt/homebrew/bin/cloudflared"
	tunnelLogFile     = "/tmp/cloudflared-go.log"
	tunnelMaxWait     = 30 * time.Second
	tunnelPollInterval = 1 * time.Second
)

var tunnelCmd *exec.Cmd

var tunnelURLPattern = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// StartNgrok starts a cloudflared tunnel for the given port.
// No authentication required.
// Returns the public HTTPS URL.
func StartNgrok(port int) (string, error) {
	// Kill any existing cloudflared tunnel processes
	exec.Command("pkill", "-f", "cloudflared tunnel").Run()
	time.Sleep(1 * time.Second)

	// Remove old log file
	os.Remove(tunnelLogFile)

	// Open log file for stderr output
	logFile, err := os.Create(tunnelLogFile)
	if err != nil {
		return "", fmt.Errorf("failed to create log file: %w", err)
	}

	// Start cloudflared tunnel
	tunnelCmd = exec.Command(cloudflaredBin, "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
	tunnelCmd.Stderr = logFile
	tunnelCmd.Stdout = nil

	if err := tunnelCmd.Start(); err != nil {
		logFile.Close()
		return "", fmt.Errorf("failed to start cloudflared: %w", err)
	}
	logFile.Close()

	// Poll log file for tunnel URL
	deadline := time.Now().Add(tunnelMaxWait)
	for time.Now().Before(deadline) {
		time.Sleep(tunnelPollInterval)

		content, err := os.ReadFile(tunnelLogFile)
		if err != nil {
			continue
		}

		match := tunnelURLPattern.Find(content)
		if match != nil {
			url := string(match)
			fmt.Printf("[tunnel] Ready: %s -> localhost:%d\n", url, port)
			return url, nil
		}
	}

	StopNgrok()
	return "", fmt.Errorf("cloudflared tunnel failed to start within %s", tunnelMaxWait)
}

// StopNgrok stops the cloudflared tunnel.
func StopNgrok() {
	if tunnelCmd != nil && tunnelCmd.Process != nil {
		tunnelCmd.Process.Kill()
		tunnelCmd = nil
	}

	// Clean up any lingering processes
	exec.Command("pkill", "-f", "cloudflared tunnel").Run()
}
