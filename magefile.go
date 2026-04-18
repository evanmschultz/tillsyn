//go:build mage

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/evanmschultz/laslig"
	"github.com/evanmschultz/laslig/gotestout"
)

// Default runs the canonical full gate when `mage` is invoked without a target.
var Default = CI

// Aliases preserves the familiar hyphenated task names while keeping the visible target list small.
var Aliases = map[string]interface{}{
	"check":              CI,
	"dev":                Dev,
	"test-golden":        TestGolden,
	"test-golden-update": TestGoldenUpdate,
	"test-pkg":           TestPkg,
	"test-func":          TestFunc,
	"fmt":                Format,
}

// coverageThreshold is the minimum allowed statement coverage for each and all packages.
const coverageThreshold = 70.0

// localBuildVCSFlag disables VCS stamping for local bare-worktree commands.
const localBuildVCSFlag = "-buildvcs=false"

// coverageLinePattern extracts package names and percentages from successful `go test -cover` output lines.
var coverageLinePattern = regexp.MustCompile(`^ok\s+(\S+)(?:\s+\S+)?\s+coverage:\s+([0-9.]+)% of statements(?: in ./\.\.\.)?$`)

// TestPkg runs tests for one package path, directory, or pattern.
func TestPkg(pkg string) error {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return errors.New("package path is required")
	}
	info, err := os.Stat(pkg)
	if err == nil && info.IsDir() {
		dirArg := normalizedGoDirArg(pkg)
		matches, globErr := filepath.Glob(filepath.Join(pkg, "*.go"))
		if globErr != nil {
			return fmt.Errorf("glob package dir %q: %w", pkg, globErr)
		}
		if len(matches) > 0 {
			return runGoTest(dirArg)
		}
		return runGoTest(dirArg + "/...")
	}
	return runGoTest(pkg)
}

// TestFunc runs one named test function scoped to one package path, directory, or pattern with race detection and no result caching.
func TestFunc(pkg, testName string) error {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return errors.New("package path is required")
	}
	testName = strings.TrimSpace(testName)
	if testName == "" {
		return errors.New("test name is required")
	}
	runPattern := "^" + testName + "$"
	info, err := os.Stat(pkg)
	if err == nil && info.IsDir() {
		dirArg := normalizedGoDirArg(pkg)
		matches, globErr := filepath.Glob(filepath.Join(pkg, "*.go"))
		if globErr != nil {
			return fmt.Errorf("glob package dir %q: %w", pkg, globErr)
		}
		if len(matches) > 0 {
			return runGoTest(dirArg, "-run", runPattern, "-race", "-count=1")
		}
		return runGoTest(dirArg+"/...", "-run", runPattern, "-race", "-count=1")
	}
	return runGoTest(pkg, "-run", runPattern, "-race", "-count=1")
}

// TestGolden runs the focused golden-file suite for the TUI package.
func TestGolden() error {
	return runGoTest("./internal/tui", "-run", "Golden")
}

// TestGoldenUpdate refreshes golden fixtures and reruns the focused TUI golden suite.
func TestGoldenUpdate() error {
	return runGoTest("./internal/tui", "-run", "Golden", "-update")
}

// Build compiles the local till binary at `./till`.
func Build() error {
	printer := newMagePrinter()
	return runCommandWithProgress(printer, "Building till from ./cmd/till", "Built till from ./cmd/till", "go", "build", localBuildVCSFlag, "-o", "./till", "./cmd/till")
}

// Run executes till directly from source.
func Run() error {
	return runCommand("go", "run", localBuildVCSFlag, "./cmd/till")
}

// Dev executes till from source against the repo-local dev runtime home.
func Dev() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("working directory: %w", err)
	}
	return runCommandWithEnv(map[string]string{
		"TILL_DEV_MODE": "true",
		"TILL_HOME":     filepath.Join(wd, ".tillsyn"),
	}, "go", "run", localBuildVCSFlag, "./cmd/till")
}

// Install builds till from the current working tree and drops the binary at
// $HOME/.local/bin/till so the stable CLI stays on PATH while runtime data
// remains under $HOME/.tillsyn.
//
// Dev-only dogfood target. Orchestrator and every subagent (builder, QA,
// research, planning) MUST NOT invoke this — see CLAUDE.md "Build Verification"
// rule #3. Only the human dev runs `mage install`, manually, from a checkout
// of the revision they want promoted. To install an older revision, run
// `git checkout <sha>` in your shell first, then `mage install`.
func Install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	installDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create install dir %q: %w", installDir, err)
	}
	installedPath := filepath.Join(installDir, "till")
	return runCommand("go", "build", localBuildVCSFlag, "-o", installedPath, "./cmd/till")
}

