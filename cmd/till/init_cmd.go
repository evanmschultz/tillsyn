package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/fsatomic"
	"github.com/evanmschultz/tillsyn/internal/platform"
	"github.com/evanmschultz/tillsyn/internal/templates"
	"github.com/evanmschultz/tillsyn/internal/tui/components"
)

// initJSONPayload is the schema for `till init --json '{...}'` headless
// invocations. `Name` and `Groups` are required; `MCP` is optional and
// defaults to true when omitted (nil pointer). Groups must be a non-empty
// slice of values from allowedInitGroups (`gen`, `go`, `fe`).
// Drop 4c.6.1 W4.D1 renamed `till-gen` -> `gen` and `till-go` -> `go`;
// `fe` added as a new canonical group. `till-gdd` removed entirely.
type initJSONPayload struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
	MCP    *bool    `json:"mcp,omitempty"`
}

// MCPRegistration reports whether MCP server registration is requested.
// A nil MCP pointer (omitted from the --json payload) defaults to true —
// MCP registration is opt-out, not opt-in. This mirrors the
// OrchSelfApprovalIsEnabled() accessor pattern on ProjectMetadata
// (internal/domain/project.go) where a nil pointer also defaults to
// the more-permissive true value.
func (p initJSONPayload) MCPRegistration() bool {
	if p.MCP == nil {
		return true
	}
	return *p.MCP
}

// allowedInitGroups lists the active agent groups `till init` accepts.
// Drop 4c.6.1 W4.D1 renamed `till-gen` -> `gen` and `till-go` -> `go`
// (canonical group names without the `till-` prefix); `fe` is a new
// canonical group added in the same drop. Order is load-bearing for the
// validation error message. `till-gdd` is removed entirely — the GDD
// reserved-group rationale evaporates with the new naming scheme; if a
// reserved group is ever needed again, re-add it to validateInitPayload.
var allowedInitGroups = []string{"gen", "go", "fe"}

// newInitCommand returns the `till init` cobra command.
//
// rootOpts is passed by pointer so the RunE closure reads the live values
// cobra wrote into &rootOpts.appName / &rootOpts.homeDir during flag parse —
// see main.go:508-513 (PersistentFlags().StringVar(&rootOpts.appName, ...)).
// Capturing by value would freeze the pre-parse defaults and ignore --app /
// --home, breaking path resolution for the project-DB record created in D7.
func newInitCommand(stdout io.Writer, rootOpts *rootCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Seed a Tillsyn project (agents directory, agents.toml, .gitignore, optional .mcp.json)",
		Long: strings.TrimSpace(`
Initialize the current working directory as a Tillsyn project: copy the
agent .md files for the chosen group into <project>/.tillsyn/agents/, write
agents.toml from the shipped example, ensure agents.local.toml is gitignored,
optionally register the project with Claude Code via .mcp.json, and create
the project record in the local Tillsyn database.

Run interactively (TUI walk for project name + group picker) or in headless
mode by passing a JSON payload via --json. Re-running till init in an
already-initialized project is safe — every write is idempotent and existing
files are skipped, never overwritten.
`),
		Example: strings.Join([]string{
			"  till init",
			"  till init --json '{\"name\":\"my-project\",\"groups\":[\"go\"],\"mcp\":true}'",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, err := cmd.Flags().GetString("json")
			if err != nil {
				return err
			}
			if strings.TrimSpace(payload) != "" {
				return runInitJSON(stdout, *rootOpts, payload)
			}
			return runInitTUI(stdout, *rootOpts)
		},
	}
	cmd.Flags().String("json", "", "Run init in headless mode with a JSON payload (e.g. --json '{\"name\":\"foo\",\"groups\":[\"go\"],\"mcp\":false}')")
	return cmd
}

// initTUIStep enumerates the bubbletea walk's current state. The walk is a
// three-step linear flow (name → group → mcp) with an explicit completion /
// cancel terminal — keeping the step type closed makes the Update logic
// dispatch on a single switch and the tests assert state directly.
type initTUIStep int

const (
	// initTUIStepName collects the project name via a textinput. Pressing
	// Enter advances to initTUIStepGroup; pressing Esc cancels.
	initTUIStepName initTUIStep = iota

	// initTUIStepGroup collects agent groups via the picker_multi.go
	// component. Space toggles, Enter confirms (minimum 1 required),
	// Esc cancels.
	initTUIStepGroup

	// initTUIStepMCP prompts the user to confirm .mcp.json registration via
	// the confirm.go sub-component. Default answer is YES (default = true
	// per REVISION_BRIEF §2.6). Enter accepts YES; y/Y explicit YES; n/N
	// NO; Esc cancels the walk.
	initTUIStepMCP

	// initTUIStepDone is the terminal state — Done() returns true and the
	// caller reads Payload().
	initTUIStepDone

	// initTUIStepCancelled is the alternate terminal state — Cancelled()
	// returns true and the caller surfaces the cancel as an error.
	initTUIStepCancelled
)

// initTUIModel is the bubbletea model that drives the `till init` walk —
// project name via textinput, agent groups via the picker_multi.go
// component, and a y/n MCP registration confirm via the confirm.go
// sub-component. The model exposes Done() / Cancelled() / Payload() so
// the caller (runInitTUI) can read the final state once tea.Program.Run
// returns the terminal model. The shape mirrors the in-repo textinput
// patterns at `internal/tui/file_picker_core.go` (textinput usage) and
// the keymap idioms at `internal/tui/model.go` (tea.KeyEnter / etc.).
type initTUIModel struct {
	step         initTUIStep
	nameInput    textinput.Model
	groupPicker  components.PickerMultiModel
	mcpConfirm   components.ConfirmModel
	emptyHint    string
	defaultName  string
	finalPayload initJSONPayload
}

