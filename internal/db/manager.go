package db

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CheckPhpMyAdmin checks if it's already installed
func CheckPhpMyAdmin() bool {
	// Mock Mode
	if runtime.GOOS == "darwin" {
		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, "mock_nginx/html/phpmyadmin")
		_, err := os.Stat(path)
		return err == nil
	}

	// Real Path
	_, err := os.Stat("/var/www/html/phpmyadmin")
	return err == nil
}

// InstallPhpMyAdmin downloads and configures it
func InstallPhpMyAdmin() error {
	if runtime.GOOS == "darwin" {
		// Mock Installation
		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, "mock_nginx/html/phpmyadmin")
		os.MkdirAll(path, 0755)
		os.WriteFile(filepath.Join(path, "index.php"), []byte("<h1>phpMyAdmin Mock</h1>"), 0644)
		fmt.Println("[MOCK] phpMyAdmin Installed to " + path)
		return nil
	}

	// Real Installation Script
	// 1. Download
	cmd := exec.Command("bash", "-c", `
		cd /var/www/html
		wget https://www.phpmyadmin.net/downloads/phpMyAdmin-latest-english.tar.gz
		tar xvf phpMyAdmin-latest-english.tar.gz
		mv phpMyAdmin-*-english phpmyadmin
		rm phpMyAdmin-latest-english.tar.gz
		
		# Config
		cd phpmyadmin
		cp config.sample.inc.php config.inc.php
		# Generate random secret for blowfish
		SECRET=$(openssl rand -base64 32 | tr -d /=+)
		sed -i "s/\\['blowfish_secret'\\] = ''/\\['blowfish_secret'\\] = '$SECRET'/" config.inc.php
		
		# Permissions
		chown -R www-data:www-data /var/www/html/phpmyadmin
	`)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install failed: %s", string(output))
	}

	return nil
}
