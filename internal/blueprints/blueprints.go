package blueprints

import (
	"archive/zip"
	"controlp/internal/db"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// WebRoot determines where websites are stored.
var WebRoot = "/var/www/html"

func init() {
	if runtime.GOOS != "linux" {
		wd, _ := os.Getwd()
		WebRoot = filepath.Join(wd, "www")
		_ = os.MkdirAll(WebRoot, 0755)
	} else {
		// Only check if missing, don't force fail here, let install handle it via sudo
		if _, err := os.Stat(WebRoot); os.IsNotExist(err) {
			// Try to create, ignoring error (will fail if not root, handled later)
			_ = os.MkdirAll(WebRoot, 0755)
		}
	}
}

// ProgressFunc defines the callback for installation progress updates
type ProgressFunc func(step string, percent int)

type App struct {
	ID, Name, Description, Icon, Color string
}

func List() []App {
	return []App{
		{"wordpress", "WordPress", "The world's most popular CMS.", "ph-wordpress-logo", "text-blue-500"},
		{"laravel", "Laravel", "A PHP web application framework.", "ph-file-php", "text-red-500"},
	}
}

// executePrivileged runs a command, using sudo with password if provided
func executePrivileged(rootPass string, name string, args ...string) error {
	if rootPass != "" && runtime.GOOS != "windows" {
		// Prepare sudo command: echo PASS | sudo -S command args...
		cmdStr := fmt.Sprintf("echo '%s' | sudo -S %s %s", rootPass, name, strings.Join(args, " "))
		// Use shell to handle the pipe
		cmd := exec.Command("sh", "-c", cmdStr)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("sudo failed: %v, output: %s", err, string(out))
		}
		return nil
	}

	// Fallback to direct execution (works if running as root)
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("command failed: %v, output: %s", err, string(out))
	}
	return nil
}

func InstallWordPress(domain, dbSuffix, dbUserSuffix, rootPass string, onProgress ProgressFunc) error {
	rootPath := filepath.Join(WebRoot, domain)

	if onProgress != nil {
		onProgress("Preparing directory...", 10)
	}

	// 1. Create Directory (Privileged)
	if err := executePrivileged(rootPass, "mkdir", "-p", rootPath); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 2. Download
	if onProgress != nil {
		onProgress("Downloading WordPress...", 30)
	}
	resp, err := http.Get("https://wordpress.org/latest.zip")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpZip := filepath.Join(os.TempDir(), "wp_"+domain+".zip")
	out, err := os.Create(tmpZip)
	if err != nil {
		return err
	}
	io.Copy(out, resp.Body)
	out.Close()
	defer os.Remove(tmpZip)

	// 3. Extract (Standard User usually fine for Temp, but need to move to Privileged dir)
	if onProgress != nil {
		onProgress("Extracting...", 50)
	}
	// Unzip to temp folder first to avoid permission issues during unzip logic
	tmpExtract := filepath.Join(os.TempDir(), "wp_extract_"+domain)
	os.RemoveAll(tmpExtract)
	if err := unzip(tmpZip, tmpExtract); err != nil {
		return err
	}

	// Move files from temp/wordpress to rootPath (Privileged Move)
	// We use 'cp -r' then 'rm' to avoid cross-device link errors, wrapped in sudo
	srcPath := filepath.Join(tmpExtract, "wordpress") + "/."
	if err := executePrivileged(rootPass, "cp", "-r", srcPath, rootPath); err != nil {
		return fmt.Errorf("failed to move files: %v", err)
	}
	os.RemoveAll(tmpExtract)

	// 5. DB
	if onProgress != nil {
		onProgress("Provisioning Database...", 75)
	}
	creds, err := db.CreateDatabase("db_"+dbSuffix, "user_"+dbUserSuffix)
	if err != nil {
		return err
	}

	// 6. Config
	if onProgress != nil {
		onProgress("Configuring...", 90)
	}
	configContent := fmt.Sprintf(`<?php
define( 'DB_NAME', '%s' );
define( 'DB_USER', '%s' );
define( 'DB_PASSWORD', '%s' );
define( 'DB_HOST', 'localhost' );
define( 'DB_CHARSET', 'utf8' );
define( 'DB_COLLATE', '' );
$table_prefix = 'wp_';
define( 'WP_DEBUG', false );
if ( ! defined( 'ABSPATH' ) ) { define( 'ABSPATH', __DIR__ . '/' ); }
require_once ABSPATH . 'wp-settings.php';
`, creds.DBName, creds.Username, creds.Password)

	// Write config to temp file then move (Privileged)
	tmpConfig := filepath.Join(os.TempDir(), "wp-config.php")
	os.WriteFile(tmpConfig, []byte(configContent), 0644)
	executePrivileged(rootPass, "mv", tmpConfig, filepath.Join(rootPath, "wp-config.php"))

	// 7. Permissions
	if onProgress != nil {
		onProgress("Finalizing permissions...", 95)
	}
	executePrivileged(rootPass, "chown", "-R", "www-data:www-data", rootPath)
	executePrivileged(rootPass, "chmod", "-R", "755", rootPath)

	return nil
}

func InstallLaravel(domain, rootPass string, onProgress ProgressFunc) error {
	rootPath := filepath.Join(WebRoot, domain)

	if onProgress != nil {
		onProgress("Preparing directory...", 10)
	}
	if err := executePrivileged(rootPass, "mkdir", "-p", rootPath); err != nil {
		return err
	}

	// Laravel Composer create-project is tricky with sudo because composer shouldn't run as root
	// Strategy: Create as current user (or www-data if possible), then fix permissions.
	// For simplicity in this context, we run in place then fix permissions.

	if onProgress != nil {
		onProgress("Running Composer...", 30)
	}

	// Check if directory is empty, composer requires it
	// We might need to clear it (Privileged)
	// Warning: Dangerous if domain is wrong.

	cmd := exec.Command("composer", "create-project", "laravel/laravel", ".", "--prefer-dist", "--no-interaction")
	cmd.Dir = rootPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("composer failed: %v, out: %s", err, string(out))
	}

	if onProgress != nil {
		onProgress("Setting permissions...", 90)
	}
	storagePath := filepath.Join(rootPath, "storage")
	bootstrapPath := filepath.Join(rootPath, "bootstrap/cache")

	executePrivileged(rootPass, "chown", "-R", "www-data:www-data", rootPath)
	executePrivileged(rootPass, "chmod", "-R", "775", storagePath)
	executePrivileged(rootPass, "chmod", "-R", "775", bootstrapPath)

	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}
