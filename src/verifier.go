package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// VerifyAndDownload downloads a package and verifies its SHA256 hash
func VerifyAndDownload(url, expectedSHA256, destPath string) error {
	// Validate URL scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("only http and https URLs are supported, got: %s", url)
	}

	expectedSHA256 = normalizeSHA256(expectedSHA256)

	tempPath := destPath + ".tmp"
	downloadCommitted := false
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear temporary download file: %w", err)
	}

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download package: HTTP %d", resp.StatusCode)
	}

	// Calculate SHA256 while reading
	hash := sha256.New()
	tee := io.TeeReader(resp.Body, hash)

	// Write to destination
	file, err := createFile(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		file.Close()
		if !downloadCommitted {
			os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(file, tee); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Verify hash
	actualSHA256 := fmt.Sprintf("%x", hash.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to finalize downloaded file: %w", err)
	}

	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to replace existing archive: %w", err)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		return fmt.Errorf("failed to move verified archive into place: %w", err)
	}

	downloadCommitted = true

	return nil
}

// createFile creates a file and its parent directories
func createFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func normalizeSHA256(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.TrimPrefix(value, "sha256:")
}
