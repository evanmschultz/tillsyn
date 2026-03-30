package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/evanmschultz/laslig"
)

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
