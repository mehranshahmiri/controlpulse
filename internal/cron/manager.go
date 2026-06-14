package cron

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Job struct {
	ID       int // Line number acts as ID
	Schedule string
	Command  string
}

// Mock path for Mac development
var mockCronFile = "mock_cron"

func init() {
	if runtime.GOOS == "darwin" {
		cwd, _ := os.Getwd()
		mockCronFile = filepath.Join(cwd, "mock_cron")
		// Ensure file exists for testing
		if _, err := os.Stat(mockCronFile); os.IsNotExist(err) {
			os.WriteFile(mockCronFile, []byte("# Mock Crontab\n"), 0644)
		}
		fmt.Println("[INIT] Cron Manager in Mock Mode (File: ./mock_cron)")
	}
}

// ListJobs reads the current crontab
func ListJobs() ([]Job, error) {
	var content string

	if runtime.GOOS == "darwin" {
		b, err := os.ReadFile(mockCronFile)
		if err != nil {
			return nil, err
		}
		content = string(b)
	} else {
		cmd := exec.Command("crontab", "-l")
		output, _ := cmd.Output() // Ignore error (exit status 1 means empty crontab)
		content = string(output)
	}

	lines := strings.Split(content, "\n")
	var jobs []Job

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Schedule vs Command
		parts := strings.Fields(line)
		if len(parts) > 5 {
			jobs = append(jobs, Job{
				ID:       i,
				Schedule: strings.Join(parts[:5], " "),
				Command:  strings.Join(parts[5:], " "),
			})
		}
	}
	return jobs, nil
}

// AddJob appends a new job
func AddJob(schedule, command string) error {
	// Basic sanitization
	if strings.Contains(command, "\n") {
		return fmt.Errorf("invalid command")
	}

	// FIX: Removed unused 'jobs' variable definition

	line := fmt.Sprintf("%s %s", schedule, command)

	if runtime.GOOS == "darwin" {
		f, err := os.OpenFile(mockCronFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.WriteString(line + "\n"); err != nil {
			return err
		}
		return nil
	}

	// Linux: Read current, append, write back
	current, _ := exec.Command("crontab", "-l").Output()
	newContent := string(current) + line + "\n"
	return saveCrontab(newContent)
}

// DeleteJob removes a job by index
func DeleteJob(targetID int) error {
	var content string
	if runtime.GOOS == "darwin" {
		b, _ := os.ReadFile(mockCronFile)
		content = string(b)
	} else {
		out, _ := exec.Command("crontab", "-l").Output()
		content = string(out)
	}

	lines := strings.Split(content, "\n")
	var newLines []string

	for i, line := range lines {
		// Skip the target line
		if i == targetID {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		} // Clean up empty lines
		newLines = append(newLines, line)
	}

	finalContent := strings.Join(newLines, "\n") + "\n"

	if runtime.GOOS == "darwin" {
		return os.WriteFile(mockCronFile, []byte(finalContent), 0644)
	}
	return saveCrontab(finalContent)
}

func saveCrontab(content string) error {
	// Write to temp file
	tmp := "/tmp/controlp_cron_edit"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}

	// Load into crontab
	if err := exec.Command("crontab", tmp).Run(); err != nil {
		return err
	}
	return nil
}
