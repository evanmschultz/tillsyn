//go:build ignore

// Package main is a Wave 2.8 test-helper agent binary. It is NOT compiled
// into the dispatcher package — the //go:build ignore tag excludes it from
// the main build — and is instead compiled into a one-shot tmpfile by
// monitor_test.go's setup helper via exec.Command("go", "build", ...).
//
// This is the documented "never raw `go`" carve-out for Wave 2.8: a tiny
// fake agent the monitor can exercise against real os/exec semantics
// (exit-code propagation, signal kills, sleep durations) without depending
// on the production claude binary being on PATH. See
// internal/app/dispatcher/monitor.go's package overview and
// WAVE_2_PLAN.md §2.8 Q5 for the rationale.
//
// Mode is selected via the first command-line argument:
//
//	exit0           — print "ok" to stdout, exit 0.
//	exit1           — print "err" to stderr, exit 1.
//	hang            — sleep up to 30s; the test kills it via cmd.Process.Kill.
//	sleep <millis>  — sleep <millis> milliseconds, exit 0 (used by the
//	                  duration-tracking test).
//
// Defaults to exit0 if no mode argument is supplied. Unrecognized modes
// print a usage summary to stderr and exit 2 so a typo in the test fixture
// fails loudly rather than silently masquerading as a clean exit.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	mode := "exit0"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	switch mode {
	case "exit0":
		fmt.Println("ok")
		os.Exit(0)
	case "exit1":
		fmt.Fprintln(os.Stderr, "err")
		os.Exit(1)
	case "hang":
		// Sleep an absurdly long time so the test always wins the race
		// between cmd.Process.Kill and a natural exit.
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "sleep":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "fakeagent: sleep mode requires <millis> arg")
			os.Exit(2)
		}
		ms, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "fakeagent: invalid sleep millis %q: %v\n", os.Args[2], err)
			os.Exit(2)
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "fakeagent: unknown mode %q (want: exit0|exit1|hang|sleep)\n", mode)
		os.Exit(2)
	}
}
