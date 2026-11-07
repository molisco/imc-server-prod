package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	defaultDest = "docs/brain"
)

func main() {
	// Get user home directory to resolve AppData/Gemini folder
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Error getting user home directory: %v\n", err)
		os.Exit(1)
	}

	brainRootDir := filepath.Join(homeDir, ".gemini", "antigravity", "brain")
	fmt.Printf("🔍 Brain root directory: %s\n", brainRootDir)

	// List all subdirectories in brainRootDir
	entries, err := os.ReadDir(brainRootDir)
	if err != nil {
		fmt.Printf("❌ Error reading brain root directory: %v\n", err)
		os.Exit(1)
	}

	filesToCopy := map[string]string{
		"implementation_plan.md":                 "implementation_plan.md",
		"walkthrough.md":                         "walkthrough.md",
		"task.md":                                "task.md",
		".system_generated/logs/transcript.jsonl": "transcript.jsonl",
	}

	totalCopied := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "tempmediaStorage" {
			continue
		}

		convID := entry.Name()
		convSrcDir := filepath.Join(brainRootDir, convID)
		convDestDir := filepath.Join(defaultDest, convID)

		// Check if it has any of the files first
		hasAny := false
		for srcRel := range filesToCopy {
			srcPath := filepath.Join(convSrcDir, srcRel)
			if _, err := os.Stat(srcPath); err == nil {
				hasAny = true
				break
			}
		}

		if !hasAny {
			continue
		}

		// Create directory for this conversation
		if err := os.MkdirAll(convDestDir, 0755); err != nil {
			fmt.Printf("❌ Error creating directory for %s: %v\n", convID, err)
			continue
		}

		fmt.Printf("📦 Backing up conversation: %s\n", convID)
		copiedCount := 0
		for srcRel, destName := range filesToCopy {
			srcPath := filepath.Join(convSrcDir, srcRel)
			destPath := filepath.Join(convDestDir, destName)

			if err := copyFile(srcPath, destPath); err != nil {
				if !os.IsNotExist(err) {
					fmt.Printf("  ❌ Error copying %s: %v\n", srcRel, err)
				}
			} else {
				fmt.Printf("  ✓ Copied: %s\n", srcRel)
				copiedCount++
				totalCopied++
			}
		}
	}

	fmt.Printf("🎉 Backup completed. Copied %d files across active conversations.\n", totalCopied)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
