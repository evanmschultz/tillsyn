package dispatcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// agentRepoRelPath is the repo-rooted relative path to the canonical agent
// markdown directory. Tests resolve absolute paths from this via
// repoRootForAgentTests, which walks parents looking for go.mod.
const agentRepoRelPath = ".claude/agents"

// repoRootForAgentTests resolves the absolute path of the tillsyn repo
// root by walking up from the test working directory looking for go.mod.
// Test packages run from the package directory; the repo root is two
// levels above internal/app/dispatcher.
func repoRootForAgentTests(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root (go.mod) from %s", wd)
		}
		dir = parent
	}
}

// TestParseAgentDefinition_EnumeratedFiles iterates the 9 files declared in
// the action_item `files` field plus the 4 sibling FE files that exist on
// disk, asserting each parses cleanly and the derived role/axis/language
// match the filename pattern. This is the table-driven core test per AC #4
// — the enumeration is expanded to cover all 13 personas because the same
// parser handles all of them and silent FE regressions are a real risk.
func TestParseAgentDefinition_EnumeratedFiles(t *testing.T) {
	cases := []struct {
		name         string
		filename     string
		wantRole     string
		wantAxis     string
		wantLanguage string
	}{
		// 9 enumerated files in the action_item brief.
		{"go-builder", "ta-go-builder", "builder", "build", "go"},
		{"go-planning", "ta-go-planning", "planner", "plan", "go"},
		{"go-build-qa-proof", "ta-go-build-qa-proof", "qa-proof", "build", "go"},
		{"go-build-qa-falsification", "ta-go-build-qa-falsification", "qa-falsification", "build", "go"},
		{"go-plan-qa-proof", "ta-go-plan-qa-proof", "qa-proof", "plan", "go"},
		{"go-plan-qa-falsification", "ta-go-plan-qa-falsification", "qa-falsification", "plan", "go"},
		{"fe-builder", "ta-fe-builder", "builder", "build", "fe"},
		{"fe-planning", "ta-fe-planning", "planner", "plan", "fe"},
		{"closeout", "ta-closeout", "closeout", "none", "none"},

		// 4 additional FE personas that exist on disk; same parser shape.
		{"fe-build-qa-proof", "ta-fe-build-qa-proof", "qa-proof", "build", "fe"},
		{"fe-build-qa-falsification", "ta-fe-build-qa-falsification", "qa-falsification", "build", "fe"},
		{"fe-plan-qa-proof", "ta-fe-plan-qa-proof", "qa-proof", "plan", "fe"},
		{"fe-plan-qa-falsification", "ta-fe-plan-qa-falsification", "qa-falsification", "plan", "fe"},
	}

	root := repoRootForAgentTests(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(root, agentRepoRelPath, tc.filename+".md")
			got, err := LoadAgentDefinition(path)
			if err != nil {
				t.Fatalf("LoadAgentDefinition(%s) returned err: %v", path, err)
			}
			if got.Name != tc.filename {
				t.Errorf("Name = %q, want %q (frontmatter name must match filename)", got.Name, tc.filename)
			}
			if got.Description == "" {
				t.Errorf("Description is empty; every ta-*.md must have a description")
			}
			if len(got.Tools) == 0 {
				t.Errorf("Tools is empty; every ta-*.md must declare at least one tool")
			}
			if got.Role != tc.wantRole {
				t.Errorf("Role = %q, want %q", got.Role, tc.wantRole)
			}
			if got.Axis != tc.wantAxis {
				t.Errorf("Axis = %q, want %q", got.Axis, tc.wantAxis)
			}
			if got.Language != tc.wantLanguage {
				t.Errorf("Language = %q, want %q", got.Language, tc.wantLanguage)
			}
			if got.SystemPrompt == "" {
				t.Errorf("SystemPrompt is empty; body after closing `---` must be preserved")
			}
		})
	}
}

