package dispatcher

import (
	"errors"
	"strings"
	"testing"
)

// TestParseCompletion_EmptyInput covers the (zero, ErrEmptyCompletion)
// path. Inputs that are empty OR contain only whitespace MUST surface the
// same sentinel so the dispatcher can route both to the "spawn produced
// nothing" outcome.
func TestParseCompletion_EmptyInput(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only_spaces", "   "},
		{"only_newlines", "\n\n\n"},
		{"mixed_whitespace", " \t\n\r\n  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := ParseCompletion(tc.input)
			if !errors.Is(err, ErrEmptyCompletion) {
				t.Fatalf("want ErrEmptyCompletion, got %v", err)
			}
			if report.Verdict != "" {
				t.Fatalf("want zero report, got Verdict=%q", report.Verdict)
			}
		})
	}
}

// TestParseCompletion_NoVerdict covers ErrNoVerdict — content present but
// no recognizable verdict line.
func TestParseCompletion_NoVerdict(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"prose_only", "Did the work. Looks good. Moving on."},
		{"only_section_headers", "## Tools Used\n- Read\n## Hylla Feedback\nNone."},
		{"verdict_with_wrong_token", "Verdict: MAYBE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := ParseCompletion(tc.input)
			if !errors.Is(err, ErrNoVerdict) {
				t.Fatalf("want ErrNoVerdict, got %v", err)
			}
			if report.Verdict != "" {
				t.Fatalf("want empty verdict, got %q", report.Verdict)
			}
		})
	}
}

// TestParseCompletion_VerdictVariants exercises every Verdict* constant.
// Acceptance criterion #4 + #6 require all four variants are recognized
// with case-insensitive matching tolerated.
func TestParseCompletion_VerdictVariants(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"pass_upper", "Verdict: PASS", VerdictPass},
		{"pass_lower", "Verdict: pass", VerdictPass},
		{"pass_mixed", "Verdict: Pass", VerdictPass},
		{"pass_with_nits", "Verdict: PASS-WITH-NITS", VerdictPassWithNits},
		{"pass_with_findings", "Verdict: PASS-WITH-FINDINGS", VerdictPassWithFindings},
		{"fail", "Verdict: FAIL", VerdictFail},
		{"bold_markdown", "**Verdict:** PASS", VerdictPass},
		{"list_marker", "- Verdict: FAIL", VerdictFail},
		{"with_trailing_text_before_newline", "Verdict: PASS-WITH-NITS — see below\n## NITs\n- foo", VerdictPassWithNits},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := ParseCompletion(tc.input)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if report.Verdict != tc.want {
				t.Fatalf("verdict: want %q, got %q", tc.want, report.Verdict)
			}
		})
	}
}

// TestParseCompletion_FirstVerdictWins guards the documented "first
// verdict line wins" policy. A subagent that re-iterates verdict prose
// later in the body MUST NOT silently flip the dispatcher's outcome.
func TestParseCompletion_FirstVerdictWins(t *testing.T) {
	input := "Verdict: PASS\n\nLater discussion mentions: Verdict: FAIL\n"
	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if report.Verdict != VerdictPass {
		t.Fatalf("verdict: want pass, got %q", report.Verdict)
	}
}

// TestParseCompletion_Section0Stripped is the load-bearing structural
// guard for the CLAUDE.md hard rule "Section 0 stays in the orchestrator-
// facing response ONLY — never inside Tillsyn comments." If this test
// regresses, the dispatcher will leak orchestrator-facing scratch into
// the durable Tillsyn thread.
func TestParseCompletion_Section0Stripped(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name: "em_dash_with_following_level1_heading",
			input: strings.Join([]string{
				"# Section 0 — SEMI-FORMAL REASONING",
				"## Planner",
				"Some secret scratch.",
				"## Builder",
				"More scratch.",
				"",
				"# Closing Report",
				"",
				"Verdict: PASS",
				"",
				"## Tools Used",
				"- Read",
			}, "\n"),
		},
		{
			name: "ascii_hyphen_runs_to_EOF",
			input: strings.Join([]string{
				"Verdict: PASS",
				"",
				"## Tools Used",
				"- Read",
				"",
				"# Section 0 - SEMI-FORMAL REASONING",
				"## Planner",
				"trailing scratch that must be stripped",
			}, "\n"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := ParseCompletion(tc.input)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if report.Verdict != VerdictPass {
				t.Fatalf("verdict: want pass, got %q", report.Verdict)
			}
			// The Tools Used section MUST survive the strip.
			if len(report.ToolsUsed) != 1 || report.ToolsUsed[0] != "Read" {
				t.Fatalf("tools_used: want [Read], got %#v", report.ToolsUsed)
			}
		})
	}
}

