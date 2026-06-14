package security

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Jail struct {
	Name      string
	Active    bool
	BannedIPs []string
}

// ListJails returns all configured jails and their banned IPs
func ListJails() ([]Jail, error) {
	if runtime.GOOS == "darwin" {
		return mockJails(), nil
	}

	// 1. Get list of jails
	// Output: "Current list of jails: sshd, nginx-http-auth"
	cmd := exec.Command("fail2ban-client", "status")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fail2ban not installed or running")
	}

	output := string(out)
	var jails []Jail

	// Parse Jails
	if strings.Contains(output, "Jail list:") {
		parts := strings.Split(output, "Jail list:")
		if len(parts) > 1 {
			jailNames := strings.Split(strings.TrimSpace(parts[1]), ",")
			for _, name := range jailNames {
				name = strings.TrimSpace(name)
				if name == "" { continue }
				
				// Get details for this jail
				ips := getBannedIPs(name)
				jails = append(jails, Jail{
					Name:      name,
					Active:    true,
					BannedIPs: ips,
				})
			}
		}
	}

	return jails, nil
}

// getBannedIPs fetches status for a specific jail
func getBannedIPs(jail string) []string {
	// fail2ban-client status sshd
	out, err := exec.Command("fail2ban-client", "status", jail).Output()
	if err != nil { return []string{} }

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Banned IP list:") {
			// Format: "Banned IP list: 1.2.3.4 5.6.7.8"
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				ipStr := strings.TrimSpace(parts[1])
				if ipStr == "" { return []string{} }
				return strings.Fields(ipStr)
			}
		}
	}
	return []string{}
}

// UnbanIP removes an IP from a jail
func UnbanIP(jail, ip string) error {
	if runtime.GOOS == "darwin" {
		fmt.Printf("[MOCK] fail2ban-client set %s unbanip %s\n", jail, ip)
		return nil
	}
	return exec.Command("fail2ban-client", "set", jail, "unbanip", ip).Run()
}

func mockJails() []Jail {
	return []Jail{
		{
			Name:      "sshd",
			Active:    true,
			BannedIPs: []string{"192.168.1.55", "10.0.0.4", "203.0.113.1"},
		},
		{
			Name:      "nginx-http-auth",
			Active:    true,
			BannedIPs: []string{"45.33.22.11"},
		},
		{
			Name:      "nginx-botsearch",
			Active:    true,
			BannedIPs: []string{}, // Empty
		},
	}
}
