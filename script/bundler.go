package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	bundledGhostscriptDir = "./bundled/ghostscript"
)

func main() {
	fmt.Println("Ghostscript bundler (Homebrew-based)")

	if runtime.GOOS != "darwin" {
		fmt.Printf("Error: This bundler only supports macOS (darwin). Detected: %s\n", runtime.GOOS)
		os.Exit(1)
	}

	brewPath, err := exec.LookPath("brew")
	if err != nil {
		fmt.Println("Error: Homebrew is required but was not found on PATH.")
		fmt.Println("Install Homebrew from: https://brew.sh and re-run this bundler.")
		os.Exit(1)
	}
	fmt.Printf("Using Homebrew at: %s\n", brewPath)

	// Ensure ghostscript formula is installed
	if !isGhostscriptInstalled(brewPath) {
		fmt.Println("Ghostscript not found in Homebrew. Installing ghostscript via Homebrew...")
		if err := runCommand(brewPath, "install", "ghostscript"); err != nil {
			fmt.Printf("Error installing ghostscript: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Ghostscript is already installed in Homebrew.")
	}

	// Determine the prefix for the ghostscript formula (typically /opt/homebrew/opt/ghostscript)
	prefix, err := getBrewPrefixForFormula(brewPath, "ghostscript")
	if err != nil {
		fmt.Printf("Error resolving Homebrew prefix for ghostscript: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Resolved ghostscript prefix: %s\n", prefix)

	// Resolve real path (follow the opt symlink into Cellar/version)
	resolvedPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		fmt.Printf("Error resolving symlink for %s: %v\n", prefix, err)
		os.Exit(1)
	}
	fmt.Printf("Resolved ghostscript source: %s\n", resolvedPrefix)

	// Copy required directories into bundled/ghostscript
	if err := os.MkdirAll(bundledGhostscriptDir, 0o755); err != nil {
		fmt.Printf("Error creating destination directory %s: %v\n", bundledGhostscriptDir, err)
		os.Exit(1)
	}

	// Always copy bin (contains gs)
	if err := copyDir(filepath.Join(resolvedPrefix, "bin"), filepath.Join(bundledGhostscriptDir, "bin")); err != nil {
		fmt.Printf("Error copying bin/: %v\n", err)
		os.Exit(1)
	}

	// Copy lib (dynamic libraries used by gs)
	if err := copyDir(filepath.Join(resolvedPrefix, "lib"), filepath.Join(bundledGhostscriptDir, "lib")); err != nil {
		fmt.Printf("Error copying lib/: %v\n", err)
		os.Exit(1)
	}

	// Copy share/ghostscript (resources)
	if err := copyDir(filepath.Join(resolvedPrefix, "share", "ghostscript"), filepath.Join(bundledGhostscriptDir, "share", "ghostscript")); err != nil {
		fmt.Printf("Error copying share/ghostscript/: %v\n", err)
		os.Exit(1)
	}

	// Ensure gs is executable
	gsPath := filepath.Join(bundledGhostscriptDir, "bin", "gs")
	if err := os.Chmod(gsPath, 0o755); err != nil {
		fmt.Printf("Warning: failed to mark gs executable: %v\n", err)
	}

	fmt.Println("✅ Ghostscript bundled successfully from Homebrew!")
	fmt.Printf("Binary: %s\n", gsPath)
	fmt.Printf("Libraries: %s\n", filepath.Join(bundledGhostscriptDir, "lib"))
	fmt.Printf("Resources: %s\n", filepath.Join(bundledGhostscriptDir, "share", "ghostscript"))
}

func isGhostscriptInstalled(brew string) bool {
	// `brew ls --versions ghostscript` returns non-empty output if installed
	cmd := exec.Command(brew, "ls", "--versions", "ghostscript")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func getBrewPrefixForFormula(brew, formula string) (string, error) {
	cmd := exec.Command(brew, "--prefix", formula)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// copyDir copies a directory recursively from src to dst.
// If the source path is a symlink to a directory, it resolves the symlink first.
func copyDir(src, dst string) error {
	// If src doesn't exist, skip silently (formula layouts may vary)
	info, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Follow symlink for directories/files at the root we copy from
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(src)
		if err != nil {
			return err
		}
		src = resolved
		info, err = os.Stat(src)
		if err != nil {
			return err
		}
		_ = info
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Compute destination path
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, rel)

		// Resolve symlinks to copy actual file content
		fileInfo, err := d.Info()
		if err != nil {
			return err
		}

		// If symlink, resolve to target
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			// Replace path and fileInfo with resolved target
			path = resolved
			fileInfo, err = os.Stat(path)
			if err != nil {
				return err
			}
		}

		if fileInfo.IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
			return nil
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		return copyFile(path, destPath, fileInfo.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Preserve a reasonable mode; ensure readability
	if mode == 0 {
		mode = 0o644
	}
	if err := os.Chmod(dst, mode); err != nil {
		return err
	}
	return nil
}