// TestParseAgentDefinition_AllOnDisk sweeps the entire .claude/agents/
// directory and asserts every ta-*.md file parses without error. Closes
// the silent-FE-regression counterexample: if a new ta-*.md lands in the
// repo, this test runs against it the moment it exists. The 13-file count
// is asserted as a floor — adding a 14th persona is fine, dropping below
// 13 is a regression that fails here.
func TestParseAgentDefinition_AllOnDisk(t *testing.T) {
	root := repoRootForAgentTests(t)
	pattern := filepath.Join(root, agentRepoRelPath, "ta-*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Glob(%s): %v", pattern, err)
	}

	// ta-test-ollama is a manual smoke-test persona for bin/agent-dispatch.sh
	// dispatch_ollama exercising (HYLLA_BIN §2). Orchestrator never routes
	// there automatically; the dispatcher Go code never calls LoadAgentDefinition
	// for it. Exclude from the production-set sweep.
	var productionMatches []string
	for _, p := range matches {
		if strings.TrimSuffix(filepath.Base(p), ".md") != "ta-test-ollama" {
			productionMatches = append(productionMatches, p)
		}
	}
	if len(productionMatches) < 13 {
		t.Fatalf("found %d production ta-*.md files at %s, want ≥13 (one per persona)", len(productionMatches), pattern)
	}

	for _, path := range productionMatches {
		path := path
		base := strings.TrimSuffix(filepath.Base(path), ".md")
		t.Run(base, func(t *testing.T) {
			def, err := LoadAgentDefinition(path)
			if err != nil {
				t.Fatalf("LoadAgentDefinition(%s) returned err: %v", path, err)
			}
			if def.Name == "" {
				t.Errorf("Name empty for %s", path)
			}
			if def.Role == "" || def.Axis == "" || def.Language == "" {
				t.Errorf("classification incomplete: role=%q axis=%q language=%q", def.Role, def.Axis, def.Language)
			}
		})
	}
}

// TestParseAgentDefinition_ErrInvalidAgentName asserts that names not
// matching the canonical pattern reject loud per AC #2.
func TestParseAgentDefinition_ErrInvalidAgentName(t *testing.T) {
	cases := []struct {
		name     string
		filename string
	}{
		{"empty", ""},
		{"missing ta prefix", "go-builder"},
		{"wrong language", "ta-rb-builder"},
		{"unknown role", "ta-go-deployer"},
		{"typo in qa axis", "ta-go-plan-qa-prof"},
		{"trailing junk", "ta-go-builder-extra"},
		{"closeout with language", "ta-go-closeout"},
		{"ta- only", "ta-"},
	}
	body := []byte("---\nname: foo\ndescription: bar\ntools: Read\n---\n\nbody\n")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAgentDefinition(tc.filename, body)
			if !errors.Is(err, ErrInvalidAgentName) {
				t.Errorf("err = %v, want errors.Is(err, ErrInvalidAgentName)", err)
			}
		})
	}
}

// TestParseAgentDefinition_ErrMalformedFrontmatter asserts that bodies
// without a valid `---` block reject loud per AC #3.
func TestParseAgentDefinition_ErrMalformedFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"no frontmatter at all", "just a body, no delimiters\n"},
		{"open without close", "---\nname: ta-go-builder\ndescription: x\ntools: Read\n\nbody-without-close\n"},
		{"empty file", ""},
		{"malformed yaml", "---\nname: ta-go-builder\ntools: [unclosed\n---\n\nbody\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAgentDefinition("ta-go-builder", []byte(tc.body))
			if !errors.Is(err, ErrMalformedFrontmatter) {
				t.Errorf("err = %v, want errors.Is(err, ErrMalformedFrontmatter)", err)
			}
		})
	}
}

