// Package pretoolgate defines the GateSpec contract struct and per-channel
// translation accessors for PreToolUse agent sandboxing. The gate contract
// is the stable interface between the dispatcher (which produces a GateSpec)
// and the per-channel enforcement logic (which consumes and translates it).
//
// The PreToolUse gate is distinct from post-build gates (mage ci, commit,
// push). This package name "pretoolgate" avoids collision with the post-build
// gate machinery in the parent dispatcher package.
package pretoolgate

// GateSpec defines the dispatcher's gate contract: what files, directories,
// and bash patterns are granted to a dispatched agent, and whether network
// access is allowed. The contract is backend-agnostic; per-channel translation
// accessors project it onto each backend's enforcement form.
//
// This struct is the hand-off shape between the dispatcher (Go code that
// produces GateSpec from a spawn request) and the per-channel adapters
// (claude hook, `claude -p`, codex exec) that consume and translate it.
//
// The gate contract shape is specified in AGENT_SANDBOX_SPEC § 3 (consensus
// 2026-05-25, reproduced on ≥2 independent macOS runs).
type GateSpec struct {
	// Edit lists the files the dispatcher grants write access to, per-FILE.
	// Consumed by the claude built-in Agent tool gate hook and the `claude -p`
	// adapter. Files are absolute paths in //double-slash form (e.g.
	// "//absolute/path/file.go"). An empty slice (present-empty, Edit != nil)
	// denotes a read-only role (e.g., plan-qa, build-qa, closeout). A nil
	// slice indicates the spec is incomplete or the role is not applicable.
	// For claude channels, paths follow the AGENT_SANDBOX_SPEC § 2 rule:
	// double-slash absolute form; single-slash denies everything.
	Edit []string

	// WritableDirs lists the directories the dispatcher grants write access to,
	// per-DIRECTORY. Consumed by the codex exec adapter via the `-C` flag and
	// `--add-dir` flags. Directories are absolute paths (e.g.
	// "/absolute/path/to/droplet-dir"). An empty slice or nil indicates no
	// directory write access (read-only on codex).
	WritableDirs []string

	// BashDeny lists bash/shell patterns the dispatcher denies to the agent.
	// Patterns include git subcommands ("git commit", "git push", etc.) and
	// other restricted commands ("mage install", "go get", "go mod", etc.).
	// Consumed by the claude hook, `claude -p` adapter (as --disallowedTools
	// Bash(...)), and codex adapter (as execpolicy prefix_rule(forbidden)).
	// An empty slice or nil indicates no patterns are denied (permissive).
	BashDeny []string

	// Network specifies whether the agent is granted network access. False
	// (the default) means network access is denied. Consumed by the codex exec
	// adapter via `-c sandbox_workspace_write.network_access=false`. The claude
	// hook and `claude -p` adapter do not enforce network isolation today.
	Network bool
}

// BuiltinEdits returns the list of files granted write access on the claude
// built-in Agent tool channel. The returned slice reflects the Edit field
// directly, preserving the nil/empty distinction: nil means the role is
// read-only or unapplicable; an empty slice means read-only (present but
// empty).
func (g *GateSpec) BuiltinEdits() []string {
	if g == nil {
		return nil
	}
	return g.Edit
}

// BuiltinBashDeny returns the list of bash patterns denied on the claude
// built-in Agent tool channel. The returned slice reflects the BashDeny field
// directly, preserving the nil/empty distinction.
func (g *GateSpec) BuiltinBashDeny() []string {
	if g == nil {
		return nil
	}
	return g.BashDeny
}

// CodexWritableDirs returns the list of directories granted write access on
// the codex exec channel. The returned slice reflects the WritableDirs field
// directly, preserving the nil/empty distinction: nil means no directory-level
// write access; an empty slice means write access was explicitly granted to
// zero directories (read-only or a spec error).
func (g *GateSpec) CodexWritableDirs() []string {
	if g == nil {
		return nil
	}
	return g.WritableDirs
}

// CodexBashDeny returns the list of bash patterns denied on the codex exec
// channel. The returned slice reflects the BashDeny field directly,
// preserving the nil/empty distinction. The codex adapter translates each
// pattern into an execpolicy prefix_rule(decision="forbidden").
func (g *GateSpec) CodexBashDeny() []string {
	if g == nil {
		return nil
	}
	return g.BashDeny
}

// NetworkAccess returns whether network access is granted. The default is
// false (no network access).
func (g *GateSpec) NetworkAccess() bool {
	if g == nil {
		return false
	}
	return g.Network
}
