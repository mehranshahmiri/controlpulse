package fs

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config: Default to Linux path
var RootDir = "/var/www/html"

// Init: Detect Mac and sync with Nginx Mock path
func init() {
	if runtime.GOOS == "darwin" {
		cwd, _ := os.Getwd()
		RootDir = filepath.Join(cwd, "mock_nginx/html")
		// Ensure it exists
		os.MkdirAll(RootDir, 0755)
	}
}

// FileEntry represents a file or folder for the UI
type FileEntry struct {
	Name  string
	Path  string
	IsDir bool
	Size  string
}

// Security: Ensure path is inside RootDir
func getSafePath(userPath string) (string, error) {
	absRoot, _ := filepath.Abs(RootDir)
	target := filepath.Join(RootDir, userPath)
	absTarget, err := filepath.Abs(target)

	// Check for ".." hacks
	if err != nil || !strings.HasPrefix(absTarget, absRoot) {
		return "", fmt.Errorf("access denied")
	}
	return absTarget, nil
}

// ListDir returns folders first, then files
func ListDir(requestPath string) ([]FileEntry, error) {
	safePath, err := getSafePath(requestPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		return nil, err
	}

	var dirs, files []FileEntry

	for _, e := range entries {
		info, _ := e.Info()
		size := fmt.Sprintf("%d B", info.Size())
		if info.Size() > 1024 {
			size = fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
		}

		relPath := filepath.Join(requestPath, e.Name())
		entry := FileEntry{
			Name:  e.Name(),
			Path:  relPath,
			IsDir: e.IsDir(),
			Size:  size,
		}

		if e.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	return append(dirs, files...), nil
}

func ReadFile(requestPath string) (string, error) {
	safePath, err := getSafePath(requestPath)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(safePath)
	return string(content), err
}

func SaveFile(requestPath string, content string) error {
	safePath, err := getSafePath(requestPath)
	if err != nil {
		return err
	}
	return os.WriteFile(safePath, []byte(content), 0644)
}

// --- NEW FEATURES ---

func CreateItem(requestPath string, name string, isFolder bool) error {
	fullPath := filepath.Join(requestPath, name)
	safePath, err := getSafePath(fullPath)
	if err != nil {
		return err
	}

	if isFolder {
		return os.MkdirAll(safePath, 0755)
	}
	// Create empty file
	f, err := os.Create(safePath)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func DeleteItem(requestPath string) error {
	safePath, err := getSafePath(requestPath)
	if err != nil {
		return err
	}
	return os.RemoveAll(safePath)
}

func UploadFile(requestPath string, file *multipart.FileHeader) error {
	// Destination path
	safePath, err := getSafePath(filepath.Join(requestPath, file.Filename))
	if err != nil {
		return err
	}

	// Open Source
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Create Destination
	dst, err := os.Create(safePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	_, err = io.Copy(dst, src)
	return err
}
