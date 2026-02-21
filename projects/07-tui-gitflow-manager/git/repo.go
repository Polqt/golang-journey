// Package git wraps git CLI commands for use by the TUI.
package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ─────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────

// Branch describes one local or remote branch.
type Branch struct {
	Name       string
	Current    bool
	Remote     string
	LastCommit string // short hash of HEAD on this branch
}

// LogEntry is one line from git log.
type LogEntry struct {
	Hash    string
	Subject string
	Author  string
	Date    string
}

// StatusFile represents a single file in the working tree status.
type StatusFile struct {
	Path   string
	Status string // M, A, D, ?, etc.
}

// ─────────────────────────────────────────────────────────────
// Repo
// ─────────────────────────────────────────────────────────────

// Repo wraps a local git repository.
type Repo struct {
	root string
}

// Open validates that path is inside a git repository and returns a Repo.
func Open(path string) (*Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	out, err := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("%q is not a git repository", abs)
	}
	root := strings.TrimSpace(string(out))
	return &Repo{root: root}, nil
}

// Path returns the repository root directory.
func (r *Repo) Path() string { return r.root }

// git runs a git command and returns combined output.
func (r *Repo) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.root}, args...)...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// CurrentBranch returns the name of the checked-out branch.
func (r *Repo) CurrentBranch() (string, error) {
	out, err := r.git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("current branch: %w", err)
	}
	return out, nil
}

// Branches returns all local branches.
func (r *Repo) Branches() ([]Branch, error) {
	// format: <refname:short>|<HEAD>|<objectname:short>|<upstream:short>
	out, err := r.git("branch", "--format=%(refname:short)|%(HEAD)|%(objectname:short)|%(upstream:short)")
	if err != nil {
		return nil, fmt.Errorf("branches: %w", err)
	}
	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		branches = append(branches, Branch{
			Name:       parts[0],
			Current:    parts[1] == "*",
			LastCommit: parts[2],
			Remote:     parts[3],
		})
	}
	return branches, nil
}

// Log returns up to n log entries for the given branch.
func (r *Repo) Log(branch string, n int) ([]LogEntry, error) {
	out, err := r.git("log", fmt.Sprintf("-n%d", n), "--format=%H|%s|%an|%ar", branch)
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}
	var entries []LogEntry
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, LogEntry{
			Hash:    parts[0],
			Subject: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return entries, nil
}

// CreateBranch creates a new branch from HEAD.
func (r *Repo) CreateBranch(name string) error {
	_, err := r.git("checkout", "-b", name)
	return err
}

// Checkout switches to the given branch.
func (r *Repo) Checkout(branch string) error {
	_, err := r.git("checkout", branch)
	return err
}

// Merge merges src into dst (checks out dst first).
func (r *Repo) Merge(src, dst string) error {
	if _, err := r.git("checkout", dst); err != nil {
		return fmt.Errorf("checkout %s: %w", dst, err)
	}
	if _, err := r.git("merge", "--no-ff", src); err != nil {
		return fmt.Errorf("merge %s → %s: %w", src, dst, err)
	}
	return nil
}

// DeleteBranch deletes the named local branch.
func (r *Repo) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.git("branch", flag, name)
	return err
}

// Tag creates an annotated tag.
func (r *Repo) Tag(name, message string) error {
	_, err := r.git("tag", "-a", name, "-m", message)
	return err
}

// Status returns a list of modified files.
func (r *Repo) Status() ([]StatusFile, error) {
	out, err := r.git("status", "--porcelain")
	if err != nil {
		return nil, err
	}
	var files []StatusFile
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		files = append(files, StatusFile{
			Status: strings.TrimSpace(line[:2]),
			Path:   line[3:],
		})
	}
	return files, nil
}