// newInitTUIModel constructs the walk model with the project name defaulted
// to filepath.Base(cwd) per SKETCH §9.3 ("default = filepath.Base(cwd);
// user can edit"). The textinput is pre-populated with the default so an
// Enter on the first frame accepts it verbatim.
//
// The group picker is pre-seeded with all allowedInitGroups and defaults
// to "gen" (index 0) selected — pressing Enter immediately on the group
// step confirms Groups = ["gen"] without any navigation.
func newInitTUIModel(cwd string) initTUIModel {
	def := filepath.Base(cwd)
	ti := textinput.New()
	ti.Prompt = "name: "
	ti.Placeholder = def
	ti.CharLimit = 120
	ti.SetValue(def)
	ti.CursorEnd()
	ti.Focus()

	// Construct the multi-select picker for the three canonical groups.
	// Pre-select "gen" (index 0) by sending one Space press while the
	// cursor is at 0 — the picker map starts empty, so the first Space
	// toggles gen to selected.
	picker := components.NewPickerMulti(allowedInitGroups)
	picker, _ = picker.Update(tea.KeyPressMsg{Code: tea.KeySpace})

	return initTUIModel{
		step:        initTUIStepName,
		nameInput:   ti,
		groupPicker: picker,
		mcpConfirm:  components.NewConfirm("Register MCP server in .mcp.json?", true),
		defaultName: def,
	}
}

// Init is the bubbletea entry point. The walk has no async work on entry —
// the textinput is already focused — so we return nil. (Following the
// `internal/tui/model.go` Init convention which returns a single tea.Cmd
// or nil.)
func (m initTUIModel) Init() tea.Cmd {
	return nil
}

// Update advances the walk one event at a time. The name step handles
// Enter (advance) and Esc (cancel) plus forwards all other keypresses to
// the textinput. The group step dispatches to the picker_multi.go component
// and enforces a minimum-1 selection before allowing the walk to advance.
func (m initTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.step {
	case initTUIStepName:
		key, ok := msg.(tea.KeyPressMsg)
		if !ok {
			// Non-key messages (WindowSize, etc.) pass through unchanged.
			return m, nil
		}
		switch {
		case key.Code == tea.KeyEsc:
			m.step = initTUIStepCancelled
			return m, tea.Quit
		case key.Code == tea.KeyEnter:
			value := strings.TrimSpace(m.nameInput.Value())
			if value == "" {
				value = m.defaultName
			}
			m.finalPayload.Name = value
			m.step = initTUIStepGroup
			return m, nil
		default:
			var cmd tea.Cmd
			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd
		}

	case initTUIStepGroup:
		// Intercept Enter before dispatching to the picker: if nothing is
		// selected, show an inline hint and refuse to advance. The picker
		// must NOT receive Enter when selection is empty (once the picker
		// marks itself Done it ignores further input, which would deadlock
		// the walk).
		if kp, ok := msg.(tea.KeyPressMsg); ok && kp.Code == tea.KeyEnter {
			if len(m.groupPicker.Selected()) == 0 {
				m.emptyHint = "select at least one group (space to toggle)"
				return m, nil
			}
			m.emptyHint = ""
		}
		var cmd tea.Cmd
		m.groupPicker, cmd = m.groupPicker.Update(msg)
		if m.groupPicker.Done() {
			if m.groupPicker.Cancelled() {
				m.step = initTUIStepCancelled
				return m, tea.Quit
			}
			m.finalPayload.Groups = m.groupPicker.Selected()
			// Advance to the MCP confirm step (D4). The hardwire is gone.
			m.step = initTUIStepMCP
			return m, nil
		}
		return m, cmd

	case initTUIStepMCP:
		// Intercept Esc directly: Esc cancels the entire walk, distinct from
		// pressing 'n' which is a valid NO answer. The confirm component merges
		// both into Cancelled() so we separate them here before forwarding.
		if kp, ok := msg.(tea.KeyPressMsg); ok && kp.Code == tea.KeyEsc {
			m.step = initTUIStepCancelled
			return m, tea.Quit
		}
		m.mcpConfirm, _ = m.mcpConfirm.Update(msg)
		if m.mcpConfirm.Done() {
			// Confirmed() = true for y/Y/Enter-default-yes (MCP enabled).
			// !Confirmed() = false for n/N (MCP disabled). Both are valid
			// answers — only Esc (handled above) cancels the walk.
			mcpYes := m.mcpConfirm.Confirmed()
			m.finalPayload.MCP = &mcpYes
			m.step = initTUIStepDone
			return m, tea.Quit
		}
		return m, nil

	default:
		// Terminal states ignore further input.
		return m, nil
	}
}

// View renders the current step. The output is intentionally simple — no
// lipgloss styling — because (a) this is a one-shot setup walk where
// clarity matters more than chrome, and (b) tests inspect the View
// substring via teatest, so plain ASCII keeps the assertions stable.
func (m initTUIModel) View() tea.View {
	var b strings.Builder
	switch m.step {
	case initTUIStepName:
		b.WriteString("Project name (enter to accept, esc to cancel)\n\n")
		b.WriteString(m.nameInput.View())
		b.WriteString("\n")
	case initTUIStepGroup:
		b.WriteString("Agent groups (j/k to move, space to toggle, enter to confirm, esc to cancel)\n\n")
		b.WriteString(m.groupPicker.View())
		if m.emptyHint != "" {
			b.WriteString("\n")
			b.WriteString(m.emptyHint)
			b.WriteString("\n")
		}
	case initTUIStepMCP:
		b.WriteString(m.mcpConfirm.View())
		b.WriteString("\n")
	case initTUIStepDone:
		b.WriteString("done\n")
	case initTUIStepCancelled:
		b.WriteString("cancelled\n")
	}
	return tea.NewView(b.String())
}

// Done reports whether the walk completed successfully. Callers MUST check
// Done() OR Cancelled() before reading Payload().
func (m initTUIModel) Done() bool {
	return m.step == initTUIStepDone
}

// Cancelled reports whether the user aborted the walk (Esc). Callers
// surface this as an error rather than running the file-copy pipeline.
func (m initTUIModel) Cancelled() bool {
	return m.step == initTUIStepCancelled
}

// Payload returns the gathered initJSONPayload. Valid only when Done() is
// true; reading Payload() on a cancelled or in-progress walk returns the
// zero value (and the Groups field will be nil/empty).
func (m initTUIModel) Payload() initJSONPayload {
	return m.finalPayload
}

