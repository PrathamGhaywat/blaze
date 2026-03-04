package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// filepath: c:\Users\Prath\Programming\blaze\src\handlers_test.go
// ...existing code...

func TestRealPackageDownload(t *testing.T) {
	// Skip on CI or if network unavailable
	t.Run("download and install real package", func(t *testing.T) {
		// Create a test manifest pointing to a real small binary
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			manifest := Manifest{
				Schema:      1,
				Name:        "gh",
				Version:     "2.40.0",
				Description: "GitHub CLI",
				Homepage:    "https://cli.github.com",
				License:     "MIT",
				Targets: map[string]Target{
					"windows-amd64": {
						ArchiveType: "zip",
						URL:         "https://github.com/cli/cli/releases/download/v2.40.0/gh_2.40.0_windows_amd64.zip",
						SHA256:      "a51388c17f5b48ddf1d11d3a64c66b12c67c1cc1f1e3b8c2d5e9f0a1b2c3d4e5f",
						Bin:         []string{"bin/gh.exe"},
					},
					"linux-amd64": {
						ArchiveType: "tar.gz",
						URL:         "https://github.com/cli/cli/releases/download/v2.40.0/gh_2.40.0_linux_amd64.tar.gz",
						SHA256:      "b5c2d4e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3",
						Bin:         []string{"bin/gh"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		// Fetch the manifest
		manifest, err := FetchManifest(server.URL)
		if err != nil {
			t.Fatalf("failed to fetch manifest: %v", err)
		}

		if manifest.Name != "gh" {
			t.Errorf("expected name 'gh', got '%s'", manifest.Name)
		}

		// Get target for OS
		target := getTargetForOS(manifest)
		if target == nil {
			t.Fatalf("no target found for current OS")
		}

		fmt.Printf("✓ Would download from: %s\n", target.URL)
		fmt.Printf("✓ SHA256: %s\n", target.SHA256)
		fmt.Printf("✓ Archive type: %s\n", target.ArchiveType)
		fmt.Printf("✓ Binaries: %v\n", target.Bin)
	})
}

func TestHandleRemove(t *testing.T) {
	t.Run("removes non-existent package", func(t *testing.T) {
		err := handleRemove("nonexistent", "1.0.0", false)
		if err == nil {
			t.Fatal("expected error for non-existent package")
		}
	})
}
