package ssl

import (
	"fmt"
	"os/exec"
	"runtime"
)

// EnableSSL runs certbot for the domain
func EnableSSL(domain string, email string) error {
	// Mock Mode for Mac
	if runtime.GOOS == "darwin" {
		fmt.Printf("[MOCK] Enabling SSL for %s (Email: %s)\n", domain, email)
		fmt.Println("[MOCK] Executing: certbot --nginx -d " + domain)
		return nil
	}

	// Real Command for Linux
	// --nginx: Use the Nginx plugin to edit config automatically
	// --non-interactive: Don't ask questions
	// --agree-tos: Agree to terms
	// --redirect: Force HTTPS redirect
	cmd := exec.Command("certbot", "--nginx", "-d", domain, 
		"--non-interactive", "--agree-tos", "--email", email, "--redirect")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("certbot failed: %s", string(output))
	}

	return nil
}