// runInitTUI drives the interactive bubbletea walk (project name + group
// picker) for `till init`. D4 ships the walk; D5 will plug the gathered
// payload into the shared file-copy pipeline.
//
// The walk's final state lives on the returned tea.Model — runInitTUI
// type-asserts back to initTUIModel and reads Done() / Cancelled() /
// Payload() to decide whether to (a) cancel the run, or (b) hand off to
// the D5 stub. The same `programFactory` seam used by `cmd/till/main.go`'s
// TUI command is reused so tests can stub the bubbletea program out when
// needed (D4's tests drive the model directly via teatest_v2; the cobra-
// end-to-end test still surfaces a `till init`-prefixed error because no
// real terminal is attached in `go test`).
func runInitTUI(stdout io.Writer, opts rootCommandOptions) error {
	_ = stdout
	_ = opts

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("till init: resolve cwd: %w", err)
	}
	m := newInitTUIModel(cwd)
	finalModel, err := programFactory(m).Run()
	if err != nil {
		return fmt.Errorf("till init: run tui: %w", err)
	}
	final, ok := finalModel.(initTUIModel)
	if !ok {
		return fmt.Errorf("till init: tui returned unexpected model type %T", finalModel)
	}
	if final.Cancelled() {
		return errors.New("till init: cancelled by user")
	}
	if !final.Done() {
		return errors.New("till init: tui terminated before completing walk")
	}
	// Hand off the gathered payload to the shared file-copy pipeline both
	// runInitTUI and runInitJSON invoke. Behavior is IDENTICAL apart from
	// input source (TUI walk vs --json payload) per SKETCH §26.W2 RiskNote.
	return runInitPipeline(stdout, opts, final.Payload())
}

// runInitJSON parses the headless `--json` payload, validates required
// fields and the group selection, then dispatches to the shared file-copy
// pipeline. D3b ships parse + validation; the file-copy step is a stub
// that D5 fills in. The stub error is the contract D5 consumes, so the
// wording is preserved verbatim across D3b → D5.
func runInitJSON(stdout io.Writer, opts rootCommandOptions, payload string) error {
	_ = stdout
	_ = opts

	var parsed initJSONPayload
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return fmt.Errorf("till init: invalid json payload: %w", err)
	}

	if err := validateInitPayload(parsed); err != nil {
		return err
	}

	// JSON-mode and TUI-mode call the same downstream pipeline — only the
	// input source differs (parsed --json payload vs gathered TUI walk).
	// Per D5 acceptance + SKETCH §26.W2 RiskNote.
	return runInitPipeline(stdout, opts, parsed)
}

// detectFLATLayout checks whether `<destDir>/.tillsyn/agents/` contains any
// `.md` regular files directly at its root — the "FLAT" layout written by
// Drop 4c.6 and earlier sessions. If the directory is absent the check is
// a no-op (returns nil). If a `.md` file is found at the root level (not
// inside a group subdirectory), the function returns a non-nil error with a
// clear remediation instruction.
//
// The check is placed in `runInitPipeline` (not inside `copyAgentFiles`) so
// it survives the D5 rewrite of `copyAgentFiles` independently — see
// W2.D2 ContextBlocks decision.
func detectFLATLayout(destDir string) error {
	agentsDir := filepath.Join(destDir, ".tillsyn", "agents")
	entries, err := os.ReadDir(agentsDir)
	switch {
	case err == nil:
		// Directory exists — scan for .md files at root.
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				return fmt.Errorf("FLAT agent layout detected at %s/. Remove it and re-run: rm -rf %s && till init --group <group>",
					agentsDir, agentsDir)
			}
		}
		return nil
	case errors.Is(err, fs.ErrNotExist):
		// Directory absent — nothing to detect.
		return nil
	default:
		return fmt.Errorf("till init: stat %q: %w", agentsDir, err)
	}
}

// detectOldSchemaAgentsTOML checks whether `<destDir>/agents.toml` uses the
// old `[agents.<kind>]` TOML schema (the schema from before Drop 4c.6.1
// migrated to the new group-keyed format). The check reads only the first 20
// lines — a pragmatic heuristic sufficient to catch the most common placement
// of section headers. If any line stripped of leading whitespace starts with
// the exact prefix `[agents.` (with trailing dot), the function returns a
// non-nil error with a clear remediation instruction.
//
// If the file is absent the check is a no-op (returns nil) — first-time
// `till init` runs have no agents.toml yet.
//
// The 20-line bound is documented in W2.D2 RiskNotes: a user with a very long
// comment block (> 20 lines) before the first section header will not be
// detected. 20 lines is considered a reasonable pragmatic bound.
func detectOldSchemaAgentsTOML(destDir string) error {
	path := filepath.Join(destDir, "agents.toml")
	f, err := os.Open(path)
	switch {
	case err == nil:
		// File open — scan first 20 lines.
		defer func() { _ = f.Close() }()
		sc := bufio.NewScanner(f)
		lineNum := 0
		for sc.Scan() && lineNum < 20 {
			lineNum++
			if strings.HasPrefix(strings.TrimSpace(sc.Text()), "[agents.") {
				tomlPath := filepath.Join(destDir, "agents.toml")
				return fmt.Errorf("agents.toml uses the old [agents.kind] schema. Remove it and re-run: rm %s && till init --group <group>",
					tomlPath)
			}
		}
		return nil
	case errors.Is(err, fs.ErrNotExist):
		// File absent — no-op.
		return nil
	default:
		return fmt.Errorf("till init: open %q: %w", path, err)
	}
}

