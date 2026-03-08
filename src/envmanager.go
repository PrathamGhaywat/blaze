package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EnvManager handles PATH and environment variable updates
type EnvManager struct {
	isWindows bool
}

// NewEnvManager creates a new environment manager
func NewEnvManager() *EnvManager {
	return &EnvManager{
		isWindows: runtime.GOOS == "windows",
	}
}

// AddToPath adds a directory to the system PATH
func (em *EnvManager) AddToPath(binDir string) error {
	if em.isWindows {
		return em.addToPathWindows(binDir)
	}
	return em.addToPathUnix(binDir)
}

// ListPathEntries returns the persisted PATH entries used by Blaze.
func (em *EnvManager) ListPathEntries() ([]string, error) {
	if em.isWindows {
		currentPath, err := em.getWindowsUserPath()
		if err != nil {
			return nil, err
		}
		return splitPathEntries(currentPath), nil
	}

	return splitPathEntries(os.Getenv("PATH")), nil
}

// addToPathWindows adds to PATH on Windows via Registry (no 1024 char limit)
func (em *EnvManager) addToPathWindows(binDir string) error {
	currentPath, err := em.getWindowsUserPath()
	if err != nil {
		return err
	}

	if containsPathEntry(splitPathEntries(currentPath), binDir) {
		return nil // Already in PATH
	}

	entries := splitPathEntries(currentPath)
	entries = append(entries, binDir)
	if err := em.setWindowsUserPath(strings.Join(entries, string(os.PathListSeparator))); err != nil {
		return err
	}

	fmt.Printf("✓ PATH updated via Registry\n")
	fmt.Printf("⚠️  You MUST CLOSE and REOPEN your terminal for PATH changes to take effect\n")

	return nil
}

// addToPathUnix adds to PATH on Unix-like systems
func (em *EnvManager) addToPathUnix(binDir string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Detect shell profile files
	profileFiles := []string{
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".profile"),
	}

	pathExportLine := fmt.Sprintf(`export PATH="%s:$PATH"`, binDir)

	for _, profileFile := range profileFiles {
		if _, err := os.Stat(profileFile); err == nil {
			// File exists, check if already added
			content, err := os.ReadFile(profileFile)
			if err != nil {
				continue
			}

			if strings.Contains(string(content), binDir) {
				return nil // Already in PATH
			}

			// Append to profile
			file, err := os.OpenFile(profileFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				continue
			}

			fmt.Fprintf(file, "\n%s\n", pathExportLine)
			file.Close()
			return nil
		}
	}

	// Create .bashrc if nothing exists
	profileFile := filepath.Join(homeDir, ".bashrc")
	file, err := os.OpenFile(profileFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to update shell profile: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "\n%s\n", pathExportLine)
	return nil
}

// RemoveFromPath removes a directory from the system PATH
func (em *EnvManager) RemoveFromPath(binDir string) error {
	if em.isWindows {
		return em.removeFromPathWindows(binDir)
	}
	return em.removeFromPathUnix(binDir)
}

// removeFromPathWindows removes from PATH on Windows
func (em *EnvManager) removeFromPathWindows(binDir string) error {
	currentPath, err := em.getWindowsUserPath()
	if err != nil {
		return err
	}

	pathParts := splitPathEntries(currentPath)

	var newPathParts []string
	for _, part := range pathParts {
		if !samePathEntry(part, binDir) {
			newPathParts = append(newPathParts, part)
		}
	}

	newPath := strings.Join(newPathParts, string(os.PathListSeparator))
	if err := em.setWindowsUserPath(newPath); err != nil {
		return err
	}

	return nil
}

// removeFromPathUnix removes from PATH on Unix-like systems
func (em *EnvManager) removeFromPathUnix(binDir string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	profileFiles := []string{
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".profile"),
	}

	for _, profileFile := range profileFiles {
		content, err := os.ReadFile(profileFile)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		var newLines []string

		for _, line := range lines {
			if !strings.Contains(line, binDir) {
				newLines = append(newLines, line)
			}
		}

		if err := os.WriteFile(profileFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			continue
		}
	}

	return nil
}

func splitPathEntries(pathValue string) []string {
	if pathValue == "" {
		return []string{}
	}

	parts := strings.Split(pathValue, string(os.PathListSeparator))
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			entries = append(entries, trimmed)
		}
	}

	return entries
}

func containsPathEntry(entries []string, candidate string) bool {
	for _, entry := range entries {
		if samePathEntry(entry, candidate) {
			return true
		}
	}

	return false
}

func samePathEntry(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func (em *EnvManager) getWindowsUserPath() (string, error) {
	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-Command",
		"$value = (Get-ItemProperty -Path 'HKCU:\\Environment' -Name PATH -ErrorAction SilentlyContinue).PATH; if ($null -eq $value) { '' } else { $value }",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to read Windows PATH from registry: %w\noutput: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func (em *EnvManager) setWindowsUserPath(pathValue string) error {
	escapedPathValue := strings.ReplaceAll(pathValue, "'", "''")
	psCmd := fmt.Sprintf("Set-ItemProperty -Path 'HKCU:\\Environment' -Name PATH -Value '%s'", escapedPathValue)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update PATH: %w\noutput: %s", err, string(output))
	}

	return nil
}
