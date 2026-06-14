package firewall

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Rule struct {
	Port   string
	Proto  string // tcp/udp
	Action string // ALLOW/DENY
	From   string // Any or specific IP
}

// ListRules parses 'ufw status numbered'
func ListRules() ([]Rule, error) {
	if runtime.GOOS == "darwin" {
		return []Rule{
			{"22", "TCP", "ALLOW", "Anywhere"},
			{"80", "TCP", "ALLOW", "Anywhere"},
			{"443", "TCP", "ALLOW", "Anywhere"},
			{"8888", "TCP", "ALLOW", "192.168.1.50"},
		}, nil
	}

	cmd := exec.Command("ufw", "status")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var rules []Rule

	// Skip header lines (Status: active, To Action From, -- -- --)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Basic parser logic (This assumes standard UFW output format)
		// 80/tcp                     ALLOW       Anywhere
		parts := strings.Fields(line)
		if len(parts) >= 3 && (parts[1] == "ALLOW" || parts[1] == "DENY") {
			// Split port/proto
			pp := strings.Split(parts[0], "/")
			proto := "TCP"
			if len(pp) > 1 {
				proto = strings.ToUpper(pp[1])
			}

			rules = append(rules, Rule{
				Port:   pp[0],
				Proto:  proto,
				Action: parts[1],
				From:   strings.Join(parts[2:], " "),
			})
		}
	}
	return rules, nil
}

// AddRule opens a port
func AddRule(port, proto string) error {
	if runtime.GOOS == "darwin" {
		fmt.Printf("[MOCK] ufw allow %s/%s\n", port, proto)
		return nil
	}
	// ufw allow 8080/tcp
	return exec.Command("ufw", "allow", fmt.Sprintf("%s/%s", port, proto)).Run()
}

// DeleteRule closes a port
// Note: Deleting by rule text is safer than ID which changes
func DeleteRule(port, proto string) error {
	if runtime.GOOS == "darwin" {
		fmt.Printf("[MOCK] ufw delete allow %s/%s\n", port, proto)
		return nil
	}
	return exec.Command("ufw", "delete", "allow", fmt.Sprintf("%s/%s", port, proto)).Run()
}