// runInitPipeline is the shared post-input file-copy pipeline both
// runInitTUI and runInitJSON invoke. It resolves the destination
// directory (cwd), runs the three idempotent copy steps in order,
// registers the optional `.mcp.json` entry, creates the project-DB
// record (idempotent — re-run skips if the project name already exists),
// and writes the Laslig success summary to stdout.
//
// `destDir` is derived from `os.Getwd()` (NOT from `opts.appName` /
// `opts.homeDir`) — the destination is cwd-relative. The project-DB
// record creation in D7 uses path resolution via
// `platform.DefaultPathsWithOptions(opts)` which reads the live
// flag values written by cobra before RunE fires — the D3a pointer fix
// on rootCommandOptions ensures opts carries the correct values.
func runInitPipeline(stdout io.Writer, opts rootCommandOptions, payload initJSONPayload) error {
	destDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("till init: resolve cwd: %w", err)
	}

	// Pre-flight checks: fail loud if a known-bad state is detected. Both
	// checks run before any file-copy side effects (no partial writes on
	// failure). See W2.D2 for FLAT layout and old-schema detection rationale.
	if err := detectFLATLayout(destDir); err != nil {
		return err
	}
	if err := detectOldSchemaAgentsTOML(destDir); err != nil {
		return err
	}

	// Resolve homeDir for HOME-tier template lookups. opts.homeDir is the
	// --home flag value (empty when not set); fall back to os.UserHomeDir().
	homeDir := strings.TrimSpace(opts.homeDir)
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("till init: resolve home dir: %w", err)
		}
	}

	// D5: multi-group subdir copy. copyAgentFiles iterates over all groups
	// and writes to .tillsyn/agents/<group>/<name>.md.
	agentsAdded, agentsSkipped, err := copyAgentFiles(destDir, payload.Groups)
	if err != nil {
		return fmt.Errorf("till init: copy agent files: %w", err)
	}

	// D6: aggregate template.toml from HOME tier or embedded defaults.
	templateAdded, templateSkipped, templateTOMLStatusFromWrite, err := writeTemplateTOML(stdout, destDir, payload.Groups, homeDir)
	if err != nil {
		return fmt.Errorf("till init: write template.toml: %w", err)
	}

	tomlAdded, _, err := copyAgentsTOML(destDir)
	if err != nil {
		return fmt.Errorf("till init: copy agents.toml: %w", err)
	}
	if err := ensureGitignore(destDir); err != nil {
		return fmt.Errorf("till init: ensure .gitignore: %w", err)
	}

	mcpAdded, mcpSkipped, err := registerMCPJSON(destDir, payload.MCPRegistration())
	if err != nil {
		return fmt.Errorf("till init: register .mcp.json: %w", err)
	}

	dbStatus, err := createProjectDBRecord(context.Background(), opts, payload)
	if err != nil {
		return fmt.Errorf("till init: create project DB record: %w", err)
	}

	// Laslig success summary.
	agentsDir := filepath.Join(destDir, ".tillsyn", "agents")
	agentsTomlPath := filepath.Join(destDir, "agents.toml")
	gitignoreStatus := "ensured"
	mcpStatus := "skipped (mcp:false)"
	if payload.MCPRegistration() {
		if mcpAdded > 0 {
			mcpStatus = "added"
		} else if mcpSkipped > 0 {
			mcpStatus = "already exists"
		}
	}
	agentsCopied := fmt.Sprintf("added=%d skipped=%d", agentsAdded, agentsSkipped)
	agentsTOMLStatus := "skipped (already exists)"
	if tomlAdded > 0 {
		agentsTOMLStatus = "added"
	}
	// Use the status string returned directly from writeTemplateTOML so the
	// Laslig row is accurate for all three cases: "added", "skipped (already
	// exists)", and "skipped (multi-group — uses per-group HOME/embedded
	// resolution)". The templateAdded and templateSkipped counters are kept for
	// symmetry with other (added,skipped,err) return patterns.
	templateTOMLStatus := templateTOMLStatusFromWrite
	_, _ = templateAdded, templateSkipped // counters available for future use

	return writeCLIKV(stdout, "Init", [][2]string{
		{"project name", payload.Name},
		{"groups", strings.Join(payload.Groups, ",")},
		{"agents dir", agentsDir},
		{"agents copied", agentsCopied},
		{"agents.toml", agentsTOMLStatus + " — " + agentsTomlPath},
		{"template.toml", templateTOMLStatus},
		{".gitignore", gitignoreStatus},
		{".mcp.json", mcpStatus},
		{"project DB", dbStatus},
	})
}

