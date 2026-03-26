//go:build !windows

package main

import (
	"os"

	charmLog "github.com/charmbracelet/log"
	xtermios "github.com/charmbracelet/x/termios"
	"golang.org/x/term"
)

// withInterruptEchoSuppressed runs one function with echoed control characters disabled on the active stdin terminal.
//
// This keeps Ctrl-C from rendering as `^C` on long-running daemon-style commands
// while preserving the normal interrupt signal behavior. If stdin is not a real
// terminal or the terminal state cannot be adjusted safely, the function runs
// unchanged.
func withInterruptEchoSuppressed(runFn func() error) error {
	if runFn == nil {
		return nil
	}
	file := os.Stdin
	if file == nil {
		return runFn()
	}
	fd := int(file.Fd())
	if !term.IsTerminal(fd) {
		return runFn()
	}

	originalState, err := term.GetState(fd)
	if err != nil || originalState == nil {
		return runFn()
	}
	currentTermios, err := xtermios.GetTermios(fd)
	if err != nil || currentTermios == nil {
		return runFn()
	}
	if err := xtermios.SetTermios(
		fd,
		uint32(currentTermios.Ispeed),
		uint32(currentTermios.Ospeed),
		nil,
		nil,
		nil,
		nil,
		map[xtermios.L]bool{
			xtermios.ECHOCTL: false,
		},
	); err != nil {
		return runFn()
	}
	defer func() {
		if err := term.Restore(fd, originalState); err != nil {
			charmLog.Warn("restore terminal state failed after interrupt echo suppression", "err", err)
		}
	}()
	return runFn()
}
