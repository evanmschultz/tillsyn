package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/evanmschultz/tillsyn/internal/fsatomic"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// initJSONPayload is the schema for `till init --json '{...}'` headless
// invocations. `Name` and `Group` are required; `MCP` defaults to false
// (zero value). Group must be one of the W2-supported values
// (`till-gen`, `till-go`); `till-gdd` is greyed-out per SKETCH §9.3 and
// rejected as reserved.
type initJSONPayload struct {
	Name  string `json:"name"`
	Group string `json:"group"`
	MCP   bool   `json:"mcp"`
}

// allowedInitGroups lists the active agent groups `till init` accepts in
// W2. `till-gdd` is deliberately omitted — it is reserved per SKETCH §9.3
// and will be re-enabled once GDD methodology lands post-dogfood. Order
// is preserved for the validation error message.
var allowedInitGroups = []string{"till-gen", "till-go"}

// reservedInitGroups lists groups recognized in the schema but rejected
// at validation time. Each entry returns a tailored "reserved" error so
// callers can distinguish typos (unknown group) from intentional-but-not-
// yet-shipped groups.
var reservedInitGroups = map[string]string{
	"till-gdd": "till-gdd",
}

// newInitCommand returns the `till init` cobra command. D3a ships the
// skeleton: --json flag wired (default ""), RunE dispatches to a TUI stub
// or a JSON stub. Subsequent droplets fill the bodies — D3b wires the JSON
// payload parser, D4 wires runInitTUI's bubbletea walk, D5 wires the
// file-copy pipeline both branches share.
func newInitCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command {
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
			"  till init --json '{\"name\":\"my-project\",\"group\":\"till-go\",\"mcp\":true}'",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, err := cmd.Flags().GetString("json")
			if err != nil {
				return err
			}
			if strings.TrimSpace(payload) != "" {
				return runInitJSON(stdout, rootOpts, payload)
			}
			return runInitTUI(stdout, rootOpts)
		},
	}
	cmd.Flags().String("json", "", "Run init in headless mode with a JSON payload (e.g. --json '{\"name\":\"foo\",\"group\":\"till-go\",\"mcp\":false}')")
	return cmd
}

// initTUIStep enumerates the bubbletea walk's current state. The walk is a
// two-step linear flow (name → group) with an explicit completion / cancel
// terminal — keeping the step type closed makes the Update logic dispatch
// on a single switch and the tests assert state directly.
type initTUIStep int

const (
	// initTUIStepName collects the project name via a textinput. Pressing
	// Enter advances to initTUIStepGroup; pressing Esc cancels.
	initTUIStepName initTUIStep = iota

	// initTUIStepGroup collects the agent group via a small cursor over
	// initTUIGroupRows. Pressing Enter on an enabled row finalizes; Esc
	// cancels.
	initTUIStepGroup

	// initTUIStepDone is the terminal state — Done() returns true and the
	// caller reads Payload().
	initTUIStepDone

	// initTUIStepCancelled is the alternate terminal state — Cancelled()
	// returns true and the caller surfaces the cancel as an error.
	initTUIStepCancelled
)

// initTUIGroupRow models one row in the group picker. `Disabled` rows are
// rendered (so the user sees them) but the cursor skips past them on
// movement and Enter is a no-op while the cursor sits on one (per SKETCH
// §9.3 — `till-gdd` is shown but unselectable until GDD methodology lands).
type initTUIGroupRow struct {
	Name     string
	Disabled bool
}

// initTUIGroupRows is the static picker model the walk renders. Order is
// load-bearing — the cursor defaults to row 0 (`till-gen`) so the most
// common pick is one Enter away.
var initTUIGroupRows = []initTUIGroupRow{
	{Name: "till-gen", Disabled: false},
	{Name: "till-go", Disabled: false},
	{Name: "till-gdd", Disabled: true},
}

