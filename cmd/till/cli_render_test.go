package main

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/laslig"
)

// TestCLIStylePolicyDefault verifies ordinary CLI output follows laslig's automatic style detection by default.
func TestCLIStylePolicyDefault(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if got := cliStylePolicy(); got != laslig.StyleAuto {
		t.Fatalf("cliStylePolicy() = %q, want %q", got, laslig.StyleAuto)
	}
}

// TestCLIStylePolicyHonorsNoColor verifies ordinary CLI output disables ANSI styling when NO_COLOR is set.
func TestCLIStylePolicyHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := cliStylePolicy(); got != laslig.StyleNever {
		t.Fatalf("cliStylePolicy() = %q, want %q", got, laslig.StyleNever)
	}
}

// TestWithCLIProgressSkipsNonStyledWriters verifies progress rendering stays off for non-terminal stderr.
func TestWithCLIProgressSkipsNonStyledWriters(t *testing.T) {
	origDelay := cliProgressDelay
	cliProgressDelay = 0
	t.Cleanup(func() { cliProgressDelay = origDelay })

	var stderr strings.Builder
	called := false
	err := withCLIProgress(&stderr, "Listing projects", func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("withCLIProgress() error = %v", err)
	}
	if !called {
		t.Fatal("expected wrapped function to run")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr progress output for non-styled writer, got %q", stderr.String())
	}
}

// TestWithCLIProgressWritesStatusLinesWhenForced verifies tests can force progress rendering through a builder.
func TestWithCLIProgressWritesStatusLinesWhenForced(t *testing.T) {
	origSupport := supportsStyledOutputFunc
	supportsStyledOutputFunc = func(io.Writer) bool { return true }
	t.Cleanup(func() { supportsStyledOutputFunc = origSupport })

	origDelay := cliProgressDelay
	cliProgressDelay = 0
	t.Cleanup(func() { cliProgressDelay = origDelay })

	var stderr strings.Builder
	err := withCLIProgress(&stderr, "Listing projects", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("withCLIProgress() error = %v", err)
	}
	rendered := strings.ToLower(stderr.String())
	for _, want := range []string{"listing projects", "complete"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in progress output, got %q", want, rendered)
		}
	}
}
