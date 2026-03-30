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

	"github.com/evanmschultz/laslig"
	"github.com/evanmschultz/laslig/gotestout"
)

// Default runs the canonical full gate when `mage` is invoked without a target.
var Default = CI

// Aliases preserves the familiar hyphenated task names while keeping the visible target list small.
var Aliases = map[string]interface{}{
	"check":              CI,
	"test-golden":        TestGolden,
	"test-golden-update": TestGoldenUpdate,
	"test-pkg":           TestPkg,
}

// coverageThreshold is the minimum allowed statement coverage for each package.
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
	if err := printer.StatusLine(laslig.StatusLine{
		Level:  laslig.NoticeInfoLevel,
		Text:   "Building till",
		Detail: "./cmd/till",
	}); err != nil {
		return fmt.Errorf("write build start: %w", err)
	}
	if err := runCommand("go", "build", localBuildVCSFlag, "-o", "./till", "./cmd/till"); err != nil {
		return err
	}
	if err := printer.StatusLine(laslig.StatusLine{
		Level:  laslig.NoticeSuccessLevel,
		Text:   "Built till",
		Detail: "./cmd/till",
	}); err != nil {
		return fmt.Errorf("write build success: %w", err)
	}
	return nil
}

// Run executes till directly from source.
func Run() error {
	return runCommand("go", "run", localBuildVCSFlag, "./cmd/till")
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
	return runCommand("git", "ls-files", "--error-unmatch", "magefile.go", "cmd/till/main.go", "cmd/till/main_test.go")
}

// formatCheck reports tracked Go files that still need gofmt.
func formatCheck() error {
	files, err := trackedGoFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	out, err := captureCommand("gofmt", append([]string{"-l"}, files...)...)
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return fmt.Errorf("gofmt required for:\n%s", out)
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
func trackedGoFiles() ([]string, error) {
	out, err := captureCommand("git", "ls-files", "*.go")
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
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// captureCommand runs one command and returns its combined stdout/stderr.
func captureCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), wrapCommandError(name, args, err)
}

// wrapCommandError annotates one command failure while preserving the nil success case.
func wrapCommandError(name string, args []string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
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

	var raw bytes.Buffer
	summary, renderErr := gotestout.Render(os.Stdout, io.TeeReader(stdout, &raw), gotestout.Options{
		Policy: mageOutputPolicy(),
		View:   gotestout.ViewCompact,
	})
	waitErr := cmd.Wait()

	if renderErr != nil {
		return "", gotestout.Summary{}, fmt.Errorf("render go test output: %w", renderErr)
	}
	if waitErr != nil {
		return raw.String(), summary, fmt.Errorf("go %s: %w", strings.Join(cmdArgs, " "), waitErr)
	}
	return raw.String(), summary, nil
}
