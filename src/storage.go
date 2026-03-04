package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StorageManager handles all blaze package storage
type StorageManager struct {
	BlazeDir string // ~/.blaze
	PackagesDir string // ~/.blaze/packages
	RegistryPath string // ~/.blaze/registry.json
}

//NewStorageManager intializes the storage manager
func NewStorageManager() (*StorageManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	blazeDir := filepath.Join(homeDir, ".blaze")
	packagesDir := filepath.Join(blazeDir, "packages")
	registryPath := filepath.Join(blazeDir, "registry.json")

	// Create directories if they don't exist
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blaze directories: %w", err)
	}

	return &StorageManager{
		BlazeDir: blazeDir,
		PackagesDir: packagesDir,
		RegistryPath: registryPath,
	}, nil
}

//GetPackagePath returns the path where a package version is stored
func (sm *StorageManager) GetPackagePath(name, version string) string {
	return filepath.Join(sm.PackagesDir, name, version)
}

// LoadRegistry reads the registry file
func (sm *StorageManager) LoadRegistry() (map[string][]string, error) {
	registry := make(map[string][]string)

	data, err := os.ReadFile(sm.RegistryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}
	
	return registry, nil
}

//SaveRegistry writes the registry file
func (sm *StorageManager) SaveRegistry(registry map[string][]string) error {
	data, err := json.MarshalIndent(registry, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(sm.RegistryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}