// detectBareRoot attempts to resolve the git bare-root path for cwd by running
// `git rev-parse --git-common-dir`. This command prints the common git
// directory shared by all worktrees — a relative path such as `.git` for a
// regular repo or an absolute path for a linked worktree.
//
// Failure policy (non-fatal): if git is absent from PATH, or the command exits
// non-zero (cwd is not a git repo), an empty string is returned. This mirrors
// the RepoBareRoot zero-value semantics in domain.Project (empty = not yet
// bootstrapped) per Drop 4a L4 WAVE_1_PLAN.md §1.8.
//
// Path resolution: trimmed output is checked with filepath.IsAbs. If it is
// already absolute (linked-worktree case — git returns the main repo's .git
// as an absolute path), the trimmed value is used directly. If relative (e.g.
// ".git"), it is joined with cwd before calling filepath.Abs. The cmd.Dir is
// set to cwd so the git subprocess runs in the intended directory regardless of
// the process working directory at call time.
func detectBareRoot(ctx context.Context, cwd string) string {
	if _, lookErr := exec.LookPath("git"); lookErr != nil {
		// git absent from PATH — non-fatal.
		return ""
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		// Not a git repo or git returned non-zero — non-fatal.
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	// Avoid filepath.Join(cwd, "/abs/path") concatenation garbage: use the
	// absolute path directly when git returns one (linked-worktree case).
	combined := trimmed
	if !filepath.IsAbs(trimmed) {
		combined = filepath.Join(cwd, trimmed)
	}
	abs, absErr := filepath.Abs(combined)
	if absErr != nil {
		return ""
	}
	return abs
}

// mapGroupsToLanguage maps the first element of groups to the project's
// Language field using the closed language enum: "go" -> "go", "fe" -> "fe",
// anything else (including "gen" and multi-word options) -> "" (no language
// bias). Selection-order wins: the user's first group pick determines the
// primary language. The fixed go-priority heuristic was explicitly rejected per
// plan-QA NIT5 — user intent expressed through group order is the policy.
//
// An empty groups slice returns "" without panicking. This should never occur
// after validateInitPayload, but the function is defensive.
func mapGroupsToLanguage(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	switch groups[0] {
	case "go":
		return "go"
	case "fe":
		return "fe"
	default:
		// "gen" and any future unmapped groups have no language bias.
		return ""
	}
}

// createProjectDBRecord opens the Tillsyn SQLite database, then either
// creates a new project record for the init payload or skips creation if a
// project with the same name already exists (idempotency — re-running
// `till init` in an already-initialized directory is safe). Returns a short
// human-readable status string suitable for the Laslig summary row.
//
// Service wiring clones the pattern from executeCommandFlow in main.go:
// platform.DefaultPathsWithOptions → parent-dir creation → sqlite.Open →
// app.NewService(minimal config). Only the project-creation path is exercised;
// no auth, no embeddings, no live-wait broker needed here.
//
// Field population:
//   - Name = payload.Name
//   - RepoPrimaryWorktree = os.Getwd() (absolute cwd at call time)
//   - RepoBareRoot = git rev-parse --git-common-dir result, resolved to
//     absolute path; empty string if git is absent or cwd is not a git repo
//   - Language = payload.Groups[0] mapped through the closed enum (go/fe/gen)
//   - Metadata.Groups = payload.Groups (typed []string field from W1.D2)
func createProjectDBRecord(ctx context.Context, opts rootCommandOptions, payload initJSONPayload) (string, error) {
	paths, err := platform.DefaultPathsWithOptions(platform.Options{
		AppName: opts.appName,
		HomeDir: opts.homeDir,
	})
	if err != nil {
		return "", fmt.Errorf("resolve runtime paths: %w", err)
	}

	dbPath := strings.TrimSpace(opts.dbPath)
	if dbPath == "" {
		dbPath = paths.DBPath
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return "", fmt.Errorf("create database directory: %w", err)
	}

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("open database %q: %w", dbPath, err)
	}
	defer func() { _ = repo.Close() }()

	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
	})

	// Idempotency: scan existing projects for a name match before creating.
	existing, err := svc.ListProjects(ctx, false)
	if err != nil {
		return "", fmt.Errorf("list existing projects: %w", err)
	}
	for _, p := range existing {
		if strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(payload.Name)) {
			return "already exists — skipped", nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve cwd for project record: %w", err)
	}

	_, err = svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{
		Name:                payload.Name,
		RepoPrimaryWorktree: cwd,
		RepoBareRoot:        detectBareRoot(ctx, cwd),
		Language:            mapGroupsToLanguage(payload.Groups),
		Metadata: domain.ProjectMetadata{
			Groups: payload.Groups,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create project %q: %w", payload.Name, err)
	}
	return "created", nil
}

// agentFileInitPerm is the permission applied to every freshly copied
// agent .md, agents.toml, .gitignore, and template.toml write. Matches the
// conventional 0o644 (user rw, group r, other r) the embedded fixtures
// themselves ship with under git.
const agentFileInitPerm os.FileMode = 0o644

// templateGroupMarkerPrefix is the comment prefix written at the top of a
// newly created .tillsyn/template.toml file so the partial-state check on
// re-run can identify which groups the file was generated for — without
// embedding a TOML `[<group>]` section header that would nest the template
// content inside a sub-table and break templates.Load schema_version detection.
//
// Format: `# till-init-groups: go,fe` (comma-separated, ascending sort order).
// The partial-state check in writeTemplateTOML looks for this prefix AND for
// the legacy `[<group>]` / `[<group>.]` patterns so that hand-authored files
// using the old section-headed format continue to suppress the WARN.
const templateGroupMarkerPrefix = "# till-init-groups: "

