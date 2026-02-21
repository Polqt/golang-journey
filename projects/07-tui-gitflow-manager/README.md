# Project 07 — TUI GitFlow Manager

> **Difficulty**: Intermediate-Senior · **Domain**: TUI, CLI, Git Internals, Developer Tooling
> **Real-world analog**: Lazygit, Gitui, GitHub CLI, Tower

---

## Why This Project Exists

Lazygit and Gitui are beloved because they make Git's mental model visual and interactive.
This project builds a **Git workflow manager** with a custom TUI using **Bubble Tea** — the
modern Go TUI framework. The killer feature: it implements the **Gitflow branching model**
as first-class workflow with branch type awareness, PR templates, and stash management.

---

## Folder Structure

```
07-tui-gitflow-manager/
├── go.mod
├── main.go
├── tui/
│   ├── app.go                   # Root Bubble Tea model
│   ├── dashboard.go             # Main screen: branches, status, log
│   ├── branch_panel.go          # Branch list with type coloring
│   ├── diff_viewer.go           # Side-by-side / unified diff pager
│   ├── commit_form.go           # Staged files + commit message form
│   ├── stash_panel.go           # Stash list with preview
│   ├── pr_template.go           # EDITOR-like PR title/body form
│   └── styles.go                # Lipgloss style definitions
├── git/
│   ├── repo.go                  # Repository wrapper (shells to git CLI)
│   ├── branch.go                # Branch type detection + operations
│   ├── log.go                   # Git log parsing
│   ├── diff.go                  # Diff parsing + syntax-aware coloring
│   ├── stash.go                 # Stash operations
│   └── status.go                # Working tree status
├── gitflow/
│   ├── workflow.go              # Gitflow rules: feature/release/hotfix
│   ├── naming.go                # Branch name validation
│   └── config.go                # .gitflow config reader
└── config/
    └── config.go                # App-level config (~/.gitflowrc)
```

---

## Technology Stack

```go
// go.mod dependencies
require (
    github.com/charmbracelet/bubbletea  v0.27.0  // TUI framework
    github.com/charmbracelet/lipgloss   v0.12.0  // styling
    github.com/charmbracelet/bubbles    v0.19.0  // list, textinput, viewport
)
```

---

## Implementation Guide

### Phase 1 — Git Wrapper (Week 1)

Wrap the `git` CLI rather than implementing git internals. This is the pragmatic choice
used by Lazygit and GitHub CLI.

```go
type Repo struct {
    Root string
}

func (r *Repo) Status() ([]FileStatus, error)
func (r *Repo) Branches() ([]Branch, error)
func (r *Repo) Log(limit int) ([]Commit, error)
func (r *Repo) Diff(ref string) (string, error)
func (r *Repo) Stage(path string) error
func (r *Repo) Unstage(path string) error
func (r *Repo) Commit(msg string, amend bool) error
func (r *Repo) Stash(msg string) error
func (r *Repo) StashPop(index int) error
func (r *Repo) Checkout(branch string) error
func (r *Repo) CreateBranch(name, base string) error
func (r *Repo) MergeBranch(name string, noFF bool) error
```

**Implementation**: run `exec.Command("git", ...)`, parse stdout. Use `bufio.Scanner` for
line-by-line parsing.

Pro tip: `git status --porcelain=v2` gives machine-readable output. Use `-z` flag for
NUL-delimited output (handles spaces in filenames).

---

### Phase 2 — Bubble Tea Architecture (Week 1-2)

Bubble Tea uses the **Elm Architecture**: `Model`, `Update(msg)`, `View()`.

```go
type App struct {
    repo     *git.Repo
    panel    ActivePanel  // BRANCHES | DIFF | COMMIT | STASH
    branches list.Model
    diff     viewport.Model
    commit   commitForm
    stash    list.Model
    status   statusBar
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return a.handleKey(msg)
    case gitRefreshMsg:
        return a.handleGitRefresh(msg)
    }
    return a, nil
}
```

**Key panels**:
- Left: Branch list (colored by type: feature=blue, release=green, hotfix=red, main=gold)
- Center: Git log for selected branch
- Right: Status / diff preview
- Bottom: Status bar with current branch, ahead/behind counts

---

### Phase 3 — Gitflow Workflow (Week 2)

Implement first-class Gitflow support:

```
KEY BINDINGS:
  n f  → new feature branch (prompts for name: feature/your-name)
  n r  → new release branch (prompts for version: release/1.2.0)
  n h  → new hotfix branch  (prompts for version: hotfix/1.1.1)
  F    → finish feature → merge to develop, delete branch
  R    → finish release → merge to main + develop, tag version
  H    → finish hotfix  → merge to main + develop, tag version
  p    → push current branch
  P    → pull with rebase
```

Branch naming validation:
- Features: `feature/*`, `feat/*`
- Releases: `release/[semver]`
- Hotfixes: `hotfix/[semver]`

---

### Phase 4 — Diff Viewer with Syntax Highlighting (Week 3)

Parse `git diff` output into structured hunks and apply ANSI colors:
- Added lines: green
- Removed lines: red
- Hunk headers: cyan
- File names: bold

Use `bubbles/viewport` for scrollable diff display.

**Bonus**: implement `--word-diff` mode: highlight changed words within a line.

---

### Phase 5 — PR Template Form (Week 3)

When finishing a feature/release branch, show a form for PR creation:

```
╭─── New Pull Request ──────────────────────────────────╮
│ Title: feat: add payment processing                    │
│                                                        │
│ Description:                                           │
│ ## What                                                │
│ Added Stripe integration for subscription payments     │
│                                                        │
│ ## Testing                                             │
│ - [ ] Unit tests passing                               │
│ - [ ] Manual test in staging                           │
│                                                        │
│ Reviewers: @alice @bob                                 │
╰────────────────────────────────────────[ Submit ] [ Cancel ]─╯
```

Copy the PR body to clipboard using `golang.design/x/clipboard` or write to a temp file
and open with `$EDITOR`.

---

## Acceptance Criteria

- [ ] App renders correctly at terminal sizes from 80x24 to 200x50
- [ ] All git operations run in background goroutines (no UI freezing)
- [ ] Gitflow branch finishing validates that develop/main are up to date
- [ ] Stash list shows diffs on hover

---

## Stretch Goals

- Add **rebase TUI**: interactive visualization of rebase steps
- Implement **worktree management**: open/switch between git worktrees
- Add **GitHub/GitLab API integration** for real PR submission