// initTUIModel is the bubbletea model that drives the `till init` walk —
// project name via textinput, agent group via a small inline picker. The
// model exposes Done() / Cancelled() / Payload() so the caller
// (runInitTUI) can read the final state once tea.Program.Run returns the
// terminal model. The shape mirrors the in-repo textinput patterns at
// `internal/tui/file_picker_core.go` (textinput usage) and the keymap
// idioms at `internal/tui/model.go` (tea.KeyEnter / tea.KeyDown / etc.).
type initTUIModel struct {
	step         initTUIStep
	nameInput    textinput.Model
	groupCursor  int
	defaultName  string
	finalPayload initJSONPayload
}

// newInitTUIModel constructs the walk model with the project name defaulted
// to filepath.Base(cwd) per SKETCH §9.3 ("default = filepath.Base(cwd);
// user can edit"). The textinput is pre-populated with the default so an
// Enter on the first frame accepts it verbatim.
func newInitTUIModel(cwd string) initTUIModel {
	def := filepath.Base(cwd)
	ti := textinput.New()
	ti.Prompt = "name: "
	ti.Placeholder = def
	ti.CharLimit = 120
	ti.SetValue(def)
	ti.CursorEnd()
	ti.Focus()
	return initTUIModel{
		step:        initTUIStepName,
		nameInput:   ti,
		groupCursor: 0,
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

// Update advances the walk one event at a time. The keymap is intentionally
// small: Enter / Esc on both steps, plus Up/Down (or j/k) on the group
// picker. Any other keypress on the name step is forwarded to the
// textinput; on the group step it is ignored.
func (m initTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		// Non-key messages (WindowSize, etc.) pass through unchanged.
		return m, nil
	}

	switch m.step {
	case initTUIStepName:
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
		switch {
		case key.Code == tea.KeyEsc:
			m.step = initTUIStepCancelled
			return m, tea.Quit
		case key.Code == tea.KeyUp || key.String() == "k":
			m.groupCursor = prevEnabledGroupRow(m.groupCursor)
			return m, nil
		case key.Code == tea.KeyDown || key.String() == "j":
			m.groupCursor = nextEnabledGroupRow(m.groupCursor)
			return m, nil
		case key.Code == tea.KeyEnter:
			row := initTUIGroupRows[m.groupCursor]
			if row.Disabled {
				// Defense-in-depth: the cursor movement helpers already
				// skip disabled rows, but if the cursor somehow lands on
				// one (e.g. future row additions), Enter is a no-op
				// rather than accepting a disabled selection.
				return m, nil
			}
			m.finalPayload.Group = row.Name
			m.finalPayload.MCP = false // TUI default per droplet acceptance.
			m.step = initTUIStepDone
			return m, tea.Quit
		default:
			return m, nil
		}

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
		b.WriteString("Agent group (↑/↓ to move, enter to confirm, esc to cancel)\n\n")
		for i, row := range initTUIGroupRows {
			marker := "  "
			if i == m.groupCursor {
				marker = "> "
			}
			label := row.Name
			if row.Disabled {
				label += " (disabled — reserved for GDD)"
			}
			b.WriteString(marker)
			b.WriteString(label)
			b.WriteString("\n")
		}
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
// zero value (and the Group field will be empty).
func (m initTUIModel) Payload() initJSONPayload {
	return m.finalPayload
}

// nextEnabledGroupRow returns the cursor position one row down from cur,
// skipping any disabled rows. If every subsequent row is disabled, the
// cursor stays put — disabled rows are NEVER landable.
func nextEnabledGroupRow(cur int) int {
	for i := cur + 1; i < len(initTUIGroupRows); i++ {
		if !initTUIGroupRows[i].Disabled {
			return i
		}
	}
	return cur
}

// prevEnabledGroupRow returns the cursor position one row up from cur,
// skipping any disabled rows. If every prior row is disabled, the cursor
// stays put.
func prevEnabledGroupRow(cur int) int {
	for i := cur - 1; i >= 0; i-- {
		if !initTUIGroupRows[i].Disabled {
			return i
		}
	}
	return cur
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

// runInitPipeline is the shared post-input file-copy pipeline both
// runInitTUI and runInitJSON invoke. It resolves the destination
// directory (cwd), runs the three idempotent copy steps in order, and
// hands forward to the D6 `.mcp.json` registration stub. Re-run safety
// is mandatory: every file write is gated by an `os.Stat` pre-check so
// existing files are SKIPPED, never overwritten.
//
// `destDir` is derived from `os.Getwd()` (NOT from `opts.appName` /
// `opts.homeDir`) — D7.5's pointer-vs-value fix on rootCommandOptions
// does not surface in D5 because the destination is cwd-relative. The
// project-DB record creation in D7 uses path resolution via
// `platform.DefaultPathsWithOptions(opts)` and so depends on the D7.5
// fix; D5 is decoupled from it.
func runInitPipeline(stdout io.Writer, opts rootCommandOptions, payload initJSONPayload) error {
	_ = stdout
	_ = opts

	destDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("till init: resolve cwd: %w", err)
	}

	if _, _, err := copyAgentFiles(destDir, payload.Group); err != nil {
		return fmt.Errorf("till init: copy agent files: %w", err)
	}
	if _, _, err := copyAgentsTOML(destDir); err != nil {
		return fmt.Errorf("till init: copy agents.toml: %w", err)
	}
	if err := ensureGitignore(destDir); err != nil {
		return fmt.Errorf("till init: ensure .gitignore: %w", err)
	}

	// D6 wires the actual `.mcp.json` registration. Until then a
	// successful D5 file-copy hands forward to this stub so callers
	// (and tests) can confirm the pipeline ran through to the D6 seam
	// without short-circuiting earlier.
	return errors.New("till init: .mcp.json registration not yet wired (W2.D6)")
}

// agentFileInitPerm is the permission applied to every freshly copied
// agent .md, agents.toml, and .gitignore write. Matches the conventional
// 0o644 (user rw, group r, other r) the embedded fixtures themselves
// ship with under git.
const agentFileInitPerm os.FileMode = 0o644

// copyAgentFiles reads the embedded `internal/templates/builtin/agents/<group>/*.md`
// set via `templates.DefaultTemplateFS` and writes each entry to
// `<destDir>/.tillsyn/agents/*.md` FLAT (no group prefix). Each write
// uses `fsatomic.WriteFile` (write-temp-in-same-dir + rename). Existing
// destination files are SKIPPED, never overwritten — re-run safety.
//
// Returns `(added, skippedExisting, err)`. `added` counts files newly
// created; `skippedExisting` counts files that already existed at the
// destination and were left untouched. On error the partial-progress
// counts so far are returned alongside the wrapped error.
func copyAgentFiles(destDir, group string) (int, int, error) {
	if strings.TrimSpace(group) == "" {
		return 0, 0, errors.New("copyAgentFiles: group required")
	}

	srcDir := path.Join("builtin", "agents", group)
	entries, err := fs.ReadDir(templates.DefaultTemplateFS, srcDir)
	if err != nil {
		return 0, 0, fmt.Errorf("read embedded %q: %w", srcDir, err)
	}

	agentsDir := filepath.Join(destDir, ".tillsyn", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return 0, 0, fmt.Errorf("mkdir %q: %w", agentsDir, err)
	}

	added, skipped := 0, 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		// FLAT copy: drop the group prefix. The destination basename is
		// the source basename, dropped directly under `.tillsyn/agents/`.
		target := filepath.Join(agentsDir, entry.Name())
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
// a parsed `initJSONPayload`. Returns a wrapped error pointing at the
// first failed invariant; `Name` and `Group` are required, and `Group`
// must be one of `allowedInitGroups` (reserved groups like `till-gdd`
// surface a tailored "reserved" error).
func validateInitPayload(p initJSONPayload) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("till init: name required")
	}
	if strings.TrimSpace(p.Group) == "" {
		return errors.New("till init: group required")
	}
	if reserved, ok := reservedInitGroups[p.Group]; ok {
		return fmt.Errorf("till init: group must be one of %v; %q is reserved", allowedInitGroups, reserved)
	}
	for _, allowed := range allowedInitGroups {
		if p.Group == allowed {
			return nil
		}
	}
	return fmt.Errorf("till init: group must be one of %v; got %q", allowedInitGroups, p.Group)
}
