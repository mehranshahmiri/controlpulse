package nginx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Site represents a website for the UI
type Site struct {
	Domain string
	Status string // "Active" or "Disabled"
}

// Global config paths (Defaults for Linux)
var (
	SitesAvailable = "/etc/nginx/sites-available"
	SitesEnabled   = "/etc/nginx/sites-enabled"
	WebRoot        = "/var/www/html"
)

// init runs when the program starts.
// It checks if we are on a Mac and switches to "Mock Mode" paths.
func init() {
	if runtime.GOOS == "darwin" {
		// Get the current project directory
		cwd, _ := os.Getwd()

		// Redirect paths to a local folder so we can test safely
		SitesAvailable = filepath.Join(cwd, "mock_nginx/sites-available")
		SitesEnabled = filepath.Join(cwd, "mock_nginx/sites-enabled")
		WebRoot = filepath.Join(cwd, "mock_nginx/html")

		// Ensure these folders actually exist
		os.MkdirAll(SitesAvailable, 0755)
		os.MkdirAll(SitesEnabled, 0755)
		os.MkdirAll(WebRoot, 0755)

		fmt.Println("[INIT] Running in Mock Mode. Files will be in ./mock_nginx/")
	}
}

// ListSites reads the config directory to find sites
func ListSites() ([]Site, error) {
	if runtime.GOOS == "darwin" {
		fmt.Printf("[DEBUG] ListSites: Reading from %s\n", SitesAvailable)
	}

	entries, err := os.ReadDir(SitesAvailable)
	if err != nil {
		if runtime.GOOS == "darwin" {
			fmt.Printf("[DEBUG] ListSites Error: %v\n", err)
		}
		// If folder doesn't exist yet, return empty list
		return []Site{}, nil
	}

	sites := []Site{} // Initialize as empty slice
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// Ignore hidden files (like .DS_Store on Mac)
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}

		status := "Disabled"
		// Check if the symlink exists in the 'enabled' folder
		// os.Stat follows symlinks, so it confirms the link AND the target exist
		if _, err := os.Stat(filepath.Join(SitesEnabled, e.Name())); err == nil {
			status = "Active"
		}

		sites = append(sites, Site{
			Domain: e.Name(),
			Status: status,
		})
	}

	if runtime.GOOS == "darwin" {
		fmt.Printf("[DEBUG] ListSites: Found %d sites\n", len(sites))
	}

	return sites, nil
}

// CreateSite generates the config files
func CreateSite(domain string) error {
	// 1. Sanitize Domain (Prevent hacks like "../")
	if strings.Contains(domain, "/") || strings.Contains(domain, "\\") || strings.Contains(domain, "..") {
		return fmt.Errorf("invalid domain name")
	}

	// 2. Prepare Nginx Config with MAGIC ALIAS
	// We add: *.nip.io so the preview link works!
	config := fmt.Sprintf(`server {
    listen 80;
    server_name %s %s.*.nip.io;
    root %s/%s;
    index index.php index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php8.3-fpm.sock;
    }
}`, domain, domain, WebRoot, domain)

	// 3. Define Paths
	configPath := filepath.Join(SitesAvailable, domain)
	linkPath := filepath.Join(SitesEnabled, domain)
	siteRoot := filepath.Join(WebRoot, domain)

	// 4. Create the Web Root Folder & default index.html
	if err := os.MkdirAll(siteRoot, 0755); err != nil {
		return fmt.Errorf("failed to create web root: %v", err)
	}
	// Create a dummy index file so the user sees something
	os.WriteFile(filepath.Join(siteRoot, "index.html"), []byte("<h1>Welcome to "+domain+"</h1>"), 0644)

	// 5. Write the Nginx Config File
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}
	// Debug Log
	if runtime.GOOS == "darwin" {
		fmt.Printf("[DEBUG] Wrote config to: %s\n", configPath)
	}

	// 6. Enable the Site (Symlink)
	// Remove existing link if it exists just in case
	os.Remove(linkPath)
	if err := os.Symlink(configPath, linkPath); err != nil {
		return fmt.Errorf("failed to link site: %v", err)
	}
	// Debug Log
	if runtime.GOOS == "darwin" {
		fmt.Printf("[DEBUG] Linked %s -> %s\n", linkPath, configPath)
	}

	// 7. Reload Nginx (or Print if Mock)
	return reloadNginx()
}

// DeleteSite removes config and web files
func DeleteSite(domain string) error {
	// Sanitize
	if strings.Contains(domain, "/") || domain == "" || domain == "default" {
		return fmt.Errorf("invalid domain")
	}

	// 1. Remove Configs
	os.Remove(filepath.Join(SitesEnabled, domain))
	os.Remove(filepath.Join(SitesAvailable, domain))

	// 2. Remove Web Files (Optional: You might want to keep data)
	// For MVP, we delete it to be clean.
	os.RemoveAll(filepath.Join(WebRoot, domain))

	return reloadNginx()
}

// reloadNginx abstracts the system command
func reloadNginx() error {
	if runtime.GOOS == "darwin" {
		fmt.Println("[MOCK] Nginx Reloaded (Check the ./mock_nginx folder)")
		return nil
	}
	// On Linux, actually reload the service
	return exec.Command("systemctl", "reload", "nginx").Run()
}
