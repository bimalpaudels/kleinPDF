//go:build ignore

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

const (
	baseURL = "https://github.com/bimalpaudels/kleinPDF-ghostscript-binary/releases/download/ghostscript-10.05.1"
)

func main() {
	var binaryName string
	switch runtime.GOARCH {
	case "arm64":
		binaryName = "ghostscript-10.05.1-macos-arm64"
	case "amd64":
		binaryName = "ghostscript-10.05.1-macos-x86_64"
	default:
		fmt.Printf("Unsupported architecture: %s\n", runtime.GOARCH)
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/%s", baseURL, binaryName)
	outputPath := "ghostscript_binary"

	fmt.Printf("Downloading %s for %s...\n", binaryName, runtime.GOARCH)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to download binary: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed to download binary: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Failed to write binary: %v\n", err)
		os.Exit(1)
	}

	// Make executable
	err = os.Chmod(outputPath, 0755)
	if err != nil {
		fmt.Printf("Failed to make binary executable: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully downloaded %s\n", outputPath)
}