// writeTemplateTOML aggregates template TOML content for each group in
// `groups` and writes the result to `<destDir>/.tillsyn/template.toml`.
//
// Source resolution per group (HOME-first, embedded fallback):
//  1. HOME tier: `<homeDir>/.tillsyn/templates/<group>.toml` — if this file
//     exists it is used as-is. This lets users override the shipped defaults
//     with project-specific or org-specific templates.
//  2. Embedded fallback: `builtin/till-<group>.toml` from
//     `templates.DefaultTemplateFS` — used when the HOME-tier file is absent.
//
// The aggregate content is written WITHOUT `[<group>]` TOML section headers.
// Each group's template content is a valid top-level templates.Template TOML
// body (schema_version = "v1" at the top level). Writing section headers
// nests schema_version under [group], which breaks templates.Load used by
// bakeProjectKindCatalog when RepoPrimaryWorktree is set by `till init`. For
// multi-group projects the first group's template content is written at the
// top level; subsequent groups are concatenated (merged in template-load
// order). For partial-state detection a comment marker is written at the very
// top of the file (format: "# till-init-groups: go,fe") so re-runs can
// identify missing groups without relying on TOML section headers.
//
// Blanket skip: if `<destDir>/.tillsyn/template.toml` already exists the
// function does NOT overwrite it (returns added=0, skipped=1, err=nil) — users
// may have customized the file. Partial-state warning: if the existing file
// is missing the group marker or a `[<group>]` or `[<group>.` section for
// one or more selected groups, a WARN line is printed to stdout (non-fatal —
// exits zero).
//
// `homeDir` must be an absolute path. The caller (`runInitPipeline`) resolves
// it via `os.UserHomeDir()` when `rootOpts.homeDir` is empty, so by the time
// `writeTemplateTOML` is called `homeDir` is always absolute.
//
// Returns `(added, skipped, status, err)`: added=1 when the file was written,
// skipped=1 when the file already existed. status is a short human-readable
// string for the Laslig summary row — one of:
//
//   - "added" — file was freshly written (single-group).
//   - "skipped (already exists)" — file existed; no write performed.
//   - "skipped (multi-group — uses per-group HOME/embedded resolution)" — file
//     deliberately not written because per-group fallback resolution handles
//     multi-group projects (PLATFORM-TEMPLATES-R1 deferred).
//
// On error the error is wrapped and status is empty.
func writeTemplateTOML(stdout io.Writer, destDir string, groups []string, homeDir string) (int, int, string, error) {
	target := filepath.Join(destDir, ".tillsyn", "template.toml")

	// Blanket skip with optional partial-state warning.
	existing, statErr := os.ReadFile(target)
	if statErr == nil {
		// File exists — check for missing groups via the marker comment or
		// legacy section headers. Both forms suppress the WARN so hand-authored
		// files using `[<group>]` TOML headers continue to work.
		content := string(existing)
		var missing []string
		for _, group := range groups {
			inMarker := templateGroupMarkerPresent(content, group)
			inSection := strings.Contains(content, "["+group+"]") || strings.Contains(content, "["+group+".")
			if !inMarker && !inSection {
				missing = append(missing, group)
			}
		}
		if len(missing) > 0 {
			fmt.Fprintf(stdout, "WARN: %s already exists but is missing sections for group(s): %v. Remove it and re-run to regenerate.\n",
				target, missing)
		}
		return 0, 1, "skipped (already exists)", nil
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return 0, 0, "", fmt.Errorf("stat %q: %w", target, statErr)
	}

	// File absent — write the group marker comment followed by the template
	// body for single-group projects.
	//
	// For single-group projects: write the template content as-is (no
	// [<group>] section header). The embedded template is a valid top-level
	// templates.Template TOML body with schema_version = "v1" at the top
	// level. Prepending a [<group>] header would nest schema_version inside a
	// sub-table and break bakeProjectKindCatalog's templates.Load call when
	// RepoPrimaryWorktree is populated (D7 fix).
	//
	// For multi-group projects: skip writing template.toml entirely. Multi-group
	// projects rely on bakeProjectKindCatalog's per-group HOME-tier and
	// embedded-default resolution (loadProjectTemplatesForGroups iterates each
	// group independently). Writing a project-level template.toml for multi-group
	// would require a semantic merge of potentially conflicting TOML documents
	// (both till-go.toml and till-fe.toml declare [kinds.plan], [kinds.build],
	// etc.); templates.Load rejects naive concatenations with "table plan already
	// exists". The PLATFORM-TEMPLATES-R1 refinement item tracks a future proper
	// multi-group template.toml aggregation path.
	if len(groups) != 1 {
		return 0, 0, "skipped (multi-group — uses per-group HOME/embedded resolution)", nil
	}

	var buf strings.Builder
	buf.WriteString(templateGroupMarkerPrefix)
	buf.WriteString(groups[0])
	buf.WriteString("\n")

	data, err := readTemplateForGroup(homeDir, groups[0])
	if err != nil {
		return 0, 0, "", fmt.Errorf("read template for group %q: %w", groups[0], err)
	}
	content := strings.TrimRight(string(data), "\n")
	buf.WriteString(content)
	buf.WriteString("\n")

	// Ensure the .tillsyn directory exists.
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, 0, "", fmt.Errorf("mkdir %q: %w", filepath.Dir(target), err)
	}

	if err := fsatomic.WriteFile(target, []byte(buf.String()), agentFileInitPerm); err != nil {
		return 0, 0, "", fmt.Errorf("write %q: %w", target, err)
	}
	return 1, 0, "added", nil
}

// templateGroupMarkerPresent reports whether content contains a
// `# till-init-groups: ...` comment that lists group as one of the
// comma-separated entries. The check is case-sensitive and trims spaces
// around the entry separator to handle `go, fe` and `go,fe` variants.
func templateGroupMarkerPresent(content, group string) bool {
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, templateGroupMarkerPrefix) {
			continue
		}
		rest := strings.TrimPrefix(line, templateGroupMarkerPrefix)
		for _, entry := range strings.Split(rest, ",") {
			if strings.TrimSpace(entry) == group {
				return true
			}
		}
	}
	return false
}

// readTemplateForGroup returns the TOML content for `group` from the HOME
// tier (`<homeDir>/.tillsyn/templates/<group>.toml`) if present, otherwise
// falls back to the embedded `builtin/till-<group>.toml`.
func readTemplateForGroup(homeDir, group string) ([]byte, error) {
	homePath := filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")
	data, err := os.ReadFile(homePath)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read HOME template %q: %w", homePath, err)
	}
	// HOME tier absent — fall back to embedded.
	embeddedPath := path.Join("builtin", "till-"+group+".toml")
	embedded, readErr := fs.ReadFile(templates.DefaultTemplateFS, embeddedPath)
	if readErr != nil {
		return nil, fmt.Errorf("read embedded template %q: %w", embeddedPath, readErr)
	}
	return embedded, nil
}

