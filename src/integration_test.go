package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFullFlow(t *testing.T) {
	// Create a mock HTTP server serving a manifest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := Manifest{
			Schema:      1,
			Name:        "test-pkg",
			Version:     "1.0.0",
			Description: "Test package",
			Homepage:    "https://example.com",
			License:     "MIT",
			Targets: map[string]Target{
				"windows-amd64": {
					ArchiveType: "zip",
					URL:         "https://example.com/fake.zip",
					SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					Bin:         []string{"bin/test.exe"},
				},
				"linux-amd64": {
					ArchiveType: "tar.gz",
					URL:         "https://example.com/fake.tar.gz",
					SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					Bin:         []string{"bin/test"},
				},
				"darwin-amd64": {
					ArchiveType: "zip",
					URL:         "https://example.com/fake.zip",
					SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					Bin:         []string{"bin/test"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	t.Run("fetch manifest from URL", func(t *testing.T) {
		manifest, err := FetchManifest(server.URL)
		if err != nil {
			t.Fatalf("failed to fetch manifest: %v", err)
		}

		if manifest.Name != "test-pkg" {
			t.Errorf("expected name 'test-pkg', got '%s'", manifest.Name)
		}

		if manifest.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", manifest.Version)
		}

		if len(manifest.Targets) != 3 {
			t.Errorf("expected 3 targets, got %d", len(manifest.Targets))
		}
	})

	t.Run("get target for current OS", func(t *testing.T) {
		manifest, err := FetchManifest(server.URL)
		if err != nil {
			t.Fatalf("failed to fetch manifest: %v", err)
		}

		target := getTargetForOS(manifest)
		if target == nil {
			t.Fatal("failed to get target for current OS")
		}

		if len(target.Bin) == 0 {
			t.Fatal("target has no binaries")
		}
	})

	t.Run("storage manager creates directories", func(t *testing.T) {
		sm, err := NewStorageManager()
		if err != nil {
			t.Fatalf("failed to create storage manager: %v", err)
		}

		if _, err := os.Stat(sm.BlazeDir); os.IsNotExist(err) {
			t.Errorf("blaze directory not created: %s", sm.BlazeDir)
		}

		if _, err := os.Stat(sm.PackagesDir); os.IsNotExist(err) {
			t.Errorf("packages directory not created: %s", sm.PackagesDir)
		}
	})

	t.Run("registry save and load", func(t *testing.T) {
		sm, err := NewStorageManager()
		if err != nil {
			t.Fatalf("failed to create storage manager: %v", err)
		}

		testRegistry := map[string][]string{
			"test-pkg":  {"1.0.0", "1.1.0"},
			"other-pkg": {"2.0.0"},
		}

		if err := sm.SaveRegistry(testRegistry); err != nil {
			t.Fatalf("failed to save registry: %v", err)
		}

		loaded, err := sm.LoadRegistry()
		if err != nil {
			t.Fatalf("failed to load registry: %v", err)
		}

		if len(loaded) != len(testRegistry) {
			t.Errorf("registry size mismatch: expected %d, got %d", len(testRegistry), len(loaded))
		}

		if len(loaded["test-pkg"]) != 2 {
			t.Errorf("expected 2 versions of test-pkg, got %d", len(loaded["test-pkg"]))
		}
	})

	t.Run("handle list shows installed packages", func(t *testing.T) {
		if err := handleList(); err != nil {
			t.Fatalf("handleList failed: %v", err)
		}
	})

	t.Run("handle remove errors on non-existent package", func(t *testing.T) {
		err := handleRemove("nonexistent-pkg", "1.0.0")
		if err == nil {
			t.Fatal("expected error for non-existent package")
		}
	})

	t.Run("handle use validates format", func(t *testing.T) {
		err := handleUse("invalid-format")
		if err == nil {
			t.Fatal("expected error for invalid package@version format")
		}
	})
}
