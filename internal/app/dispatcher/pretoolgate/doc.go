// Package pretoolgate defines the PreToolUse agent sandboxing gate contract
// and per-channel translation accessors.
//
// # Gate Contract
//
// The PreToolUse gate is applied at dispatch time to restrict a dispatched
// agent's access to files, directories, and shell commands. The contract is
// defined by GateSpec, a backend-agnostic struct that specifies:
//
//   - Edit: files granted write access (claude built-in and `claude -p` channels)
//   - WritableDirs: directories granted write access (codex exec channel)
//   - BashDeny: shell patterns denied (all channels)
//   - Network: whether network access is granted (codex only, today)
//
// The contract shape is specified in AGENT_SANDBOX_SPEC § 3 (consensus
// 2026-05-25, reproduced on ≥2 macOS runs). See AGENT_SANDBOX_SPEC.md for
// the full enforcement architecture and per-channel translation rules.
//
// # Per-Channel Accessors
//
// GateSpec provides per-channel translation accessors that project the contract
// onto each backend's enforcement form:
//
//   - BuiltinEdits(), BuiltinBashDeny(): claude built-in Agent tool gate hook
//   - CodexWritableDirs(), CodexBashDeny(): codex exec `--sandbox` + execpolicy
//
// These accessors are thin projections (they return the field directly or a
// derived value) with no business logic. Channel-specific argv assembly (e.g.,
// `--allowedTools`, execpolicy rules) remains in the adapters.
//
// # Nil vs Empty Distinction
//
// The Edit and WritableDirs fields preserve the nil/empty distinction:
//   - nil: the field is unset or the role is inapplicable
//   - empty []string{}: the field is set but grants zero items (read-only role)
//
// This distinction is load-bearing for read-only roles (e.g., plan-qa,
// build-qa, closeout), which set Edit: []string{} to signal read-only
// explicitly rather than leaving it nil.
//
// # Example
//
// A builder droplet might produce:
//
//	spec := &GateSpec{
//		Edit:         []string{"//abs/file.go", "//abs/file_test.go"},
//		WritableDirs: []string{"/abs/droplet-dir"},
//		BashDeny:     []string{"git commit", "git push", "mage install", "go get", "go mod"},
//		Network:      false,
//	}
//
// The adapters consume this and translate:
//   - claude hook: allowlist file by Edit path
//   - `claude -p`: `--allowedTools "Edit(//abs/file.go)" …` + `--disallowedTools "Bash(git commit:*)" …`
//   - codex exec: `-C /abs/droplet-dir` + execpolicy rules for BashDeny patterns
//
// # Distinction from Post-Build Gates
//
// This package defines the PreToolUse gate (agent sandboxing at dispatch time).
// The post-build gates (mage ci, commit, push) are defined in the parent
// dispatcher package (gates.go, gate_commit.go, etc.). These are distinct
// concepts:
//
//   - PreToolUse gate: restricts what a dispatched agent can access
//   - Post-build gates: verify that shipped code meets standards
//
// The package name "pretoolgate" prevents collision with the post-build
// gate machinery.
package pretoolgate
