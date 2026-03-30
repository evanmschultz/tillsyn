package main

import (
	"testing"

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
