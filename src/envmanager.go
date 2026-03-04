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

// addToPathWindows adds to PATH on Windows via setx
func (em *EnvManager) addToPathWindows(binDir string) error {
    // Get current PATH
    currentPath := os.Getenv("PATH")
    if strings.Contains(currentPath, binDir) {
        return nil // Already in PATH
    }

    // Use setx to set user PATH permanently
    cmd := exec.Command("setx", "PATH", currentPath+";"+binDir)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to update PATH on Windows: %w", err)
    }

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
    currentPath := os.Getenv("PATH")
    pathParts := strings.Split(currentPath, ";")

    var newPathParts []string
    for _, part := range pathParts {
        if part != binDir {
            newPathParts = append(newPathParts, part)
        }
    }

    newPath := strings.Join(newPathParts, ";")
    cmd := exec.Command("setx", "PATH", newPath)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to update PATH on Windows: %w", err)
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