package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

// FetchManifest downloads and parses a manifest from a URL
func FetchManifest(url string) (*Manifest, error) {
    // Validate URL scheme
    if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
        return nil, fmt.Errorf("only http and https URLs are supported, got: %s", url)
    }

    // Fetch the manifest
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch manifest: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
    }

    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read manifest: %w", err)
    }

    // Parse JSON
    var manifest Manifest
    if err := json.Unmarshal(body, &manifest); err != nil {
        return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
    }

    // Validate manifest
    if err := validateManifest(&manifest); err != nil {
        return nil, err
    }

    return &manifest, nil
}

// validateManifest checks required fields and structure
func validateManifest(m *Manifest) error {
    if m.Schema != 1 {
        return fmt.Errorf("unsupported schema version: %d", m.Schema)
    }
    if m.Name == "" || m.Version == "" {
        return fmt.Errorf("manifest missing required fields: name and version")
    }
    if len(m.Targets) == 0 {
        return fmt.Errorf("manifest has no targets defined")
    }

    for targetName, target := range m.Targets {
        if target.URL == "" || target.SHA256 == "" {
            return fmt.Errorf("target %s missing required fields: url or sha256", targetName)
        }
        if len(target.Bin) == 0 {
            return fmt.Errorf("target %s has no binaries defined", targetName)
        }
    }

    return nil
}