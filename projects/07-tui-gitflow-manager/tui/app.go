// Package tui implements the Bubble Tea application model for gitflow-manager.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Polqt/gitflow/git"
	"github.com/Polqt/gitflow/gitflow"
)

// ─────────────────────────────────────────────────────────────
// Styles
// ─────────────────────────────────────────────────────────────

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	normalStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	borderStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
)

// ─────────────────────────────────────────────────────────────
// View enum
// ─────────────────────────────────────────────────────────────

type viewKind int

const (
	viewDashboard viewKind = iota
	viewBranchList
	viewCreateBranch
	viewMerge
	viewLog
	viewConflictResolve
)

// ─────────────────────────────────────────────────────────────
// Messages
// ─────────────────────────────────────────────────────────────

type branchesLoadedMsg struct{ branches []git.Branch }
type logLoadedMsg struct{ entries []git.LogEntry }
type errorMsg struct{ err error }
type successMsg struct{ msg string }
type operationDoneMsg struct{}

// ─────────────────────────────────────────────────────────────
// App model
// ─────────────────────────────────────────────────────────────

// App is the root Bubble Tea model.
type App struct {
	repo     *git.Repo
	flow     *gitflow.Workflow
	view     viewKind
	width    int
	height   int

	// UI components
	branchList  list.Model
	textInput   textinput.Model
	spinner     spinner.Model
	loading     bool

	// State
	branches    []git.Branch
	currentBranch string
	logEntries  []git.LogEntry
	status      string
	statusErr   bool
}

// New creates an App for the git repository at repoPath.
func New(repoPath string) (*App, error) {
	repo, err := git.Open(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}
	flow := gitflow.New(repo)

	ti := textinput.New()
	ti.Placeholder = "feature/my-feature"
	ti.CharLimit = 100

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	del := list.NewDefaultDelegate()
	del.Styles.SelectedTitle = selectedStyle
	del.Styles.NormalTitle = normalStyle

	l := list.New(nil, del, 0, 0)
	l.Title = "Branches"
	l.SetShowStatusBar(false)

	return &App{
		repo:       repo,
		flow:       flow,
		view:       viewDashboard,
		branchList: l,
		textInput:  ti,
		spinner:    sp,
	}, nil
}

// Init loads initial data.
func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadBranches(), spinner.Tick)
}

// Update handles messages.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.branchList.SetSize(msg.Width-4, msg.Height-8)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if a.view == viewDashboard {
				return a, tea.Quit
			}
			a.view = viewDashboard
			return a, nil
		case "n":
			if a.view == viewDashboard || a.view == viewBranchList {
				a.view = viewCreateBranch
				a.textInput.SetValue("")
				a.textInput.Focus()
				return a, textinput.Blink
			}
		case "m":
			if a.view == viewBranchList {
				a.view = viewMerge
				return a, nil
			}
		case "l":
			if a.view == viewDashboard {
				a.view = viewLog
				return a, a.loadLog()
			}
		case "b":
			a.view = viewBranchList
			return a, nil
		case "enter":
			return a.handleEnter()
		case "esc":
			a.view = viewDashboard
			return a, nil
		}

	case branchesLoadedMsg:
		a.loading = false
		a.branches = msg.branches
		items := make([]list.Item, len(msg.branches))
		for i, b := range msg.branches {
			items[i] = branchItem(b)
		}
		a.branchList.SetItems(items)
		if cur, err := a.repo.CurrentBranch(); err == nil {
			a.currentBranch = cur
		}

	case logLoadedMsg:
		a.loading = false
		a.logEntries = msg.entries

	case successMsg:
		a.status = msg.msg
		a.statusErr = false
		return a, a.loadBranches()

	case errorMsg:
		a.status = msg.err.Error()
		a.statusErr = true
		a.loading = false

	case spinner.TickMsg:
		sp, cmd := a.spinner.Update(msg)
		a.spinner = sp
		cmds = append(cmds, cmd)
	}

	// Delegate to focused sub-component.
	switch a.view {
	case viewBranchList:
		var cmd tea.Cmd
		a.branchList, cmd = a.branchList.Update(msg)
		cmds = append(cmds, cmd)
	case viewCreateBranch, viewMerge:
		var cmd tea.Cmd
		a.textInput, cmd = a.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View renders the UI.
func (a App) View() string {
	switch a.view {
	case viewDashboard:
		return a.dashboardView()
	case viewBranchList:
		return a.branchListView()
	case viewCreateBranch:
		return a.createBranchView()
	case viewMerge:
		return a.mergeView()
	case viewLog:
		return a.logView()
	default:
		return "unknown view"
	}
}

// ─────────────────────────────────────────────────────────────
// Views
// ─────────────────────────────────────────────────────────────

func (a App) dashboardView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("⎇  GitFlow Manager") + "\n\n")
	b.WriteString(dimStyle.Render("Repo: "+a.repo.Path()) + "\n")
	b.WriteString(dimStyle.Render("Branch: "+a.currentBranch) + "\n\n")

	menu := []string{
		"[b]  Browse branches",
		"[n]  New feature/release/hotfix branch",
		"[l]  View commit log",
		"[q]  Quit",
	}
	for _, item := range menu {
		b.WriteString(normalStyle.Render("  "+item) + "\n")
	}

	if a.status != "" {
		b.WriteString("\n")
		if a.statusErr {
			b.WriteString(errorStyle.Render("✗ "+a.status) + "\n")
		} else {
			b.WriteString(successStyle.Render("✓ "+a.status) + "\n")
		}
	}
	return borderStyle.Render(b.String())
}

