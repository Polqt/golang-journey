// Package gitflow implements the gitflow branching model automation.
package gitflow

import (
	"fmt"
	"strings"

	"github.com/Polqt/gitflow/git"
)

// ─────────────────────────────────────────────────────────────
// Branch naming conventions
// ─────────────────────────────────────────────────────────────

const (
	prefixFeature  = "feature/"
	prefixRelease  = "release/"
	prefixHotfix   = "hotfix/"
	prefixBugfix   = "bugfix/"
	branchMain     = "main"
	branchDevelop  = "develop"
)

// BranchKind classifies a branch by its naming convention.
type BranchKind int

const (
	KindFeature BranchKind = iota
	KindRelease
	KindHotfix
	KindBugfix
	KindMain
	KindDevelop
	KindOther
)

// Classify returns the BranchKind for the given branch name.
func Classify(name string) BranchKind {
	switch {
	case strings.HasPrefix(name, prefixFeature):
		return KindFeature
	case strings.HasPrefix(name, prefixRelease):
		return KindRelease
	case strings.HasPrefix(name, prefixHotfix):
		return KindHotfix
	case strings.HasPrefix(name, prefixBugfix):
		return KindBugfix
	case name == branchMain || name == "master":
		return KindMain
	case name == branchDevelop:
		return KindDevelop
	default:
		return KindOther
	}
}

// ─────────────────────────────────────────────────────────────
// Workflow
// ─────────────────────────────────────────────────────────────

// Workflow orchestrates gitflow operations on a Repo.
type Workflow struct {
	repo *git.Repo
}

// New creates a Workflow backed by the given Repo.
func New(repo *git.Repo) *Workflow {
	return &Workflow{repo: repo}
}

// CreateBranch creates a branch following gitflow conventions:
//   - feature/* → branches from develop
//   - release/* → branches from develop
//   - hotfix/*  → branches from main
//   - bugfix/*  → branches from develop
func (w *Workflow) CreateBranch(name string) error {
	kind := Classify(name)
	var base string
	switch kind {
	case KindFeature, KindRelease, KindBugfix:
		base = branchDevelop
	case KindHotfix:
		base = branchMain
	default:
		// Unknown prefix — branch from current HEAD.
		return w.repo.CreateBranch(name)
	}

	// Check that base exists.
	branches, err := w.repo.Branches()
	if err != nil {
		return err
	}
	baseExists := false
	for _, b := range branches {
		if b.Name == base {
			baseExists = true
			break
		}
	}
	if !baseExists {
		return fmt.Errorf("base branch %q not found — run 'git checkout -b %s' first", base, base)
	}

	// Checkout base, then create new branch.
	if err := w.repo.Checkout(base); err != nil {
		return fmt.Errorf("checkout %s: %w", base, err)
	}
	return w.repo.CreateBranch(name)
}

// Merge merges src into dst following gitflow rules.
//   - feature/* → merge into develop (squash-merge optional)
//   - release/* → merge into both develop and main, then tag
//   - hotfix/*  → merge into both main and develop, then tag
func (w *Workflow) Merge(src, dst string) error {
	kind := Classify(src)
	switch kind {
	case KindFeature, KindBugfix:
		return w.repo.Merge(src, branchDevelop)

	case KindRelease:
		version := strings.TrimPrefix(src, prefixRelease)
		if err := w.repo.Merge(src, branchMain); err != nil {
			return fmt.Errorf("merge to main: %w", err)
		}
		if err := w.repo.Tag("v"+version, "Release "+version); err != nil {
			return fmt.Errorf("tag: %w", err)
		}
		if err := w.repo.Merge(src, branchDevelop); err != nil {
			return fmt.Errorf("merge to develop: %w", err)
		}
		return nil

	case KindHotfix:
		version := strings.TrimPrefix(src, prefixHotfix)
		if err := w.repo.Merge(src, branchMain); err != nil {
			return fmt.Errorf("merge to main: %w", err)
		}
		if err := w.repo.Tag("v"+version+"-hotfix", "Hotfix "+version); err != nil {
			return fmt.Errorf("tag: %w", err)
		}
		if err := w.repo.Merge(src, branchDevelop); err != nil {
			return fmt.Errorf("merge to develop: %w", err)
		}
		return nil

	default:
		// Manual merge.
		return w.repo.Merge(src, dst)
	}
}

// FinishRelease finalises a release branch: merges to main+develop, tags, deletes branch.
func (w *Workflow) FinishRelease(version string) error {
	branch := prefixRelease + version
	if err := w.Merge(branch, ""); err != nil {
		return err
	}
	return w.repo.DeleteBranch(branch, false)
}

// FinishHotfix finalises a hotfix branch.
func (w *Workflow) FinishHotfix(version string) error {
	branch := prefixHotfix + version
	if err := w.Merge(branch, ""); err != nil {
		return err
	}
	return w.repo.DeleteBranch(branch, false)
}

// FinishFeature merges feature branch to develop and deletes it.
func (w *Workflow) FinishFeature(name string) error {
	branch := prefixFeature + name
	if err := w.repo.Merge(branch, branchDevelop); err != nil {
		return err
	}
	return w.repo.DeleteBranch(branch, false)
}
