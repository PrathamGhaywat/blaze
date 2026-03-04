package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArchive extracts a zip or tar.gz file to a destination
func ExtractArchive(archivePath, destPath, archiveType string) error {
	switch archiveType {
	case "zip":
		return extractZip(archivePath, destPath)
	case "tar.gz":
		return extractTarGz(archivePath, destPath)
	case "tar":
		return extractTar(archivePath, destPath)
	default:
		return fmt.Errorf("unsupported archive type: %s", archiveType)
	}
}

// extractZip extracts a zip file
func extractZip(zipPath, destPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(destPath, file.Name)

		if strings.HasSuffix(file.Name, "/") {
			os.MkdirAll(path, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			src.Close()
			return err
		}

		io.Copy(dst, src)
		dst.Close()
		src.Close()
	}

	return nil
}

// extractTarGz extracts a tar.gz file
func extractTarGz(tarGzPath, destPath string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	return extractTarReader(gzReader, destPath)
}

// extractTar extracts a tar file
func extractTar(tarPath, destPath string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar: %w", err)
	}
	defer file.Close()

	return extractTarReader(file, destPath)
}

// extractTarReader extracts from a tar reader
func extractTarReader(reader io.Reader, destPath string) error {
	tarReader := tar.NewReader(reader)

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(destPath, header.Name)

		if header.Typeflag == tar.TypeDir {
			os.MkdirAll(path, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return err
		}

		io.Copy(file, tarReader)
		file.Close()
	}

	return nil
}
