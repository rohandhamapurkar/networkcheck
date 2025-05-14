package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
)

const (
	checkInterval = 2 * time.Second
	testURL       = "https://www.google.com"
	timeout       = 5 * time.Second
)

func main() {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Setup signal catching for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Clear screen and hide cursor
	fmt.Print("\033[H\033[2J\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor when done

	fmt.Println("Internet Connection Monitor")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("----------------------------")

	// Create ticker for periodic checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Success and failure formatters
	success := color.New(color.FgGreen, color.Bold)
	failure := color.New(color.FgRed, color.Bold)
	info := color.New(color.FgCyan)

	// Status tracking
	var lastStatus bool
	var statusChangeTime time.Time
	var downtime time.Duration
	var uptime time.Duration

	// Initial status check
	lastStatus = checkConnection(client)
	statusChangeTime = time.Now()
	displayStatus(lastStatus, success, failure, info, 0, client)

	// Main loop
	for {
		select {
		case <-ticker.C:
			currentStatus := checkConnection(client)
			now := time.Now()
			duration := now.Sub(statusChangeTime)

			// Always update using actual elapsed time
			if currentStatus != lastStatus {
				if currentStatus {
					downtime += duration
				} else {
					uptime += duration
				}
				statusChangeTime = now
				lastStatus = currentStatus
			} else {
				if currentStatus {
					uptime += duration
				} else {
					downtime += duration
				}
				statusChangeTime = now
			}

			displayStatus(currentStatus, success, failure, info, duration, client)

		case <-sigChan:
			// Clean up and exit
			fmt.Println("\n\nExiting Connection Monitor")
			fmt.Printf("Total uptime: %s\n", formatDuration(uptime))
			fmt.Printf("Total downtime: %s\n", formatDuration(downtime))
			return
		}
	}
}

func checkConnection(client *http.Client) bool {
	resp, err := client.Get(testURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// displayStatus prints the current connection status, duration, and network latency if connected.
func displayStatus(connected bool, success, failure, info *color.Color, duration time.Duration, client *http.Client) {
	// Move cursor to status line (row 4, clear line)
	fmt.Print("\033[4;0H\033[K")

	// Get current time for status display
	timeNow := time.Now().Format("15:04:05")

	// Print connection status with color
	if connected {
		success.Printf("[%s] ✓ CONNECTED    ", timeNow)
	} else {
		failure.Printf("[%s] ✗ DISCONNECTED ", timeNow)
	}

	// Print duration of current state if available
	if duration > 0 {
		info.Printf("Duration: %s", formatDuration(duration))
	}

	// If connected, print network latency
	if connected {
		// Move cursor to row 6, clear line
		fmt.Print("\033[6;0H\033[K")
		fmt.Print("Network Latency: ")

		// Measure latency by timing an HTTP GET request
		start := time.Now()
		resp, err := client.Get(testURL)
		if err == nil {
			resp.Body.Close()
			latency := time.Since(start)
			fmt.Printf("%s", latency.Round(time.Millisecond))
		} else {
			fmt.Print("Unknown")
		}
	}
}

// formatDuration returns a human-readable string for a time.Duration (e.g., 1h 2m 3s)
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