// TestParseCompletion_AllSectionsExtracted covers acceptance criterion #1
// + #3 + #6: every Section* field on CompletionReport is populated from
// the corresponding markdown section, the standard sections are extracted,
// and the PASS-WITH-FINDINGS verdict round-trips.
func TestParseCompletion_AllSectionsExtracted(t *testing.T) {
	input := strings.Join([]string{
		"# Section 0 — SEMI-FORMAL REASONING",
		"## Planner",
		"scratch",
		"",
		"# Closing Report",
		"",
		"Verdict: PASS-WITH-FINDINGS",
		"",
		"## Acceptance Coverage",
		"- AC1 — implemented in foo.go",
		"- AC2 — implemented in bar.go",
		"",
		"## Attack Vectors",
		"- A1: nil receiver — guarded by ErrXxx",
		"- A2: empty input — returns ErrEmptyCompletion",
		"",
		"## NITs",
		"- N1: rename foo → bar in follow-up",
		"",
		"## Failures",
		"- F1: missing test for edge case Z",
		"",
		"## Open Questions",
		"- Q1: should we support codex-exec terminal format too?",
		"",
		"## Tools Used",
		"- Read",
		"- Edit",
		"- mage test-pkg",
		"",
		"## Hylla Feedback",
		"- Query: TerminalReport",
		"- Missed because: ranking",
		"- Worked via: keyword fallback",
		"- Suggestion: boost exact tail_symbol match",
	}, "\n")

	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if report.Verdict != VerdictPassWithFindings {
		t.Fatalf("verdict: want pass-with-findings, got %q", report.Verdict)
	}
	if got, want := len(report.AcceptanceCoverage), 2; got != want {
		t.Fatalf("acceptance_coverage len: want %d, got %d (%#v)", want, got, report.AcceptanceCoverage)
	}
	if got, want := len(report.AttackVectors), 2; got != want {
		t.Fatalf("attack_vectors len: want %d, got %d (%#v)", want, got, report.AttackVectors)
	}
	if got, want := len(report.NITs), 1; got != want {
		t.Fatalf("nits len: want %d, got %d (%#v)", want, got, report.NITs)
	}
	if got, want := len(report.Failures), 1; got != want {
		t.Fatalf("failures len: want %d, got %d (%#v)", want, got, report.Failures)
	}
	if got, want := len(report.OpenQuestions), 1; got != want {
		t.Fatalf("open_questions len: want %d, got %d (%#v)", want, got, report.OpenQuestions)
	}
	if got, want := len(report.ToolsUsed), 3; got != want {
		t.Fatalf("tools_used len: want %d, got %d (%#v)", want, got, report.ToolsUsed)
	}
	if report.ToolsUsed[2] != "mage test-pkg" {
		t.Fatalf("tools_used[2]: want %q, got %q", "mage test-pkg", report.ToolsUsed[2])
	}
	if !strings.Contains(report.HyllaFeedback, "TerminalReport") {
		t.Fatalf("hylla_feedback: want body containing TerminalReport, got %q", report.HyllaFeedback)
	}
}

// TestParseCompletion_HyllaFeedbackNoneStanza guards the documented
// "None — Hylla answered everything needed." collapse-to-empty behavior.
// Cascade-end aggregation only consumes non-empty bodies; the canonical
// None stanza is informational and MUST NOT propagate.
func TestParseCompletion_HyllaFeedbackNoneStanza(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		empty bool
	}{
		{"em_dash_canonical", "None — Hylla answered everything needed.", true},
		{"ascii_hyphen", "None - Hylla answered everything needed.", true},
		{"period_form", "None. Hylla answered everything needed.", true},
		{"real_miss", "- Query: Foo\n- Missed because: bar", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := "Verdict: PASS\n\n## Hylla Feedback\n" + tc.body + "\n"
			report, err := ParseCompletion(input)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tc.empty && report.HyllaFeedback != "" {
				t.Fatalf("want empty, got %q", report.HyllaFeedback)
			}
			if !tc.empty && report.HyllaFeedback == "" {
				t.Fatalf("want non-empty, got empty")
			}
		})
	}
}

