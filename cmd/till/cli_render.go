package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/evanmschultz/laslig"
)

// cliProgressDelay keeps spinners off very fast one-shot commands.
var cliProgressDelay = 200 * time.Millisecond

// newCLIPrinter returns the default laslig printer for ordinary CLI result output.
func newCLIPrinter(stdout io.Writer) *laslig.Printer {
	return laslig.New(stdout, laslig.Policy{
		Format: laslig.FormatAuto,
		Style:  cliStylePolicy(),
	})
}

// newStyledCLIPrinter returns one forced-human laslig printer for explicitly styled terminal output.
func newStyledCLIPrinter(stdout io.Writer) *laslig.Printer {
	return laslig.New(stdout, laslig.Policy{
		Format: laslig.FormatHuman,
		Style:  laslig.StyleAlways,
	})
}

// withCLIProgress renders one transient spinner on stderr while a styled human command stays quiet.
func withCLIProgress(stderr io.Writer, label string, fn func() error) error {
	label = strings.TrimSpace(label)
	if label == "" || stderr == nil || !supportsStyledOutputFunc(stderr) {
		return fn()
	}

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- fn()
	}()

	timer := time.NewTimer(cliProgressDelay)
	defer timer.Stop()

	var spinner *laslig.Spinner
	for {
		select {
		case err := <-resultCh:
			if spinner != nil {
				level := laslig.NoticeSuccessLevel
				message := label + " complete"
				if err != nil {
					level = laslig.NoticeErrorLevel
					message = label + " failed"
				}
				_ = spinner.Stop(message, level)
			}
			return err
		case <-timer.C:
			spinner = newCLIPrinter(stderr).NewSpinner()
			if err := spinner.Start(label); err != nil {
				spinner = nil
			}
		}
	}
}

// commandProgressLabel returns one human-readable progress label for a one-shot CLI command.
func commandProgressLabel(command string) string {
	switch strings.TrimSpace(command) {
	case "":
		return ""
	case "auth.issue-session":
		return "Issuing auth session"
	case "auth.request.create":
		return "Creating auth request"
	case "auth.request.list":
		return "Listing auth requests"
	case "auth.request.show":
		return "Loading auth request"
	case "auth.request.approve":
		return "Approving auth request"
	case "auth.request.deny":
		return "Denying auth request"
	case "auth.request.cancel":
		return "Canceling auth request"
	case "auth.session.list":
		return "Listing auth sessions"
	case "auth.session.validate":
		return "Validating auth session"
	case "auth.session.revoke", "auth.revoke-session":
		return "Revoking auth session"
	case "project.list":
		return "Listing projects"
	case "project.create":
		return "Creating project"
	case "project.show":
		return "Loading project"
	case "project.discover":
		return "Discovering project readiness"
	case "embeddings.status":
		return "Inspecting embeddings status"
	case "embeddings.reindex":
		return "Running embeddings reindex"
	case "capture-state":
		return "Capturing state"
	case "kind.list":
		return "Listing kinds"
	case "kind.upsert":
		return "Upserting kind"
	case "kind.allowlist.list":
		return "Loading kind allowlist"
	case "kind.allowlist.set":
		return "Updating kind allowlist"
	case "template.library.list":
		return "Listing template libraries"
	case "template.library.show":
		return "Loading template library"
	case "template.library.upsert":
		return "Upserting template library"
	case "template.project.bind":
		return "Binding project template"
	case "template.project.binding":
		return "Loading project template binding"
	case "template.contract.show":
		return "Loading template contract"
	case "lease.list":
		return "Listing leases"
	case "lease.issue":
		return "Issuing lease"
	case "lease.heartbeat":
		return "Refreshing lease heartbeat"
	case "lease.renew":
		return "Renewing lease"
	case "lease.revoke":
		return "Revoking lease"
	case "lease.revoke-all":
		return "Revoking leases"
	case "handoff.create":
		return "Creating handoff"
	case "handoff.get":
		return "Loading handoff"
	case "handoff.list":
		return "Listing handoffs"
	case "handoff.update":
		return "Updating handoff"
	case "export":
		return "Exporting snapshot"
	case "import":
		return "Importing snapshot"
	default:
		command = strings.ReplaceAll(command, ".", " ")
		command = strings.ReplaceAll(command, "-", " ")
		return "Running " + command
	}
}

// cliStylePolicy resolves the laslig style policy for ordinary CLI result output.
func cliStylePolicy() laslig.StylePolicy {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return laslig.StyleNever
	}
	return laslig.StyleAuto
}

// writeCLITable renders one titled table through laslig.
func writeCLITable(stdout io.Writer, title string, header []string, rows [][]string, empty string) error {
	return writeCLITableWithPrinter(newCLIPrinter(stdout), title, header, rows, empty)
}

// writeCLITableWithPrinter renders one titled table through a caller-provided laslig printer.
func writeCLITableWithPrinter(printer *laslig.Printer, title string, header []string, rows [][]string, empty string) error {
	if err := printer.Table(laslig.Table{
		Title:  strings.TrimSpace(title),
		Header: append([]string(nil), header...),
		Rows:   append([][]string(nil), rows...),
		Empty:  strings.TrimSpace(empty),
	}); err != nil {
		return fmt.Errorf("write %s table: %w", strings.ToLower(strings.TrimSpace(title)), err)
	}
	return nil
}

// writeCLIKV renders one titled key/value block through laslig.
func writeCLIKV(stdout io.Writer, title string, rows [][2]string) error {
	return writeCLIKVWithPrinter(newCLIPrinter(stdout), title, rows)
}

// writeCLIKVWithPrinter renders one titled key/value block through a caller-provided laslig printer.
func writeCLIKVWithPrinter(printer *laslig.Printer, title string, rows [][2]string) error {
	if err := printer.KV(laslig.KV{
		Title: strings.TrimSpace(title),
		Pairs: cliFields(rows),
	}); err != nil {
		return fmt.Errorf("write %s kv block: %w", strings.ToLower(strings.TrimSpace(title)), err)
	}
	return nil
}

// writeCLIPanel renders one boxed guidance block through laslig.
func writeCLIPanel(stdout io.Writer, title, body, footer string) error {
	return writeCLIPanelWithPrinter(newCLIPrinter(stdout), title, body, footer)
}

// writeCLIPanelWithPrinter renders one boxed guidance block through a caller-provided laslig printer.
func writeCLIPanelWithPrinter(printer *laslig.Printer, title, body, footer string) error {
	if err := printer.Panel(laslig.Panel{
		Title:  strings.TrimSpace(title),
		Body:   firstNonEmptyTrimmed(body, "-"),
		Footer: strings.TrimSpace(footer),
	}); err != nil {
		return fmt.Errorf("write %s panel: %w", strings.ToLower(strings.TrimSpace(title)), err)
	}
	return nil
}

// cliFields converts key/value rows into laslig fields with stable empty-value fallbacks.
func cliFields(rows [][2]string) []laslig.Field {
	fields := make([]laslig.Field, 0, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row[0])
		if label == "" {
			continue
		}
		fields = append(fields, laslig.Field{
			Label: label,
			Value: firstNonEmptyTrimmed(row[1], "-"),
		})
	}
	return fields
}
