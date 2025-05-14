package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
)

var (
	// Default values
	defaultCheckInterval = 2 * time.Second
	defaultTestURL       = "https://www.google.com"
	defaultTimeout       = 5 * time.Second
)

func main() {
	// Define command line flags
	checkIntervalFlag := flag.Duration("interval", defaultCheckInterval, "Interval between connection checks (e.g. 2s, 1m)")
	testURLFlag := flag.String("url", defaultTestURL, "URL to test connection against")
	timeoutFlag := flag.Duration("timeout", defaultTimeout, "HTTP request timeout")
	flag.Parse()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: *timeoutFlag,
	}

	// Setup signal catching for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Clear screen and hide cursor
	fmt.Print("\033[H\033[2J\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor when done

	fmt.Println("Internet Connection Monitor")
	fmt.Printf("Testing connection to: %s\n", *testURLFlag)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("----------------------------")

	// Create ticker for periodic checks
	ticker := time.NewTicker(*checkIntervalFlag)
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
	
	// Latency statistics
	var minLatency time.Duration = -1
	var maxLatency time.Duration
	var totalLatency time.Duration
	var latencyCount int

	// Initial status check
	var latency time.Duration
	lastStatus, latency = checkConnection(client, *testURLFlag)
	statusChangeTime = time.Now()
	
	// Update latency stats if connected
	if lastStatus && latency > 0 {
		minLatency = latency
		maxLatency = latency
		totalLatency = latency
		latencyCount = 1
	}
	
	displayStatus(lastStatus, success, failure, info, 0, latency)

	// Main loop
	for {
		select {
		case <-ticker.C:
			currentStatus, latency := checkConnection(client, *testURLFlag)
			now := time.Now()
			duration := now.Sub(statusChangeTime)

			// Update uptime/downtime tracking - simplified logic
			if currentStatus {
				uptime += duration
				
				// Update latency statistics
				if latency > 0 {
					if minLatency < 0 || latency < minLatency {
						minLatency = latency
					}
					if latency > maxLatency {
						maxLatency = latency
					}
					totalLatency += latency
					latencyCount++
				}
			} else {
				downtime += duration
			}
			
			// Update tracking variables
			statusChangeTime = now
			if currentStatus != lastStatus {
				lastStatus = currentStatus
			}

			displayStatus(currentStatus, success, failure, info, duration, latency)

		case <-sigChan:
			// Clean up and exit
			fmt.Println("\n\nExiting Connection Monitor")
			fmt.Printf("Total uptime: %s\n", formatDuration(uptime))
			fmt.Printf("Total downtime: %s\n", formatDuration(downtime))
			if latencyCount > 0 {
				fmt.Printf("Min latency: %s\n", minLatency)
				fmt.Printf("Max latency: %s\n", maxLatency)
				fmt.Printf("Avg latency: %s\n", totalLatency/time.Duration(latencyCount))
			}
			return
		}
	}
}

// checkConnection tests the internet connection and returns connection status and latency
func checkConnection(client *http.Client, url string) (bool, time.Duration) {
	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()
	latency := time.Since(start)
	return resp.StatusCode >= 200 && resp.StatusCode < 300, latency
}

// displayStatus prints the current connection status, duration, and network latency if connected.
func displayStatus(connected bool, success, failure, info *color.Color, duration time.Duration, latency time.Duration) {
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

		// Print measured latency
		fmt.Printf("%s", latency.Round(time.Millisecond))
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