// TestParseCompletion_FencedCodeHeadingsIgnored is the falsification guard
// against a `## Heading` inside a fenced code block falsely terminating
// the prior section. Without code-fence masking a code sample showing
// `## Tools Used` would clobber an actual Tools Used section that comes
// earlier in the body.
func TestParseCompletion_FencedCodeHeadingsIgnored(t *testing.T) {
	input := strings.Join([]string{
		"Verdict: PASS",
		"",
		"## Tools Used",
		"- Read",
		"- Edit",
		"",
		"## Acceptance Coverage",
		"- AC1 — see snippet:",
		"```",
		"## Tools Used",
		"- DO_NOT_PICK_THIS_UP",
		"```",
	}, "\n")

	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got, want := len(report.ToolsUsed), 2; got != want {
		t.Fatalf("tools_used len: want %d, got %d (%#v)", want, got, report.ToolsUsed)
	}
	for _, tool := range report.ToolsUsed {
		if strings.Contains(tool, "DO_NOT_PICK_THIS_UP") {
			t.Fatalf("code-block heading leaked into tools_used: %q", tool)
		}
	}
}

// TestParseCompletion_BulletVariants confirms the bullet extractor
// handles `-`, `*`, `+`, `1.`, `1)` markers and continuation lines.
func TestParseCompletion_BulletVariants(t *testing.T) {
	input := strings.Join([]string{
		"Verdict: PASS",
		"",
		"## NITs",
		"- dash item",
		"* asterisk item",
		"+ plus item",
		"1. numbered-period item",
		"2) numbered-paren item",
		"   continuation line for the prior item",
	}, "\n")

	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got, want := len(report.NITs), 5; got != want {
		t.Fatalf("nits len: want %d, got %d (%#v)", want, got, report.NITs)
	}
	if !strings.Contains(report.NITs[4], "continuation line") {
		t.Fatalf("continuation not joined into 5th entry: %q", report.NITs[4])
	}
}

// TestParseCompletion_DuplicateSectionLastWins guards the documented
// "duplicate heading → last occurrence wins" policy. Subagent prompt
// drift that produces two `## NITs` headings should not silently merge
// the bodies; the later one wins.
func TestParseCompletion_DuplicateSectionLastWins(t *testing.T) {
	input := strings.Join([]string{
		"Verdict: PASS",
		"",
		"## NITs",
		"- stale entry from earlier draft",
		"",
		"## NITs",
		"- final entry",
	}, "\n")

	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got, want := len(report.NITs), 1; got != want {
		t.Fatalf("nits len: want %d, got %d (%#v)", want, got, report.NITs)
	}
	if report.NITs[0] != "final entry" {
		t.Fatalf("nits[0]: want %q, got %q", "final entry", report.NITs[0])
	}
}

// TestParseCompletion_SectionAliases covers the documented section title
// aliases — `## Findings` MUST map to Failures; `## Unknowns` MUST map to
// OpenQuestions; `## Falsification Attacks` MUST map to AttackVectors.
// Aliases exist because the 8-persona QA split (post-2026-05-21) varies
// per-axis vocabulary slightly and the parser must accept both forms.
func TestParseCompletion_SectionAliases(t *testing.T) {
	input := strings.Join([]string{
		"Verdict: FAIL",
		"",
		"## Findings",
		"- F1 mapped from Findings alias",
		"",
		"## Unknowns",
		"- U1 mapped from Unknowns alias",
		"",
		"## Falsification Attacks",
		"- A1 mapped from Falsification Attacks alias",
	}, "\n")

	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got, want := len(report.Failures), 1; got != want {
		t.Fatalf("failures len: want %d, got %d (%#v)", want, got, report.Failures)
	}
	if got, want := len(report.OpenQuestions), 1; got != want {
		t.Fatalf("open_questions len: want %d, got %d (%#v)", want, got, report.OpenQuestions)
	}
	if got, want := len(report.AttackVectors), 1; got != want {
		t.Fatalf("attack_vectors len: want %d, got %d (%#v)", want, got, report.AttackVectors)
	}
}

// TestParseCompletion_CRLFLineEndings guards round-trip through
// Windows-style line endings. Some adapter buffering paths emit CRLF;
// the parser MUST handle both transparently.
func TestParseCompletion_CRLFLineEndings(t *testing.T) {
	input := "Verdict: PASS\r\n\r\n## Tools Used\r\n- Read\r\n- Edit\r\n"
	report, err := ParseCompletion(input)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if report.Verdict != VerdictPass {
		t.Fatalf("verdict: want pass, got %q", report.Verdict)
	}
	if got, want := len(report.ToolsUsed), 2; got != want {
		t.Fatalf("tools_used len: want %d, got %d (%#v)", want, got, report.ToolsUsed)
	}
}
