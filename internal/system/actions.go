package system

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// RestartService runs systemctl restart <service> safely
func RestartService(serviceName string) error {
	// 1. Security: Strict Whitelist
	// We do NOT let the user pass arbitrary strings to exec.Command
	// Map "friendly names" to actual service names
	allowed := map[string]string{
		"nginx": "nginx",
		"php":   "php8.3-fpm", // Adjust based on your PHP version (e.g., php8.1-fpm)
		"mysql": "mysql",
	}

	realName, ok := allowed[serviceName]
	if !ok {
		return fmt.Errorf("service '%s' is not allowed", serviceName)
	}

	// 2. Check OS (Don't run systemctl on Mac)
	if runtime.GOOS == "darwin" {
		fmt.Printf("[MOCK] Executing: systemctl restart %s\n", realName)
		time.Sleep(1 * time.Second) // Fake delay to feel real
		return nil
	}

	// 3. Execute on Linux
	cmd := exec.Command("systemctl", "restart", realName)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