func (a App) branchListView() string {
	return borderStyle.Render(
		titleStyle.Render("Branches") + "\n" +
			dimStyle.Render("[n] new  [m] merge  [q] back") + "\n\n" +
			a.branchList.View())
}

func (a App) createBranchView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Create Branch") + "\n\n")
	b.WriteString(dimStyle.Render("Branch name (e.g. feature/login, release/1.2.0, hotfix/crash):") + "\n")
	b.WriteString(a.textInput.View() + "\n\n")
	b.WriteString(dimStyle.Render("[enter] create  [esc] cancel"))
	return borderStyle.Render(b.String())
}

func (a App) mergeView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Merge Branch") + "\n\n")
	b.WriteString(dimStyle.Render("Source branch to merge into current ("+a.currentBranch+"):") + "\n")
	b.WriteString(a.textInput.View() + "\n\n")
	b.WriteString(dimStyle.Render("[enter] merge  [esc] cancel"))
	return borderStyle.Render(b.String())
}

func (a App) logView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Commit Log — "+a.currentBranch) + "\n\n")
	if a.loading {
		b.WriteString(a.spinner.View() + " loading...\n")
	} else {
		for i, e := range a.logEntries {
			if i > 20 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more", len(a.logEntries)-i)))
				break
			}
			hash := dimStyle.Render(e.Hash[:7])
			msg := normalStyle.Render(e.Subject)
			b.WriteString(fmt.Sprintf("  %s  %s\n", hash, msg))
		}
	}
	b.WriteString("\n" + dimStyle.Render("[q] back"))
	return borderStyle.Render(b.String())
}

// ─────────────────────────────────────────────────────────────
// Commands
// ─────────────────────────────────────────────────────────────

func (a *App) loadBranches() tea.Cmd {
	a.loading = true
	repo := a.repo
	return func() tea.Msg {
		branches, err := repo.Branches()
		if err != nil {
			return errorMsg{err}
		}
		return branchesLoadedMsg{branches}
	}
}

func (a *App) loadLog() tea.Cmd {
	a.loading = true
	repo := a.repo
	return func() tea.Msg {
		entries, err := repo.Log(a.currentBranch, 50)
		if err != nil {
			return errorMsg{err}
		}
		return logLoadedMsg{entries}
	}
}

func (a App) handleEnter() (tea.Model, tea.Cmd) {
	switch a.view {
	case viewCreateBranch:
		name := strings.TrimSpace(a.textInput.Value())
		if name == "" {
			a.status = "branch name cannot be empty"
			a.statusErr = true
			return a, nil
		}
		flow := a.flow
		return a, func() tea.Msg {
			if err := flow.CreateBranch(name); err != nil {
				return errorMsg{err}
			}
			return successMsg{msg: "created " + name}
		}

	case viewMerge:
		src := strings.TrimSpace(a.textInput.Value())
		if src == "" {
			return a, nil
		}
		flow := a.flow
		cur := a.currentBranch
		return a, func() tea.Msg {
			if err := flow.Merge(src, cur); err != nil {
				return errorMsg{err}
			}
			return successMsg{msg: "merged " + src + " → " + cur}
		}
	}
	return a, nil
}

// ─────────────────────────────────────────────────────────────
// list.Item adapter
// ─────────────────────────────────────────────────────────────

type branchItem git.Branch

func (b branchItem) Title() string {
	prefix := "  "
	if git.Branch(b).Current {
		prefix = "* "
	}
	return prefix + b.Name
}
func (b branchItem) Description() string { return b.LastCommit }
func (b branchItem) FilterValue() string { return b.Name }
