package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	fmt.Println("Simple Ghostscript bundler")

	if runtime.GOOS != "darwin" {
		fmt.Printf("Error: This bundler only supports macOS. Detected: %s\n", runtime.GOOS)
		os.Exit(1)
	}

	// Check if ghostscript is installed via brew
	cmd := exec.Command("brew", "--prefix", "ghostscript")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error: Ghostscript not found. Install with: brew install ghostscript")
		os.Exit(1)
	}

	gsPrefix := strings.TrimSpace(string(output))
	fmt.Printf("Found Ghostscript at: %s\n", gsPrefix)

	// Clean and create bundled directory
	bundleDir := "./bundled/ghostscript"
	os.RemoveAll(bundleDir)
	os.MkdirAll(bundleDir, 0755)

	// Copy only what we need - much simpler!
	copyPaths := map[string]string{
		filepath.Join(gsPrefix, "bin/gs"):                          filepath.Join(bundleDir, "bin/gs"),
		filepath.Join(gsPrefix, "share/ghostscript"):               filepath.Join(bundleDir, "share/ghostscript"),
		filepath.Join(gsPrefix, "lib"):                             filepath.Join(bundleDir, "lib"),
	}

	for src, dest := range copyPaths {
		if err := copyRecursive(src, dest); err != nil {
			fmt.Printf("Error copying %s: %v\n", src, err)
			os.Exit(1)
		}
		fmt.Printf("✓ Copied %s\n", filepath.Base(src))
	}

	// Ensure gs is executable
	os.Chmod(filepath.Join(bundleDir, "bin/gs"), 0755)

	fmt.Println("✅ Ghostscript bundled successfully!")
	fmt.Printf("Bundle size: %s\n", getDirSize(bundleDir))
}

func copyRecursive(src, dest string) error {
	// Use Lstat to detect symlinks without following them
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// If it's a symlink, resolve it once and copy the target
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(src)
		if err != nil {
			return err
		}
		// Copy the resolved target
		return copyRecursive(resolved, dest)
	}

	if info.IsDir() {
		return copyDir(src, dest)
	}
	return copyFile(src, dest, info.Mode())
}

func copyDir(src, dest string) error {
	os.MkdirAll(dest, 0755)
	
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		
		// Skip if it's a symlink to avoid loops - we'll resolve at the top level
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		
		if err := copyRecursive(srcPath, destPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string, mode os.FileMode) error {
	os.MkdirAll(filepath.Dir(dest), 0755)
	
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return os.Chmod(dest, mode)
}

func getDirSize(path string) string {
	var size int64
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}