// TestParseAgentDefinition_ModelField verifies the `model` field
// round-trips verbatim from frontmatter — load-bearing for the multi-
// backend routing thesis (haiku/sonnet/opus per persona).
func TestParseAgentDefinition_ModelField(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantModel string
	}{
		{
			name:      "haiku for builder",
			body:      "---\nname: ta-go-builder\ndescription: x\nmodel: haiku\ntools: Read\n---\n\nbody\n",
			wantModel: "haiku",
		},
		{
			name:      "sonnet for build-qa",
			body:      "---\nname: ta-go-build-qa-proof\ndescription: x\nmodel: sonnet\ntools: Read\n---\n\nbody\n",
			wantModel: "sonnet",
		},
		{
			name:      "opus for plan-qa",
			body:      "---\nname: ta-go-plan-qa-proof\ndescription: x\nmodel: opus\ntools: Read\n---\n\nbody\n",
			wantModel: "opus",
		},
		{
			name:      "absent model is empty string",
			body:      "---\nname: ta-go-planning\ndescription: x\ntools: Read\n---\n\nbody\n",
			wantModel: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a filename that always matches the regex; the role/axis
			// derivation is exercised elsewhere.
			fn := "ta-go-builder"
			if strings.Contains(tc.body, "ta-go-build-qa-proof") {
				fn = "ta-go-build-qa-proof"
			} else if strings.Contains(tc.body, "ta-go-plan-qa-proof") {
				fn = "ta-go-plan-qa-proof"
			} else if strings.Contains(tc.body, "ta-go-planning") {
				fn = "ta-go-planning"
			}
			def, err := ParseAgentDefinition(fn, []byte(tc.body))
			if err != nil {
				t.Fatalf("ParseAgentDefinition: %v", err)
			}
			if def.Model != tc.wantModel {
				t.Errorf("Model = %q, want %q", def.Model, tc.wantModel)
			}
		})
	}
}

// TestParseAgentDefinition_ToolsField verifies the comma-separated tools
// string splits + trims correctly. The ta-*.md files ship dozens of tools
// per line; the parser must not drop any.
func TestParseAgentDefinition_ToolsField(t *testing.T) {
	cases := []struct {
		name      string
		toolsLine string
		want      []string
	}{
		{"single tool", "Read", []string{"Read"}},
		{"two tools", "Read, Write", []string{"Read", "Write"}},
		{"extra whitespace", "  Read,   Write , Edit ", []string{"Read", "Write", "Edit"}},
		{"trailing comma", "Read, Write,", []string{"Read", "Write"}},
		{"empty string", "", nil},
		{"only whitespace", "   ", nil},
		{"mcp-style names", "mcp__tillsyn__till_action_item, mcp__ta__schema", []string{"mcp__tillsyn__till_action_item", "mcp__ta__schema"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte("---\nname: ta-go-builder\ndescription: x\ntools: " + tc.toolsLine + "\n---\n\nbody\n")
			def, err := ParseAgentDefinition("ta-go-builder", body)
			if err != nil {
				t.Fatalf("ParseAgentDefinition: %v", err)
			}
			if len(def.Tools) != len(tc.want) {
				t.Fatalf("Tools len = %d (%v), want %d (%v)", len(def.Tools), def.Tools, len(tc.want), tc.want)
			}
			for i, got := range def.Tools {
				if got != tc.want[i] {
					t.Errorf("Tools[%d] = %q, want %q", i, got, tc.want[i])
				}
			}
		})
	}
}

// TestParseAgentDefinition_SystemPromptPreserved verifies the body after
// the closing `---` is preserved verbatim (the system prompt is the load-
// bearing payload downstream consumers render).
func TestParseAgentDefinition_SystemPromptPreserved(t *testing.T) {
	body := []byte("---\nname: ta-go-builder\ndescription: x\ntools: Read\n---\n\nFirst line of system prompt.\n\nSecond paragraph with **markdown**.\n")
	def, err := ParseAgentDefinition("ta-go-builder", body)
	if err != nil {
		t.Fatalf("ParseAgentDefinition: %v", err)
	}
	if !strings.Contains(def.SystemPrompt, "First line of system prompt.") {
		t.Errorf("SystemPrompt missing first line; got %q", def.SystemPrompt)
	}
	if !strings.Contains(def.SystemPrompt, "Second paragraph with **markdown**.") {
		t.Errorf("SystemPrompt missing second paragraph; got %q", def.SystemPrompt)
	}
}

