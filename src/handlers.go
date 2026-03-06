package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// handleAdd fetches, downloads, extracts, and registers a package
func handleAdd(manifestURL string) error {
	fmt.Printf("📦 Adding package from: %s\n", manifestURL)

	// Fetch and parse manifest
	manifest, err := FetchManifest(manifestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	fmt.Printf("✓ Manifest loaded: %s@%s\n", manifest.Name, manifest.Version)

	// Get the target for current OS/Arch
	target := getTargetForOS(manifest)
	if target == nil {
		return fmt.Errorf("no compatible target found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Setup storage
	sm, err := NewStorageManager()
	if err != nil {
		return fmt.Errorf("failed to setup storage: %w", err)
	}

	pkgPath := sm.GetPackagePath(manifest.Name, manifest.Version)
	if _, err := os.Stat(pkgPath); err == nil {
		return fmt.Errorf("package %s@%s is already installed", manifest.Name, manifest.Version)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check existing installation: %w", err)
	}

	archivePath := filepath.Join(pkgPath, "archive")
	extractPath := filepath.Join(pkgPath, "content")
	addedPathDirs := make([]string, 0)
	installCommitted := false

	defer func() {
		if installCommitted {
			return
		}

		for index := len(addedPathDirs) - 1; index >= 0; index-- {
			if err := NewEnvManager().RemoveFromPath(addedPathDirs[index]); err != nil {
				fmt.Printf("⚠️  Failed to roll back PATH entry %s: %v\n", addedPathDirs[index], err)
			}
		}

		if err := os.RemoveAll(pkgPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("⚠️  Failed to clean up partial install at %s: %v\n", pkgPath, err)
		}
	}()

	fmt.Printf("⬇️  Downloading and verifying package...\n")

	// Download and verify SHA256
	if err := VerifyAndDownload(target.URL, target.SHA256, archivePath); err != nil {
		return fmt.Errorf("failed to download/verify package: %w", err)
	}

	fmt.Printf("✓ Download verified\n")

	// Extract archive
	fmt.Printf("📂 Extracting archive...\n")

	if err := ExtractArchive(archivePath, extractPath, target.ArchiveType); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	fmt.Printf("✓ Archive extracted\n")

	// Add binaries to PATH
	em := NewEnvManager()
	seenBinDirs := make(map[string]struct{})
	for _, bin := range target.Bin {
		binPath := filepath.Join(extractPath, bin)
		binDir := filepath.Dir(binPath)

		fmt.Printf("🔗 Adding to PATH: %s\n", binDir)

		// Check if binary exists
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			fmt.Printf("   ⚠️  Binary not found at: %s\n", binPath)
			fmt.Printf("   Available files in extract path:\n")

			// List what's actually there
			filepath.Walk(extractPath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info == nil {
					return nil
				}
				if !info.IsDir() && strings.Contains(strings.ToLower(info.Name()), strings.ToLower(filepath.Base(binPath))) {
					fmt.Printf("      - %s\n", path)
				}
				return nil
			})

			return fmt.Errorf("binary not found at expected path: %s", binPath)
		} else if err != nil {
			return fmt.Errorf("failed to validate binary path %s: %w", binPath, err)
		}

		if _, exists := seenBinDirs[binDir]; exists {
			continue
		}

		if err := em.AddToPath(binDir); err != nil {
			return fmt.Errorf("failed to add to PATH: %w", err)
		}

		seenBinDirs[binDir] = struct{}{}
		addedPathDirs = append(addedPathDirs, binDir)
	}

	// Update registry
	registry, err := sm.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	for _, version := range registry[manifest.Name] {
		if version == manifest.Version {
			installCommitted = true
			fmt.Printf("✅ Package already registered: %s@%s\n", manifest.Name, manifest.Version)
			return nil
		}
	}

	registry[manifest.Name] = append(registry[manifest.Name], manifest.Version)
	if err := sm.SaveRegistry(registry); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	installCommitted = true

	fmt.Printf("✅ Package installed: %s@%s\n", manifest.Name, manifest.Version)
	return nil
}

// getTargetForOS returns the target matching current OS and architecture
func getTargetForOS(manifest *Manifest) *Target {
	osArch := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	if target, exists := manifest.Targets[osArch]; exists {
		return &target
	}

	// Fallback: try just the OS
	for key, target := range manifest.Targets {
		if strings.HasPrefix(key, runtime.GOOS) {
			return &target
		}
	}

	return nil
}

// handleRemove removes a package version
func handleRemove(pkgName, version string, all bool) error {
	fmt.Printf("🗑️  Removing package: %s\n", pkgName)

	sm, err := NewStorageManager()
	if err != nil {
		return fmt.Errorf("failed to setup storage: %w", err)
	}

	// Load registry
	registry, err := sm.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	versions, exists := registry[pkgName]
	if !exists || len(versions) == 0 {
		return fmt.Errorf("package %s not found", pkgName)
	}

	// Remove all versions if --all flag
	if all {
		for _, v := range versions {
			pkgPath := sm.GetPackagePath(pkgName, v)
			if err := os.RemoveAll(pkgPath); err != nil {
				return fmt.Errorf("failed to remove package directory: %w", err)
			}
		}
		delete(registry, pkgName)
		if err := sm.SaveRegistry(registry); err != nil {
			return fmt.Errorf("failed to save registry: %w", err)
		}
		fmt.Printf("✅ Removed all versions of %s (%d versions)\n", pkgName, len(versions))
		return nil
	}

	// If version specified, remove only that version
	if version != "" {
		found := false
		for i, v := range versions {
			if v == version {
				registry[pkgName] = append(versions[:i], versions[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("version %s not found for package %s", version, pkgName)
		}
	} else {
		// If multiple versions exist without specifying one, error
		if len(versions) > 1 {
			versionsStr := strings.Join(versions, ", ")
			return fmt.Errorf("multiple versions of %s exist: %s. Specify version or use --all to remove all!", pkgName, versionsStr)
		}
		// Remove the only version
		version = versions[0]
		delete(registry, pkgName)
	}

	// Remove from filesystem
	pkgPath := sm.GetPackagePath(pkgName, version)
	if err := os.RemoveAll(pkgPath); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	// Update registry
	if len(registry[pkgName]) == 0 {
		delete(registry, pkgName)
	}

	if err := sm.SaveRegistry(registry); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("✅ Removed %s@%s\n", pkgName, version)
	return nil
}

// handleUse switches to a specific package version
func handleUse(pkgSpec string) error {
	parts := strings.Split(pkgSpec, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use: package@version")
	}

	pkgName := parts[0]
	version := parts[1]

	fmt.Printf("🔄 Switching to: %s@%s\n", pkgName, version)

	sm, err := NewStorageManager()
	if err != nil {
		return fmt.Errorf("failed to setup storage: %w", err)
	}

	// Load registry
	registry, err := sm.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	versions, exists := registry[pkgName]
	if !exists {
		return fmt.Errorf("package %s not installed", pkgName)
	}

	// Check if version exists
	found := false
	for _, v := range versions {
		if v == version {
			found = true
			break
		}
	}

	if !found {
		versionsStr := strings.Join(versions, ", ")
		return fmt.Errorf("version %s not found for %s. Available: %s", version, pkgName, versionsStr)
	}

	fmt.Printf("✅ Switched to %s@%s\n", pkgName, version)
	return nil
}

// handleList lists all installed packages
func handleList() error {
	sm, err := NewStorageManager()
	if err != nil {
		return fmt.Errorf("failed to setup storage: %w", err)
	}

	registry, err := sm.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if len(registry) == 0 {
		fmt.Println("📋 No packages installed")
		return nil
	}

	fmt.Println("📋 Installed packages:")
	for pkgName, versions := range registry {
		for _, version := range versions {
			fmt.Printf("  • %s@%s\n", pkgName, version)
		}
	}

	return nil
}
