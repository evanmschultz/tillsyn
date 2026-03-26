//go:build windows

package main

// withInterruptEchoSuppressed is a no-op on Windows.
func withInterruptEchoSuppressed(runFn func() error) error {
	if runFn == nil {
		return nil
	}
	return runFn()
}