// CI runs the canonical full gate.
func CI() error {
	printer := newMagePrinter()
	for _, stage := range []struct {
		title string
		run   func() error
	}{
		{title: "Sources", run: verifySources},
		{title: "Formatting", run: formatCheck},
		{title: "Coverage", run: coverage},
		{title: "Build", run: Build},
	} {
		if err := runStage(printer, stage.title, stage.run); err != nil {
			return err
		}
	}
	return nil
}

// newMagePrinter returns the default laslig printer for Mage output.
func newMagePrinter() *laslig.Printer {
	return laslig.New(os.Stdout, mageOutputPolicy())
}

// mageOutputPolicy resolves the laslig policy used for Mage output.
func mageOutputPolicy() laslig.Policy {
	style := laslig.StyleAuto
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		style = laslig.StyleNever
	}
	return laslig.Policy{
		Format: laslig.FormatAuto,
		Style:  style,
	}
}

// runStage renders one stage heading and executes the corresponding step.
func runStage(printer *laslig.Printer, title string, fn func() error) error {
	if err := printer.Section(title); err != nil {
		return fmt.Errorf("render %s stage: %w", title, err)
	}
	return fn()
}

// verifySources ensures the required automation and CLI entrypoint sources are still tracked.
func verifySources() error {
	printer := newMagePrinter()
	_, err := captureCommandWithProgress(printer, "Verifying tracked sources", "Verified tracked sources", "git", "ls-files", "--error-unmatch", "magefile.go", "cmd/till/main.go", "cmd/till/main_test.go")
	return err
}

// Format rewrites Go sources with `go tool gofumpt -w`. With path == "." (or empty) it targets every tracked Go file; otherwise path is treated as a single file or directory to rewrite.
func Format(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		files, err := trackedGoFiles()
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}
		return runGofumptWrite(files)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("gofumpt path %q: %w", path, err)
	}
	return runGofumptWrite([]string{path})
}

// formatCheck reports tracked Go files that still need gofumpt formatting.
func formatCheck() error {
	files, err := trackedGoFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	out, err := runGofumptList(files)
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return fmt.Errorf("gofumpt required for:\n%s", out)
}

// runGofumptList invokes `go tool gofumpt -l` against paths and returns the captured stdout.
// Stdout-only capture keeps `go tool` download chatter on a cold `$GOMODCACHE` out of the filename
// list parsed by `formatCheck`; stderr is routed to a separate buffer that surfaces on error only.
func runGofumptList(paths []string) (string, error) {
	printer := newMagePrinter()
	args := gofumptArgs("-l", paths)
	return captureStdoutWithProgress(printer, "Checking Go formatting", "Checked Go formatting", "go", args...)
}

// runGofumptWrite invokes `go tool gofumpt -w` against paths and streams tool output.
func runGofumptWrite(paths []string) error {
	printer := newMagePrinter()
	args := gofumptArgs("-w", paths)
	return runCommandWithProgress(printer, "Formatting Go sources", "Formatted Go sources", "go", args...)
}

// gofumptArgs assembles the `go tool gofumpt <mode> <paths...>` argv used by every formatter invocation.
func gofumptArgs(mode string, paths []string) []string {
	args := make([]string, 0, 3+len(paths))
	args = append(args, "tool", "gofumpt", mode)
	args = append(args, paths...)
	return args
}

// coverage runs the full suite with coverage enabled and enforces the per-package floor.
func coverage() error {
	raw, summary, err := runGoTestCapture("-cover", "./...")
	if err != nil {
		return err
	}
	if summary.HasFailures() {
		return errors.New("go test -cover ./...: test summary reported failures")
	}

	printer := newMagePrinter()
	rows, belowThreshold, err := coverageRows(raw)
	if err != nil {
		return err
	}
	if err := printer.Table(laslig.Table{
		Header:  []string{"package", "cover"},
		Rows:    rows,
		Caption: fmt.Sprintf("Minimum package coverage: %.1f%%.", coverageThreshold),
	}); err != nil {
		return fmt.Errorf("write coverage table: %w", err)
	}
	if len(belowThreshold) > 0 {
		if err := printer.Notice(laslig.Notice{
			Level:  laslig.NoticeErrorLevel,
			Title:  "Coverage threshold not met",
			Body:   fmt.Sprintf("Each package must stay at or above %.1f%% coverage.", coverageThreshold),
			Detail: []string{strings.Join(belowThreshold, ", ")},
		}); err != nil {
			return fmt.Errorf("write coverage notice: %w", err)
		}
		return fmt.Errorf("coverage below %.1f%%: %s", coverageThreshold, strings.Join(belowThreshold, ", "))
	}
	if err := printer.Notice(laslig.Notice{
		Level: laslig.NoticeSuccessLevel,
		Title: "Coverage threshold met",
		Body:  fmt.Sprintf("All packages are at or above %.1f%% coverage.", coverageThreshold),
	}); err != nil {
		return fmt.Errorf("write coverage success: %w", err)
	}
	return nil
}