// copyAgentFiles reads the embedded `internal/templates/builtin/agents/<group>/*.md`
// set via `templates.DefaultTemplateFS` for each group in `groups` and writes
// each entry to `<destDir>/.tillsyn/agents/<group>/<name>.md` (subdir-per-group).
// Each write uses `fsatomic.WriteFile` (write-temp-in-same-dir + rename).
// Existing destination files are SKIPPED, never overwritten — re-run safety.
//
// Embed source paths use the canonical W4.D1 unprefixed group names: `go`,
// `fe`, `gen`. The destination directory `<destDir>/.tillsyn/agents/<group>/`
// is created for each group before any file writes.
//
// FLAT detection (detecting old flat-layout `.md` files directly under
// `.tillsyn/agents/`) lives in `runInitPipeline`, NOT in this function.
// This preserves the W2.D2 check independently of the D5 signature refactor.
//
// Returns `(added, skippedExisting, err)`. Both counters are aggregated
// across all groups in the slice. On error the partial-progress counts so
// far are returned alongside the wrapped error.
func copyAgentFiles(destDir string, groups []string) (int, int, error) {
	added, skipped := 0, 0
	for _, group := range groups {
		srcDir := path.Join("builtin", "agents", group)
		entries, err := fs.ReadDir(templates.DefaultTemplateFS, srcDir)
		if err != nil {
			return added, skipped, fmt.Errorf("read embedded %q: %w", srcDir, err)
		}

		groupDir := filepath.Join(destDir, ".tillsyn", "agents", group)
		if err := os.MkdirAll(groupDir, 0o755); err != nil {
			return added, skipped, fmt.Errorf("mkdir %q: %w", groupDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			target := filepath.Join(groupDir, entry.Name())
			if _, statErr := os.Stat(target); statErr == nil {
				skipped++
				continue
			} else if !errors.Is(statErr, fs.ErrNotExist) {
				return added, skipped, fmt.Errorf("stat %q: %w", target, statErr)
			}

			srcPath := path.Join(srcDir, entry.Name())
			data, readErr := fs.ReadFile(templates.DefaultTemplateFS, srcPath)
			if readErr != nil {
				return added, skipped, fmt.Errorf("read embedded %q: %w", srcPath, readErr)
			}
			if err := fsatomic.WriteFile(target, data, agentFileInitPerm); err != nil {
				return added, skipped, fmt.Errorf("write %q: %w", target, err)
			}
			added++
		}
	}
	return added, skipped, nil
}

// copyAgentsTOML copies the embedded `internal/templates/builtin/agents.example.toml`
// fixture to `<destDir>/agents.toml` atomically via `fsatomic.WriteFile`.
// If `<destDir>/agents.toml` already exists, the copy is SKIPPED — re-run
// safety.
//
// Returns `(added, skippedExisting, err)`. `added` is either 0 (target
// already existed) or 1 (target created).
func copyAgentsTOML(destDir string) (int, int, error) {
	target := filepath.Join(destDir, "agents.toml")
	if _, statErr := os.Stat(target); statErr == nil {
		return 0, 1, nil
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return 0, 0, fmt.Errorf("stat %q: %w", target, statErr)
	}

	const srcPath = "builtin/agents.example.toml"
	data, err := fs.ReadFile(templates.DefaultTemplateFS, srcPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read embedded %q: %w", srcPath, err)
	}
	if err := fsatomic.WriteFile(target, data, agentFileInitPerm); err != nil {
		return 0, 0, fmt.Errorf("write %q: %w", target, err)
	}
	return 1, 0, nil
}

// gitignoreAgentsLocalLine is the literal line `ensureGitignore` adds to
// `<destDir>/.gitignore` when it is not already present. Match is
// line-exact (trim-equal) per W2-FF10 round-2 LOCKED line-iteration fix.
const gitignoreAgentsLocalLine = "agents.local.toml"

// ensureGitignore guarantees `<destDir>/.gitignore` contains a line whose
// trimmed value equals `agents.local.toml`. If `.gitignore` is absent the
// file is created with just that line. If it exists, the body is
// line-iterated (NOT raw bytes.Contains) and the line is appended only
// when not already present.
//
// W2-FF10 round-2 LOCKED rationale: a raw `bytes.Contains(data,
// []byte("\nagents.local.toml\n"))` form requires a leading `\n` and
// misses the first-line-only case (file consists solely of
// `agents.local.toml\n` from a prior run with no preceding entries).
// Line-iteration via `bufio.Scanner` against the file content handles
// both the first-line-only case AND trailing-whitespace-on-line variants
// uniformly.
//
// Trailing-newline handling: if an existing file does NOT end with `\n`,
// the appended block starts with `\n` so the new line lands on its own
// line and the final file still ends with `\n`.
//
// Every write goes through `fsatomic.WriteFile` so the file is either
// fully present with the new contents or untouched — never observed
// half-written by a concurrent reader on POSIX.
func ensureGitignore(destDir string) error {
	target := filepath.Join(destDir, ".gitignore")

	data, err := os.ReadFile(target)
	switch {
	case err == nil:
		// File exists — line-iterate to check presence.
		if gitignoreLinePresent(data, gitignoreAgentsLocalLine) {
			return nil
		}
		body := data
		if len(body) > 0 && body[len(body)-1] != '\n' {
			body = append(body, '\n')
		}
		body = append(body, []byte(gitignoreAgentsLocalLine+"\n")...)
		if err := fsatomic.WriteFile(target, body, agentFileInitPerm); err != nil {
			return fmt.Errorf("write %q: %w", target, err)
		}
		return nil
	case errors.Is(err, fs.ErrNotExist):
		// File absent — create with just the line.
		body := []byte(gitignoreAgentsLocalLine + "\n")
		if err := fsatomic.WriteFile(target, body, agentFileInitPerm); err != nil {
			return fmt.Errorf("write %q: %w", target, err)
		}
		return nil
	default:
		return fmt.Errorf("read %q: %w", target, err)
	}
}

// gitignoreLinePresent reports whether `data` contains a line whose
// trimmed value equals `want`. Implements the W2-FF10 round-2 LOCKED
// line-iteration fix — see `ensureGitignore` for the rationale.
func gitignoreLinePresent(data []byte, want string) bool {
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == want {
			return true
		}
	}
	return false
}

// validateInitPayload checks required fields and the group selection on
// a parsed initJSONPayload. Returns an error pointing at the first failed
// invariant: Name is required; Groups must be non-empty with each element
// in allowedInitGroups. Invalid groups surface a clear list of allowed
// values and the first invalid group name encountered.
func validateInitPayload(p initJSONPayload) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("till init: name required")
	}
	if len(p.Groups) == 0 {
		return errors.New("till init: groups required (must supply at least one group)")
	}
	var invalid []string
	for _, g := range p.Groups {
		found := false
		for _, allowed := range allowedInitGroups {
			if g == allowed {
				found = true
				break
			}
		}
		if !found {
			invalid = append(invalid, g)
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("till init: invalid group(s) %v; allowed: %v", invalid, allowedInitGroups)
	}
	return nil
}

// mcpServerEntry holds the configuration for the `tillsyn` stdio MCP server
// entry that `registerMCPJSON` writes into `.mcp.json`. The schema matches
// Claude Code's stdio transport format: `command` is the path to the server
// binary, `args` are optional CLI arguments, `env` is an optional environment
// variable map. The `type` field is omitted — Claude Code treats absence as
// `stdio`.
//
// This struct is used ONLY for constructing the new `tillsyn` entry. All
// pre-existing entries (including HTTP/SSE/SDK entries authored by
// `claude mcp add --transport http ...`) are preserved verbatim as
// json.RawMessage and are never deserialized through this typed struct.
//
// Source: Context7 `/websites/code_claude` → "Configure stdio MCP Server in
// .mcp.json". Schema verified against the canonical Claude Code docs.
type mcpServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// mcpServersKey is the JSON key for the server map inside `.mcp.json`.
const mcpServersKey = "mcpServers"