// TestLoadAgentDefinition_FileNotFound asserts I/O errors surface clean.
func TestLoadAgentDefinition_FileNotFound(t *testing.T) {
	_, err := LoadAgentDefinition(filepath.Join(t.TempDir(), "ta-go-builder.md"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "read agent file") {
		t.Errorf("error message should mention 'read agent file'; got %v", err)
	}
}

// TestParseAgentDefinition_MCPServersFromFrontmatter verifies the `mcp_servers:`
// frontmatter block parses into the MCPServers map and the per-server config
// (command, args, tools) round-trips correctly.
func TestParseAgentDefinition_MCPServersFromFrontmatter(t *testing.T) {
	body := []byte(`---
name: ta-go-builder
description: Builder
model: haiku
tools: Read, Write
mcp_servers:
  tillsyn-dev:
    command: till
    args: [mcp]
    tools: [till.action_item, till.comment]
  hylla-search:
    command: hylla
    args: [mcp, search]
    tools: [hylla_search_keyword, hylla_search_vector]
---

body
`)
	def, err := ParseAgentDefinition("ta-go-builder", body)
	if err != nil {
		t.Fatalf("ParseAgentDefinition: %v", err)
	}

	if len(def.MCPServers) != 2 {
		t.Fatalf("MCPServers len = %d, want 2; got %v", len(def.MCPServers), def.MCPServers)
	}

	// Check tillsyn-dev server
	tilldev, ok := def.MCPServers["tillsyn-dev"]
	if !ok {
		t.Fatal("missing tillsyn-dev server")
	}
	if tilldev.Command != "till" {
		t.Errorf("tillsyn-dev.Command = %q, want %q", tilldev.Command, "till")
	}
	if len(tilldev.Args) != 1 || tilldev.Args[0] != "mcp" {
		t.Errorf("tillsyn-dev.Args = %v, want %v", tilldev.Args, []string{"mcp"})
	}
	if len(tilldev.Tools) != 2 || tilldev.Tools[0] != "till.action_item" || tilldev.Tools[1] != "till.comment" {
		t.Errorf("tillsyn-dev.Tools = %v, want %v", tilldev.Tools, []string{"till.action_item", "till.comment"})
	}

	// Check hylla-search server
	hylla, ok := def.MCPServers["hylla-search"]
	if !ok {
		t.Fatal("missing hylla-search server")
	}
	if hylla.Command != "hylla" {
		t.Errorf("hylla-search.Command = %q, want %q", hylla.Command, "hylla")
	}
	if len(hylla.Args) != 2 || hylla.Args[0] != "mcp" || hylla.Args[1] != "search" {
		t.Errorf("hylla-search.Args = %v, want %v", hylla.Args, []string{"mcp", "search"})
	}
	if len(hylla.Tools) != 2 || hylla.Tools[0] != "hylla_search_keyword" || hylla.Tools[1] != "hylla_search_vector" {
		t.Errorf("hylla-search.Tools = %v, want %v", hylla.Tools, []string{"hylla_search_keyword", "hylla_search_vector"})
	}
}

// TestParseAgentDefinition_MCPServersAbsentYieldsEmpty verifies backward
// compatibility: agents without the `mcp_servers:` key parse cleanly with
// a nil or empty MCPServers map.
func TestParseAgentDefinition_MCPServersAbsentYieldsEmpty(t *testing.T) {
	body := []byte("---\nname: ta-go-builder\ndescription: x\ntools: Read\n---\n\nbody\n")
	def, err := ParseAgentDefinition("ta-go-builder", body)
	if err != nil {
		t.Fatalf("ParseAgentDefinition: %v", err)
	}
	if len(def.MCPServers) != 0 {
		t.Errorf("MCPServers = %v with len %d, want 0", def.MCPServers, len(def.MCPServers))
	}
}
