package system

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type UpdatePackage struct {
	Name    string
	Version string
	Type    string // Security, Bugfix, etc. (Mocked for now)
}

// CheckUpdates refreshes apt and lists upgradable packages
func CheckUpdates() ([]UpdatePackage, error) {
	if runtime.GOOS == "darwin" {
		// Mock Data for Mac
		return []UpdatePackage{
			{"nginx", "1.24.0-ubuntu3", "Security"},
			{"openssl", "3.0.2-0ubuntu1.10", "Security"},
			{"curl", "7.81.0-1ubuntu1.14", "Bugfix"},
			{"linux-image-generic", "5.15.0-86.96", "Kernel"},
			{"php8.3-fpm", "8.3.0-1+ubuntu22.04.1", "Feature"},
		}, nil
	}

	// 1. Run apt-get update to fetch latest lists (Silent)
	// We ignore errors here because sometimes repos are flaky, but list might still work
	exec.Command("apt-get", "update", "-q").Run()

	// 2. List upgradable packages
	// Format: package_name/focal-updates 1.2.3 amd64 [upgradable from: 1.2.2]
	cmd := exec.Command("apt", "list", "--upgradable")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list updates: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	var packages []UpdatePackage

	for _, line := range lines {
		// Skip header "Listing..." and empty lines
		if strings.HasPrefix(line, "Listing") || strings.TrimSpace(line) == "" {
			continue
		}

		// Simple parsing
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// parts[0] is usually "nginx/jammy-updates"
			name := strings.Split(parts[0], "/")[0]
			version := parts[1]
			
			// Simple heuristics for type based on name
			updateType := "General"
			if strings.Contains(name, "security") || strings.Contains(name, "ssl") {
				updateType = "Security"
			} else if strings.Contains(name, "linux-image") {
				updateType = "Kernel"
			}

			packages = append(packages, UpdatePackage{
				Name:    name,
				Version: version,
				Type:    updateType,
			})
		}
	}

	return packages, nil
}

// RunUpgrade performs the system upgrade
func RunUpgrade() error {
	if runtime.GOOS == "darwin" {
		fmt.Println("[MOCK] apt-get upgrade -y executed")
		return nil
	}

	// Non-interactive upgrade
	// DEBIAN_FRONTEND=noninteractive prevents prompts
	cmd := exec.Command("bash", "-c", "DEBIAN_FRONTEND=noninteractive apt-get upgrade -y")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("upgrade failed: %v", err)
	}
	return nil
}
