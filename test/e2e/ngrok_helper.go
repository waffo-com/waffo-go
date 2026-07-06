//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	cloudflaredBin     = "/opt/homebrew/bin/cloudflared"
	tunnelLogFile      = "/tmp/cloudflared-go.log"
	tunnelMaxWait      = 30 * time.Second
	tunnelPollInterval = 1 * time.Second
	tunnelStartRetries = 3
)

var tunnelCmd *exec.Cmd

var tunnelURLPattern = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// StartNgrok starts a cloudflared tunnel for the given port.
// No authentication required.
// Returns the public HTTPS URL.
func StartNgrok(port int) (string, error) {
	killExistingTunnelsIfRequested()
	time.Sleep(1 * time.Second)

	var lastErr error
	for attempt := 1; attempt <= tunnelStartRetries; attempt++ {
		stopTunnelProcess()
		os.Remove(tunnelLogFile)

		logFile, err := os.Create(tunnelLogFile)
		if err != nil {
			return "", fmt.Errorf("failed to create log file: %w", err)
		}

		tunnelCmd = exec.Command(cloudflaredBin, "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
		tunnelCmd.Stderr = logFile
		tunnelCmd.Stdout = nil

		if err := tunnelCmd.Start(); err != nil {
			logFile.Close()
			return "", fmt.Errorf("failed to start cloudflared: %w", err)
		}
		logFile.Close()

		url, err := waitForTunnelURL(port)
		if err == nil {
			if healthErr := probePublicHealth(url); healthErr != nil {
				message := fmt.Sprintf("cloudflared tunnel URL was published but /health was not reachable from this machine: %v", healthErr)
				if os.Getenv("WAFFO_E2E_REQUIRE_TUNNEL_HEALTH") == "true" {
					err = fmt.Errorf("%s", message)
				} else {
					fmt.Printf("[tunnel] %s\n", message)
				}
			}
		}
		if err == nil {
			fmt.Printf("[tunnel] Ready on attempt %d/%d: %s -> localhost:%d\n", attempt, tunnelStartRetries, url, port)
			return url, nil
		}

		lastErr = err
		fmt.Printf("[tunnel] Attempt %d/%d failed: %v\n", attempt, tunnelStartRetries, err)
		stopTunnelProcess()
		if attempt < tunnelStartRetries {
			time.Sleep(time.Duration(attempt) * tunnelPollInterval)
		}
	}

	killExistingTunnelsIfRequested()
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("cloudflared tunnel failed to start")
}

// StopNgrok stops the cloudflared tunnel.
func StopNgrok() {
	stopTunnelProcess()
	killExistingTunnelsIfRequested()
}

func waitForTunnelURL(port int) (string, error) {
	deadline := time.Now().Add(tunnelMaxWait)
	for time.Now().Before(deadline) {
		time.Sleep(tunnelPollInterval)

		if tunnelCmd != nil && tunnelCmd.ProcessState != nil && tunnelCmd.ProcessState.Exited() {
			return "", fmt.Errorf("cloudflared exited early")
		}

		content, err := os.ReadFile(tunnelLogFile)
		if err != nil {
			continue
		}

		match := tunnelURLPattern.Find(content)
		if match != nil {
			return string(match), nil
		}
	}

	return "", fmt.Errorf("cloudflared tunnel failed to publish a URL within %s for localhost:%d", tunnelMaxWait, port)
}

func probePublicHealth(baseURL string) error {
	timeout := 5 * time.Second
	if os.Getenv("WAFFO_E2E_REQUIRE_TUNNEL_HEALTH") == "true" {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(timeout)
	healthURL := baseURL + "/health"
	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "ok" {
				return nil
			} else {
				lastErr = fmt.Errorf("health returned %d: %.120s", resp.StatusCode, string(body))
			}
		}
		time.Sleep(tunnelPollInterval)
	}

	return fmt.Errorf("cloudflared tunnel URL was published but /health was not reachable within %s: %v", timeout, lastErr)
}

func stopTunnelProcess() {
	if tunnelCmd != nil && tunnelCmd.Process != nil {
		tunnelCmd.Process.Kill()
		tunnelCmd = nil
	}
}

func killExistingTunnelsIfRequested() {
	if os.Getenv("WAFFO_E2E_KILL_EXISTING_TUNNELS") != "true" {
		return
	}
	exec.Command("pkill", "-f", "cloudflared tunnel").Run()
}
