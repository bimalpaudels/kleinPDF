package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	GithubReleasesAPI = "https://api.github.com/repos/bimalpaudels/kleinPDF-ghostscript-binary/releases/latest"
	BundledDir        = "./bundled/ghostscript"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {
	fmt.Println("Ghostscript Binary Bundler for macOS")
	fmt.Printf("Detected architecture: %s\n", runtime.GOARCH)
	
	// Ensure we're on macOS
	if runtime.GOOS != "darwin" {
		fmt.Printf("Error: This bundler is designed for macOS only. Current OS: %s\n", runtime.GOOS)
		os.Exit(1)
	}
	
	// Determine the architecture-specific binary name
	var binaryName string
	switch runtime.GOARCH {
	case "amd64":
		binaryName = "ghostscript-10.05.1-macos-x86_64.tar.gz"
	case "arm64":
		binaryName = "ghostscript-10.05.1-macos-arm64.tar.gz"
	default:
		fmt.Printf("Error: Unsupported architecture: %s\n", runtime.GOARCH)
		os.Exit(1)
	}
	
	fmt.Printf("Looking for binary: %s\n", binaryName)
	
	// Get latest release info
	release, err := getLatestRelease()
	if err != nil {
		fmt.Printf("Error getting latest release: %v\n", err)
		os.Exit(1)
	}
	
	// Find the correct asset
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	
	if downloadURL == "" {
		fmt.Printf("Error: Could not find %s in release assets\n", binaryName)
		os.Exit(1)
	}
	
	// Create bundled directory
	err = os.MkdirAll(BundledDir, 0755)
	if err != nil {
		fmt.Printf("Error creating bundled directory: %v\n", err)
		os.Exit(1)
	}
	
	// Download and extract
	fmt.Printf("Downloading %s...\n", binaryName)
	err = downloadAndExtract(downloadURL, BundledDir)
	if err != nil {
		fmt.Printf("Error downloading and extracting: %v\n", err)
		os.Exit(1)
	}
	
	// Make binary executable
	gsPath := filepath.Join(BundledDir, "gs")
	err = os.Chmod(gsPath, 0755)
	if err != nil {
		fmt.Printf("Warning: Could not make gs executable: %v\n", err)
	}
	
	fmt.Println("✅ Ghostscript binary bundled successfully!")
	fmt.Printf("Binary location: %s\n", gsPath)
}

func getLatestRelease() (*GitHubRelease, error) {
	resp, err := http.Get(GithubReleasesAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	// Simple JSON parsing for tag_name and assets
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	release := &GitHubRelease{}
	
	// Parse JSON manually (simple parsing for this specific structure)
	bodyStr := string(body)
	
	// Extract tag_name
	if tagStart := strings.Index(bodyStr, `"tag_name":"`); tagStart != -1 {
		tagStart += len(`"tag_name":"`)
		if tagEnd := strings.Index(bodyStr[tagStart:], `"`); tagEnd != -1 {
			release.TagName = bodyStr[tagStart : tagStart+tagEnd]
		}
	}
	
	// Extract assets
	if assetsStart := strings.Index(bodyStr, `"assets":[`); assetsStart != -1 {
		assetsSection := bodyStr[assetsStart:]
		
		// Find all asset objects
		assetStart := 0
		for {
			nameIndex := strings.Index(assetsSection[assetStart:], `"name":"`)
			if nameIndex == -1 {
				break
			}
			nameIndex += assetStart + len(`"name":"`)
			
			nameEnd := strings.Index(assetsSection[nameIndex:], `"`)
			if nameEnd == -1 {
				break
			}
			name := assetsSection[nameIndex : nameIndex+nameEnd]
			
			// Find corresponding download URL
			urlStart := strings.Index(assetsSection[nameIndex:], `"browser_download_url":"`)
			if urlStart == -1 {
				break
			}
			urlStart += nameIndex + len(`"browser_download_url":"`)
			
			urlEnd := strings.Index(assetsSection[urlStart:], `"`)
			if urlEnd == -1 {
				break
			}
			url := assetsSection[urlStart : urlStart+urlEnd]
			
			release.Assets = append(release.Assets, struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{
				Name:               name,
				BrowserDownloadURL: url,
			})
			
			assetStart = urlStart + urlEnd
		}
	}
	
	return release, nil
}

func downloadAndExtract(url, destDir string) error {
	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	// Create gzip reader
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzReader)
	
	// Extract files preserving the original directory structure
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %v", err)
		}
		
		// Skip directories (they will be created automatically)
		if header.Typeflag == tar.TypeDir {
			continue
		}
		
		// Preserve the original path structure from the tar file
		// The GitHub release now has the correct structure like:
		// ghostscript-x86_64/gs
		// ghostscript-x86_64/gs-wrapper.sh
		// ghostscript-x86_64/lib/*.dylib
		// ghostscript-x86_64/share/ghostscript/10.05.1/Resource/Init/gs_init.ps
		// etc.
		
		// Remove the top-level directory name (e.g., "ghostscript-x86_64/")
		originalPath := header.Name
		pathParts := strings.Split(originalPath, "/")
		if len(pathParts) <= 1 {
			continue // Skip files at root level of tar
		}
		
		// Join the path parts except the first one to get the relative path
		relativePath := strings.Join(pathParts[1:], "/")
		destPath := filepath.Join(destDir, relativePath)
		
		// Create the directory structure if it doesn't exist
		destDirPath := filepath.Dir(destPath)
		err = os.MkdirAll(destDirPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", destDirPath, err)
		}
		
		// Create the file
		outFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", destPath, err)
		}
		
		// Copy content
		_, err = io.Copy(outFile, tarReader)
		outFile.Close()
		
		if err != nil {
			return fmt.Errorf("failed to write file %s: %v", destPath, err)
		}
		
		// Set file permissions
		err = os.Chmod(destPath, os.FileMode(header.Mode))
		if err != nil {
			fmt.Printf("Warning: Could not set permissions for %s: %v\n", destPath, err)
		}
		
		fmt.Printf("Extracted: %s\n", relativePath)
	}
	
	return nil
}