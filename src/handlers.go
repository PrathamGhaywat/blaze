package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
	selectedTargetKey, target := getTargetForOS(manifest)
	if target == nil {
		return fmt.Errorf(
			"no compatible target found for runtime %s/%s. available targets: %s",
			runtime.GOOS,
			runtime.GOARCH,
			strings.Join(sortedTargetKeys(manifest.Targets), ", "),
		)
	}

	fmt.Printf("✓ Selected target: %s for runtime %s/%s\n", selectedTargetKey, runtime.GOOS, runtime.GOARCH)

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

// getTargetForOS returns the target matching current OS and architecture.
func getTargetForOS(manifest *Manifest) (string, *Target) {
	osName := runtime.GOOS
	archNames := platformArchAliases(runtime.GOARCH)

	for _, archName := range archNames {
		key := fmt.Sprintf("%s-%s", osName, archName)
		if target, exists := manifest.Targets[key]; exists {
			return key, &target
		}
	}

	for _, key := range sortedTargetKeys(manifest.Targets) {
		if !strings.HasPrefix(strings.ToLower(key), strings.ToLower(osName)+"-") {
			continue
		}

		target := manifest.Targets[key]
		return key, &target
	}

	return "", nil
}

func platformArchAliases(goArch string) []string {
	switch strings.ToLower(goArch) {
	case "amd64":
		return []string{"amd64", "x64", "x86_64"}
	case "386":
		return []string{"386", "x86", "i386"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	case "arm":
		return []string{"arm"}
	default:
		return []string{strings.ToLower(goArch)}
	}
}

func sortedTargetKeys(targets map[string]Target) []string {
	keys := make([]string, 0, len(targets))
	for key := range targets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

func handleCleanup() error {
	sm, err := NewStorageManager()
	if err != nil {
		return fmt.Errorf("failed to setup storage: %w", err)
	}

	em := NewEnvManager()
	registry, err := sm.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	removedRegistryEntries := 0
	cleanedRegistry := make(map[string][]string, len(registry))
	registryChanged := false

	for pkgName, versions := range registry {
		validVersions := make([]string, 0, len(versions))
		for _, version := range versions {
			contentPath := filepath.Join(sm.GetPackagePath(pkgName, version), "content")
			info, statErr := os.Stat(contentPath)
			if statErr == nil && info.IsDir() {
				validVersions = append(validVersions, version)
				continue
			}

			if statErr != nil && !os.IsNotExist(statErr) {
				return fmt.Errorf("failed to inspect package %s@%s: %w", pkgName, version, statErr)
			}

			fmt.Printf("🧹 Removing stale registry entry: %s@%s\n", pkgName, version)
			removedRegistryEntries++
			registryChanged = true
		}

		if len(validVersions) > 0 {
			cleanedRegistry[pkgName] = validVersions
		} else if len(versions) > 0 {
			registryChanged = true
		}
	}

	if registryChanged {
		if err := sm.SaveRegistry(cleanedRegistry); err != nil {
			return fmt.Errorf("failed to save cleaned registry: %w", err)
		}
	}

	pathEntries, err := em.ListPathEntries()
	if err != nil {
		return fmt.Errorf("failed to read PATH entries: %w", err)
	}

	removedPathEntries := 0
	for _, entry := range pathEntries {
		if !isBlazeManagedPath(sm, entry) {
			continue
		}

		info, statErr := os.Stat(entry)
		if statErr == nil && info.IsDir() {
			continue
		}

		if statErr != nil && !os.IsNotExist(statErr) {
			return fmt.Errorf("failed to inspect PATH entry %s: %w", entry, statErr)
		}

		fmt.Printf("🧹 Removing dead PATH entry: %s\n", entry)
		if err := em.RemoveFromPath(entry); err != nil {
			return fmt.Errorf("failed to remove dead PATH entry %s: %w", entry, err)
		}
		removedPathEntries++
	}

	if removedRegistryEntries == 0 && removedPathEntries == 0 {
		fmt.Println("🧹 Nothing to clean up")
		return nil
	}

	fmt.Printf("✅ Cleanup complete: removed %d stale registry entries and %d dead PATH entries\n", removedRegistryEntries, removedPathEntries)
	return nil
}

func isBlazeManagedPath(sm *StorageManager, candidate string) bool {
	cleanCandidate := filepath.Clean(candidate)
	cleanPackagesDir := filepath.Clean(sm.PackagesDir)
	packagesPrefix := cleanPackagesDir + string(os.PathSeparator)

	if runtime.GOOS == "windows" {
		candidateLower := strings.ToLower(cleanCandidate)
		packagesLower := strings.ToLower(cleanPackagesDir)
		return candidateLower == packagesLower || strings.HasPrefix(candidateLower, strings.ToLower(packagesPrefix))
	}

	return cleanCandidate == cleanPackagesDir || strings.HasPrefix(cleanCandidate, packagesPrefix)
}
