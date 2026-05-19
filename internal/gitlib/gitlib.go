package gitlib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Init initializes a git repository at the given path
func Init(path string) error {
	// Check if already a git repo
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil // Already a git repo
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	return nil
}

// Commit stages all changes and commits with the given message
func Commit(path string, message string) error {
	// git add .
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = path
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w, output: %s", err, string(out))
	}

	// git commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = path
	if out, err := commitCmd.CombinedOutput(); err != nil {
		// Check if there's anything to commit
		if string(out) == "nothing to commit, working tree clean\n" {
			return nil
		}
		return fmt.Errorf("git commit failed: %w, output: %s", err, string(out))
	}

	return nil
}

// ShowHistory displays git log
func ShowHistory(path string) error {
	cmd := exec.Command("git", "log", "--oneline", "-20")
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Rollback reverts to a specific commit
func Rollback(path string, commit string) error {
	// git reset --hard <commit>
	cmd := exec.Command("git", "reset", "--hard", commit)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git reset failed: %w", err)
	}

	fmt.Printf("✓ Rolled back to %s\n", commit)
	return nil
}

// IsRepo checks if path is a git repository
func IsRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}