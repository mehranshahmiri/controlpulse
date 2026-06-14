package logs

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// ReadLog returns the last N lines of a specific log file
func ReadLog(logType string) (string, error) {
	// 1. Map Log Names to Real Paths
	// On Linux, these are standard. On Mac, we mock them.
	var path string
	
	if runtime.GOOS == "darwin" {
		// Mock Data for Dev Mode
		return MockLogData(logType), nil
	} else {
		// Real Linux Paths
		switch logType {
		case "nginx_access":
			path = "/var/log/nginx/access.log"
		case "nginx_error":
			path = "/var/log/nginx/error.log"
		case "php_error":
			path = "/var/log/php8.3-fpm.log" // Adjust version as needed
		case "syslog":
			path = "/var/log/syslog"
		default:
			return "", fmt.Errorf("unknown log type")
		}
	}

	// 2. Open File
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open log file: %v", err)
	}
	defer file.Close()

	// 3. Read Last 50 Lines (Simple implementation)
	// In a massive production app, you'd use 'seek', but for text logs this is fine.
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		// Keep memory low, just keep last 50
		if len(lines) > 50 {
			lines = lines[1:]
		}
	}

	return strings.Join(lines, "\n"), nil
}

// MockLogData generates fake logs for testing on Mac
func MockLogData(logType string) string {
	timestamp := "2024-12-28 10:00:00"
	switch logType {
	case "nginx_access":
		return fmt.Sprintf("%s [INFO] 192.168.1.1 GET /index.php 200 OK\n%s [INFO] 192.168.1.5 GET /style.css 304 Not Modified", timestamp, timestamp)
	case "nginx_error":
		return fmt.Sprintf("%s [ERR] Directory index of '/var/www/html/' is forbidden", timestamp)
	case "php_error":
		return fmt.Sprintf("[%s] PHP Fatal error: Uncaught TypeError: Argument 1 passed to main() must be int", timestamp)
	default:
		return "No data."
	}
}