// trackedGoFiles returns all tracked Go files in stable git order.
// Uses stdout-only capture so any git stderr chatter (hints, warnings) cannot leak into the
// filename list consumed by downstream callers such as `formatCheck`.
func trackedGoFiles() ([]string, error) {
	printer := newMagePrinter()
	out, err := captureStdoutWithProgress(printer, "Listing tracked Go files", "Listed tracked Go files", "git", "ls-files", "*.go")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}
	return files, nil
}

// normalizedGoDirArg preserves a caller's local `./` package semantics when normalizing directory paths.
func normalizedGoDirArg(path string) string {
	normalized := filepath.ToSlash(filepath.Clean(path))
	if strings.HasPrefix(filepath.ToSlash(path), "./") && !strings.HasPrefix(normalized, "./") {
		return "./" + normalized
	}
	return normalized
}

// coverageRows extracts package coverage rows and threshold failures from one `go test -json -cover` stream.
func coverageRows(raw string) ([][]string, []string, error) {
	events, err := gotestout.Parse(strings.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("parse go test event stream: %w", err)
	}

	rows := make([][]string, 0)
	var belowThreshold []string
	for _, event := range events {
		if event.Action != gotestout.ActionOutput {
			continue
		}
		match := coverageLinePattern.FindStringSubmatch(strings.TrimSpace(event.Output))
		if match == nil {
			continue
		}
		percent, parseErr := strconv.ParseFloat(match[2], 64)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("parse coverage for %q: %w", match[1], parseErr)
		}
		rows = append(rows, []string{match[1], fmt.Sprintf("%.1f%%", percent)})
		if percent < coverageThreshold {
			belowThreshold = append(belowThreshold, fmt.Sprintf("%s %.1f%%", match[1], percent))
		}
	}
	if len(rows) == 0 {
		return nil, nil, errors.New("no coverage rows were parsed from go test output")
	}
	return rows, belowThreshold, nil
}

// runCommand executes one command and streams its stdout/stderr to the current terminal.
func runCommand(name string, args ...string) error {
	return runCommandWithEnv(nil, name, args...)
}

// runCommandWithEnv executes one command with optional environment overrides.
func runCommandWithEnv(overrides map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if len(overrides) > 0 {
		env := os.Environ()
		for key, value := range overrides {
			env = append(env, key+"="+value)
		}
		cmd.Env = env
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// runCommandInDir executes one command with cmd.Dir set to dir and streams
// stdout/stderr to the current terminal. Used by the install flow so the
// `go build` step runs inside the temp detached worktree rather than the
// caller's (possibly dirty) working directory.
func runCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s (in %s): %w", name, strings.Join(args, " "), dir, err)
	}
	return nil
}

// captureCommand runs one command and returns its combined stdout/stderr.
// Use this only when the caller cares about the command's exit code and treats output as opaque
// diagnostic text; for any gate that parses the captured stream, use captureStdoutCommand instead
// so toolchain chatter on stderr cannot poison the semantic output.
func captureCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), wrapCommandError(name, args, err)
}

// captureStdoutCommand runs one command, returns its stdout only, and routes stderr to a separate
// buffer. On error, the stderr contents are appended to the wrapped error message so failure
// diagnostics are preserved; on success, stderr is discarded. This is the helper every gate that
// parses semantic output (filenames, JSON, counts) must use — it keeps `go tool` download banners,
// git hints, and other stderr-only chatter out of the stream callers depend on.
func captureStdoutCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.String(), wrapCommandErrorWithStderr(name, args, err, stderr.String())
	}
	return stdout.String(), nil
}

// wrapCommandError annotates one command failure while preserving the nil success case.
func wrapCommandError(name string, args []string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
}

// wrapCommandErrorWithStderr annotates one command failure and appends captured stderr so callers
// using stdout-only capture still surface meaningful diagnostics when the command fails.
func wrapCommandErrorWithStderr(name string, args []string, err error, stderr string) error {
	if err == nil {
		return nil
	}
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return fmt.Errorf("%s %s: %w\nstderr:\n%s", name, strings.Join(args, " "), err, trimmed)
}

// runGoTest renders one `go test -json` invocation through laslig/gotestout.
func runGoTest(args ...string) error {
	_, summary, err := runGoTestCapture(args...)
	if err != nil {
		return err
	}
	if summary.HasFailures() {
		return fmt.Errorf("go test %s: test summary reported failures", strings.Join(args, " "))
	}
	return nil
}