// mcpJSONFileName is the filename written by registerMCPJSON.
const mcpJSONFileName = ".mcp.json"

// tillsyn MCP server key in .mcp.json.
const mcpServerKey = "tillsyn"

// registerMCPJSON optionally registers the `tillsyn` MCP server entry in
// `<destDir>/.mcp.json`. When `includeMCP` is false the function is a
// no-op (re-run safe; reports added=0, skipped=1 for the opt-out case).
//
// When `includeMCP` is true:
//   - The `till` binary path is resolved via exec.LookPath("till"); on
//     failure the canonical install path (`~/.local/bin/till`) is used
//     instead (per magefile.go:144 which writes the binary there).
//   - If `.mcp.json` is absent a minimal file with just the `tillsyn` entry
//     is created atomically via fsatomic.WriteFile.
//   - If `.mcp.json` exists it is parsed, checked for a pre-existing
//     `tillsyn` entry (idempotent — no duplicate, no overwrite), and if
//     absent the entry is added and the file rewritten atomically.
//
// Preservation contract (Drop 4c.6 W2.D6 Round-2 FF1 fix): the file is
// parsed as a two-level map[string]json.RawMessage so that all pre-existing
// server entries (stdio, HTTP, SSE, SDK) and all sibling top-level keys are
// preserved JSON-semantically on rewrite (each server-entry value is stored
// as raw bytes and round-trips unchanged; the top-level object is
// re-indented by json.MarshalIndent). Only the new `tillsyn` entry is
// deserialized through the typed mcpServerEntry struct.
//
// Returns (added, skipped, err). `added` is 1 when the entry was created;
// `skipped` is 1 when the entry already existed or `includeMCP` is false.
func registerMCPJSON(destDir string, includeMCP bool) (int, int, error) {
	if !includeMCP {
		return 0, 1, nil
	}

	tillBin, err := exec.LookPath("till")
	if err != nil {
		// Fall back to the canonical dev-install path written by
		// `mage install` per magefile.go:144.
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return 0, 0, fmt.Errorf("resolve home dir for till binary path: %w", homeErr)
		}
		tillBin = filepath.Join(home, ".local", "bin", "till")
	}

	// Serialize the new tillsyn entry once; reused in both the file-exists
	// and the fresh-file branches.
	tillsyn := mcpServerEntry{Command: tillBin}
	tillsynRaw, marshalErr := json.Marshal(tillsyn)
	if marshalErr != nil {
		return 0, 0, fmt.Errorf("marshal tillsyn entry: %w", marshalErr)
	}

	target := filepath.Join(destDir, mcpJSONFileName)
	data, readErr := os.ReadFile(target)
	switch {
	case readErr == nil:
		// File exists — parse the entire file as a raw top-level JSON object
		// so sibling keys beyond "mcpServers" are preserved verbatim.
		var topLevel map[string]json.RawMessage
		if unmarshalErr := json.Unmarshal(data, &topLevel); unmarshalErr != nil {
			return 0, 0, fmt.Errorf("parse %q: %w", target, unmarshalErr)
		}
		if topLevel == nil {
			topLevel = make(map[string]json.RawMessage)
		}

		// Parse the mcpServers sub-map as raw messages so every pre-existing
		// server entry (stdio, HTTP, SSE, SDK) is preserved byte-equivalent.
		servers := make(map[string]json.RawMessage)
		if raw, ok := topLevel[mcpServersKey]; ok && len(raw) > 0 {
			if unmarshalErr := json.Unmarshal(raw, &servers); unmarshalErr != nil {
				return 0, 0, fmt.Errorf("parse %q mcpServers: %w", target, unmarshalErr)
			}
			// Guard against {"mcpServers":null}: json.Unmarshal of a JSON null
			// value into a map pointer sets it to nil even though the map was
			// pre-initialised above. Without this guard the write at
			// servers[mcpServerKey] panics with "assignment to entry in nil map".
			if servers == nil {
				servers = make(map[string]json.RawMessage)
			}
		}

		if _, found := servers[mcpServerKey]; found {
			// Entry already present — idempotent skip.
			return 0, 1, nil
		}

		// Add the new tillsyn entry and write the servers map back into the
		// top-level object (all other top-level keys remain untouched).
		servers[mcpServerKey] = json.RawMessage(tillsynRaw)
		serversRaw, marshalErr2 := json.Marshal(servers)
		if marshalErr2 != nil {
			return 0, 0, fmt.Errorf("marshal %q mcpServers: %w", target, marshalErr2)
		}
		topLevel[mcpServersKey] = json.RawMessage(serversRaw)
		out, marshalErr3 := json.MarshalIndent(topLevel, "", "  ")
		if marshalErr3 != nil {
			return 0, 0, fmt.Errorf("marshal %q: %w", target, marshalErr3)
		}
		if writeErr := fsatomic.WriteFile(target, append(out, '\n'), agentFileInitPerm); writeErr != nil {
			return 0, 0, fmt.Errorf("write %q: %w", target, writeErr)
		}
		return 1, 0, nil

	case errors.Is(readErr, fs.ErrNotExist):
		// File absent — create a minimal file with just the tillsyn entry.
		servers := map[string]json.RawMessage{
			mcpServerKey: json.RawMessage(tillsynRaw),
		}
		serversRaw, marshalErr2 := json.Marshal(servers)
		if marshalErr2 != nil {
			return 0, 0, fmt.Errorf("marshal new %q mcpServers: %w", target, marshalErr2)
		}
		topLevel := map[string]json.RawMessage{
			mcpServersKey: json.RawMessage(serversRaw),
		}
		out, marshalErr3 := json.MarshalIndent(topLevel, "", "  ")
		if marshalErr3 != nil {
			return 0, 0, fmt.Errorf("marshal new %q: %w", target, marshalErr3)
		}
		if writeErr := fsatomic.WriteFile(target, append(out, '\n'), agentFileInitPerm); writeErr != nil {
			return 0, 0, fmt.Errorf("write new %q: %w", target, writeErr)
		}
		return 1, 0, nil

	default:
		return 0, 0, fmt.Errorf("read %q: %w", target, readErr)
	}
}