// runGoTestCapture renders one `go test -json` stream and returns the captured raw JSON plus summary counts.
func runGoTestCapture(args ...string) (string, gotestout.Summary, error) {
	cmdArgs := append([]string{"test", localBuildVCSFlag, "-json"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", gotestout.Summary{}, fmt.Errorf("create go test stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return "", gotestout.Summary{}, fmt.Errorf("start go test: %w", err)
	}

	printer := newMagePrinter()
	spinner := startMageSpinner(printer, "Running go test "+strings.Join(args, " "))

	var raw bytes.Buffer
	stream := &spinnerHandoffReader{
		reader: io.TeeReader(stdout, &raw),
		onFirstOutput: func() {
			stopMageSpinner(spinner, "Test stream detected", laslig.NoticeSuccessLevel)
		},
	}
	summary, renderErr := gotestout.Render(os.Stdout, stream, gotestout.Options{
		Policy: mageOutputPolicy(),
		View:   gotestout.ViewCompact,
	})
	waitErr := cmd.Wait()

	if renderErr != nil {
		stopMageSpinner(spinner, "Rendering test output failed", laslig.NoticeErrorLevel)
		return "", gotestout.Summary{}, fmt.Errorf("render go test output: %w", renderErr)
	}
	if waitErr != nil {
		stopMageSpinner(spinner, "go test failed", laslig.NoticeErrorLevel)
		return raw.String(), summary, fmt.Errorf("go %s: %w", strings.Join(cmdArgs, " "), waitErr)
	}
	stopMageSpinner(spinner, "Tests complete", laslig.NoticeSuccessLevel)
	return raw.String(), summary, nil
}

// spinnerHandoffReader stops one spinner when the wrapped stream emits its first payload bytes.
type spinnerHandoffReader struct {
	reader        io.Reader
	onFirstOutput func()
	once          sync.Once
}

// Read proxies one read and triggers the handoff callback on the first non-empty chunk.
func (r *spinnerHandoffReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.onFirstOutput != nil {
		r.once.Do(r.onFirstOutput)
	}
	return n, err
}

// runCommandWithProgress renders one transient spinner while a command stays quiet.
func runCommandWithProgress(printer *laslig.Printer, startText, successText, name string, args ...string) error {
	spinner := startMageSpinner(printer, startText)
	err := runCommand(name, args...)
	if err != nil {
		stopMageSpinner(spinner, startText+" failed", laslig.NoticeErrorLevel)
		return err
	}
	stopMageSpinner(spinner, successText, laslig.NoticeSuccessLevel)
	return nil
}

// runCommandInDirWithProgress renders one transient spinner while a command
// executes with cmd.Dir set to dir. Used by the install build step so the
// Go toolchain resolves modules against the temp worktree rather than the
// caller's live working directory.
func runCommandInDirWithProgress(printer *laslig.Printer, dir, startText, successText, name string, args ...string) error {
	spinner := startMageSpinner(printer, startText)
	err := runCommandInDir(dir, name, args...)
	if err != nil {
		stopMageSpinner(spinner, startText+" failed", laslig.NoticeErrorLevel)
		return err
	}
	stopMageSpinner(spinner, successText, laslig.NoticeSuccessLevel)
	return nil
}

// captureCommandWithProgress renders one transient spinner while a captured command stays quiet.
// Returns combined stdout+stderr; prefer captureStdoutWithProgress when the output is parsed.
func captureCommandWithProgress(printer *laslig.Printer, startText, successText, name string, args ...string) (string, error) {
	spinner := startMageSpinner(printer, startText)
	out, err := captureCommand(name, args...)
	if err != nil {
		stopMageSpinner(spinner, startText+" failed", laslig.NoticeErrorLevel)
		return "", err
	}
	stopMageSpinner(spinner, successText, laslig.NoticeSuccessLevel)
	return out, nil
}

// captureStdoutWithProgress renders one transient spinner while capturing stdout only; stderr is
// kept out of the returned stream and surfaces only when the command fails. Use this for every
// gate whose captured output is parsed (filename lists, JSON, counts) so toolchain chatter cannot
// pollute the decision stream.
func captureStdoutWithProgress(printer *laslig.Printer, startText, successText, name string, args ...string) (string, error) {
	spinner := startMageSpinner(printer, startText)
	out, err := captureStdoutCommand(name, args...)
	if err != nil {
		stopMageSpinner(spinner, startText+" failed", laslig.NoticeErrorLevel)
		return "", err
	}
	stopMageSpinner(spinner, successText, laslig.NoticeSuccessLevel)
	return out, nil
}

// startMageSpinner starts one transient spinner when printer output supports it.
func startMageSpinner(printer *laslig.Printer, text string) *laslig.Spinner {
	if printer == nil {
		return nil
	}
	spinner := printer.NewSpinner()
	if err := spinner.Start(text); err != nil {
		return nil
	}
	return spinner
}

// stopMageSpinner stops one transient spinner when it was started successfully.
func stopMageSpinner(spinner *laslig.Spinner, message string, level laslig.NoticeLevel) {
	if spinner == nil {
		return
	}
	_ = spinner.Stop(message, level)
}
