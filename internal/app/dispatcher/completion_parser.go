// Package dispatcher completion_parser.go ships the Lane C D7 completion
// parser: a CLI-agnostic extractor that consumes the terminal-event text
// produced by a headless subagent spawn (via Monitor + CLIAdapter) and
// surfaces a structured CompletionReport for the dispatcher to:
//
//  1. Post as the closing-comment body on the action item via till.comment.
//  2. Determine metadata.outcome ("success" | "failure") for the move_state
//     transition that retires the spawn's action item.
//
// Backend-agnostic by construction: ParseCompletion accepts a free-form
// markdown string and never asks who produced it (claude -p, codex exec,
// claude --bare → ollama, future bare backends). The parser is the seam
// where subagent prose becomes a typed dispatcher record.
//
// Why a separate file (vs folding into commit_agent.go): CommitAgent
// extracts a SINGLE-LINE conventional-commit message and length-caps it.
// ParseCompletion extracts a RICH structured verdict-bundle and applies
// section-aware extraction. Different shape, different cap, different
// failure modes — kept separate to keep each surface narrowly testable.
//
// Section 0 stripping: per the project CLAUDE.md hard rule, the
// "# Section 0 — SEMI-FORMAL REASONING" block is orchestrator-facing
// scratch and MUST NEVER land inside Tillsyn description/comments/handoffs.
// ParseCompletion strips that block (and any leading whitespace before it)
// BEFORE further parsing so downstream `till.comment` posts cannot leak
// Section 0 into the durable thread.
package dispatcher

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// VerdictPass marks the build / QA subagent reported clean success. Maps to
// metadata.outcome="success" at move_state time.
const VerdictPass = "pass"

// VerdictPassWithNits marks success with minor follow-ups the subagent
// could not absorb inline. Treated as success at move_state (the NITs are
// surfaced via the closing comment for orchestrator triage).
const VerdictPassWithNits = "pass-with-nits"

// VerdictPassWithFindings marks QA-falsification success that uncovered
// concrete attacks the orchestrator must respond to (vs informational
// NITs). Distinct from pass-with-nits because the findings ARE the load-
// bearing output; move_state still treats this as success.
const VerdictPassWithFindings = "pass-with-findings"

// VerdictFail marks the subagent reported the work-unit is not complete.
// Maps to metadata.outcome="failure" at move_state time so the orchestrator
// can route to a follow-up build round.
const VerdictFail = "fail"

// ErrEmptyCompletion is returned by ParseCompletion when the input is the
// empty string or contains only whitespace. The dispatcher treats this as a
// distinct failure mode from "agent ran but produced no verdict line" — an
// empty completion usually means the spawn never reached its closing
// comment block (crash mid-prose), while a missing-verdict completion has
// content but no parseable verdict marker.
var ErrEmptyCompletion = errors.New("dispatcher: completion parser received empty input")

// ErrNoVerdict is returned by ParseCompletion when the input has content
// but no parseable `Verdict: PASS|PASS-WITH-NITS|PASS-WITH-FINDINGS|FAIL`
// line. Distinct from ErrEmptyCompletion so the dispatcher can route a
// no-verdict completion to "agent ran but malformed report" (probable
// prompt drift) vs an empty completion (probable spawn crash).
var ErrNoVerdict = errors.New("dispatcher: completion parser found no verdict line")

// CompletionReport is the structured shape ParseCompletion produces from a
// subagent's terminal-event text. Fields mirror the standard subagent
// closing-comment sections defined in the per-role agent prompts under
// .claude/agents/ta-*.md (post-2026-05-21 8-persona split).
//
// Nil and empty slices are equivalent throughout — both mean "no items
// reported under this section." The distinction is not load-bearing for
// downstream consumers.
//
// HyllaFeedback is kept as a single string (vs a parsed structure) because
// the per-miss shape (Query / Missed because / Worked via / Suggestion) is
// itself a free-form sub-template that varies across agent prompts. The
// dispatcher posts the section body verbatim into the closing comment so
// the orchestrator can aggregate at cascade-end; structured parsing of the
// per-miss entries is YAGNI today.
type CompletionReport struct {
	// Verdict is one of the four Verdict* constants (lowercased) when
	// extraction succeeds. Empty when ErrNoVerdict is returned.
	Verdict string

	// AcceptanceCoverage lists the parent action_item's acceptance criteria
	// the subagent claims to have addressed, one entry per criterion. Sourced
	// from the "## Acceptance Coverage" markdown section.
	AcceptanceCoverage []string

	// AttackVectors lists the falsification attacks the subagent enumerated
	// against its own (or a peer's) output. Sourced from "## Attack Vectors"
	// or "## Falsification Attacks" sections.
	AttackVectors []string

	// NITs lists minor-severity issues the subagent surfaced for orchestrator
	// triage. Sourced from "## NITs" markdown section.
	NITs []string

	// Failures lists hard-failure findings (load-bearing, must-fix issues)
	// the subagent surfaced. Distinct from NITs by severity — failures
	// indicate the work-unit is NOT complete as-is. Sourced from
	// "## Failures" or "## Findings" markdown section.
	Failures []string

	// OpenQuestions lists unknowns the subagent could not resolve and
	// routes to the orchestrator for clarification. Sourced from
	// "## Open Questions" or "## Unknowns" markdown section.
	OpenQuestions []string

	// ToolsUsed lists the MCP/tool names the subagent invoked during its
	// run, one entry per tool. Sourced from the "## Tools Used" markdown
	// section. Empty (nil) is valid — some agent prompts do not require
	// the section, and the orchestrator's tool-call tracking memory rule
	// (feedback_track_subagent_tool_calls) addresses absent sections at
	// the orchestrator layer rather than the parser layer.
	ToolsUsed []string

	// HyllaFeedback is the verbatim body of the "## Hylla Feedback" section
	// (including its sub-bullets) as a single string. Empty when the section
	// is missing OR contains the standard "None — Hylla answered everything
	// needed." stanza. The dispatcher posts this verbatim so the orchestrator
	// can aggregate at cascade-end.
	HyllaFeedback string
}

// section0HeadingRE matches the orchestrator-facing scratch block heading
// at any position in the input. Anchored with (?m) so ^ binds to line
// starts. The em-dash variant matches what CLAUDE.md prescribes; an
// ASCII-hyphen variant is matched too because terminal-text round-trip
// through some adapters mangles em-dashes.
//
// Captured group: the whole heading line. The strip implementation slices
// from the heading START forward to the NEXT level-1 heading OR EOF, so
// the regex only needs to locate the start.
var section0HeadingRE = regexp.MustCompile(`(?m)^#\s*Section\s*0\s*[—-]\s*SEMI[-\s]FORMAL\s*REASONING\b.*$`)

// nextLevel1HeadingRE matches any `# ` markdown heading line. Used to find
// the end of a Section 0 block. Section 0 is by convention a level-1
// heading, so the NEXT level-1 heading after Section 0's start is where
// the parser's downstream consumption begins.
var nextLevel1HeadingRE = regexp.MustCompile(`(?m)^#\s+\S`)

// verdictLineRE extracts the verdict marker from a line of the form
// `Verdict: PASS-WITH-NITS` (case-insensitive, leading whitespace
// tolerated, optional `**Verdict:**` bold markdown form tolerated). The
// captured group is the verdict token. Matches at any position on its
// line so a leading list-marker (e.g. `- Verdict: PASS`) is tolerated.
var verdictLineRE = regexp.MustCompile(`(?im)^[\s\-*>]*\**\s*Verdict\s*\**\s*:\s*\**\s*(PASS-WITH-NITS|PASS-WITH-FINDINGS|PASS|FAIL)\b`)

// sectionHeadingRE matches a level-2 markdown heading. Captured group is
// the heading title (with surrounding whitespace untrimmed; callers trim).
// Used to slice the body into named sections.
var sectionHeadingRE = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

// codeFenceRE matches a markdown fenced code block (triple-backtick or
// triple-tilde). Used to mask code blocks before heading extraction so a
// `## Heading` inside a code sample does not falsely terminate a section.
var codeFenceRE = regexp.MustCompile("(?ms)^(```|~~~)[^\\n]*\\n.*?^(```|~~~)\\s*$")

// ParseCompletion extracts a structured CompletionReport from a subagent's
// closing-comment markdown.
//
// Algorithm:
//
//  1. Reject empty/whitespace-only input with ErrEmptyCompletion.
//  2. Strip the orchestrator-facing "# Section 0 — SEMI-FORMAL REASONING"
//     block (everything from the Section 0 heading up to the next level-1
//     heading OR EOF). The strip happens BEFORE further parsing so a
//     Section 0 leak into Tillsyn comments is structurally impossible.
//  3. Extract the verdict via verdictLineRE; ErrNoVerdict if absent.
//  4. Mask fenced code blocks so headings inside code samples do not
//     falsely terminate sections.
//  5. Split the masked text into named level-2 sections.
//  6. Map known section titles to CompletionReport fields, splitting body
//     text into bullet-style entries (one entry per `- ` / `* ` / `1.`
//     line, leading marker stripped, surrounding whitespace trimmed).
//  7. HyllaFeedback is kept as verbatim body text (no bullet splitting)
//     and is empty when the body is "None — Hylla answered everything
//     needed." or the equivalent ASCII-hyphen form.
//
// Returns:
//
//   - (report, nil) on the happy path. Verdict + every populated section
//     are extracted; sections absent from the input are nil/empty slices.
//   - (zero, ErrEmptyCompletion) when input has no non-whitespace content.
//   - (zero, ErrNoVerdict) when input has content but no verdict line.
func ParseCompletion(terminalText string) (CompletionReport, error) {
	if strings.TrimSpace(terminalText) == "" {
		return CompletionReport{}, ErrEmptyCompletion
	}

	stripped := stripSection0(terminalText)

	verdict, ok := extractVerdict(stripped)
	if !ok {
		return CompletionReport{}, fmt.Errorf("%w (input_len=%d)", ErrNoVerdict, len(stripped))
	}

	masked := maskCodeFences(stripped)
	sections := splitSections(masked, stripped)

	report := CompletionReport{Verdict: verdict}

	for title, body := range sections {
		normalized := normalizeSectionTitle(title)
		switch normalized {
		case "acceptance coverage", "coverage":
			report.AcceptanceCoverage = extractBullets(body)
		case "attack vectors", "falsification attacks", "attacks":
			report.AttackVectors = extractBullets(body)
		case "nits":
			report.NITs = extractBullets(body)
		case "failures", "findings":
			report.Failures = extractBullets(body)
		case "open questions", "unknowns":
			report.OpenQuestions = extractBullets(body)
		case "tools used", "tools":
			report.ToolsUsed = extractBullets(body)
		case "hylla feedback":
			report.HyllaFeedback = extractHyllaFeedbackBody(body)
		}
	}

	return report, nil
}

// stripSection0 removes the "# Section 0 — SEMI-FORMAL REASONING" block
// from input. The block spans from the Section 0 heading START to the
// next level-1 heading OR EOF (whichever comes first). When no Section 0
// heading is present the input is returned unchanged.
//
// Implementation note: locating the END of Section 0 by "next level-1
// heading" rather than a fixed line count handles the case where a
// subagent's body uses additional `# ` headings AFTER Section 0 — the
// strip stops at the first one, preserving the rest of the body.
func stripSection0(input string) string {
	startLoc := section0HeadingRE.FindStringIndex(input)
	if startLoc == nil {
		return input
	}
	tail := input[startLoc[1]:]
	endLoc := nextLevel1HeadingRE.FindStringIndex(tail)
	if endLoc == nil {
		// Section 0 runs to EOF. Everything before is kept.
		return strings.TrimSpace(input[:startLoc[0]])
	}
	// Re-stitch: prefix BEFORE Section 0 + content FROM the next level-1
	// heading onward. The space-preservation is loose by design; downstream
	// section parsing does its own whitespace handling.
	prefix := input[:startLoc[0]]
	suffix := tail[endLoc[0]:]
	return prefix + suffix
}

// extractVerdict returns the verdict token (lowercased, matched against
// the Verdict* constants) and ok=true when verdictLineRE finds a match.
// Empty token + ok=false otherwise.
//
// Match policy: the FIRST verdict line wins. Subagent prompts emit
// exactly one `Verdict: ...` line; if a re-iterated verdict appears later
// in the prose, the first one is the load-bearing one.
func extractVerdict(input string) (string, bool) {
	match := verdictLineRE.FindStringSubmatch(input)
	if len(match) < 2 {
		return "", false
	}
	switch strings.ToUpper(match[1]) {
	case "PASS-WITH-NITS":
		return VerdictPassWithNits, true
	case "PASS-WITH-FINDINGS":
		return VerdictPassWithFindings, true
	case "PASS":
		return VerdictPass, true
	case "FAIL":
		return VerdictFail, true
	}
	return "", false
}

// maskCodeFences replaces every fenced code block with the same number of
// blank lines so subsequent heading extraction does not pick up `## H`
// lines inside code samples. Line count is preserved so byte offsets into
// the masked string align with the unmasked string at line boundaries —
// callers that need verbatim body text re-extract from the unmasked input
// using the line-range identified via the mask.
func maskCodeFences(input string) string {
	return codeFenceRE.ReplaceAllStringFunc(input, func(block string) string {
		// Preserve newlines so line counts match.
		newlines := strings.Count(block, "\n")
		return strings.Repeat("\n", newlines)
	})
}

// splitSections returns a map of level-2 heading title → body content for
// every `## Heading` block in `masked`. The body text comes from the
// `unmasked` source so any in-section code fences round-trip verbatim.
//
// On duplicate headings the LAST occurrence wins. Subagent prompts emit
// each section at most once; a duplicate indicates prompt drift and the
// dispatcher prefers the later body (most-recent overrides earlier).
func splitSections(masked, unmasked string) map[string]string {
	out := make(map[string]string)

	headings := sectionHeadingRE.FindAllStringSubmatchIndex(masked, -1)
	if len(headings) == 0 {
		return out
	}

	for i, hdr := range headings {
		titleStart := hdr[2]
		titleEnd := hdr[3]
		bodyStart := hdr[1] // immediately after the heading line
		var bodyEnd int
		if i+1 < len(headings) {
			bodyEnd = headings[i+1][0]
		} else {
			bodyEnd = len(unmasked)
		}
		if bodyStart > len(unmasked) {
			bodyStart = len(unmasked)
		}
		if bodyEnd > len(unmasked) {
			bodyEnd = len(unmasked)
		}
		if bodyEnd < bodyStart {
			bodyEnd = bodyStart
		}
		title := unmasked[titleStart:titleEnd]
		body := unmasked[bodyStart:bodyEnd]
		out[title] = body
	}
	return out
}

// normalizeSectionTitle lowercases + trims a heading title so lookups in
// ParseCompletion's section switch match across stylistic variations
// ("Tools Used" vs "tools used" vs "Tools  Used  ").
func normalizeSectionTitle(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	// Collapse runs of internal whitespace to single spaces.
	return strings.Join(strings.Fields(t), " ")
}

// bulletPrefixRE matches the leading list-marker on a bullet-line, so it
// can be stripped before the entry is appended. Matches `- `, `* `,
// `1.`, `42)` style markers with arbitrary leading whitespace.
var bulletPrefixRE = regexp.MustCompile(`^\s*(?:[-*+]|\d+[\.\)])\s+`)

// extractBullets splits a section body into individual entries, one per
// bullet-marked line. Leading list-markers are stripped, surrounding
// whitespace is trimmed, blank lines are skipped, and continuation lines
// (lines indented further than their parent bullet) are joined into the
// parent's text with a single space.
//
// When the body contains NO bullet markers, returns nil — the section
// exists but has no enumerable entries.
func extractBullets(body string) []string {
	var entries []string
	var current strings.Builder
	flush := func() {
		s := strings.TrimSpace(current.String())
		if s != "" {
			entries = append(entries, s)
		}
		current.Reset()
	}

	for _, line := range strings.Split(body, "\n") {
		trimmedRight := strings.TrimRight(line, " \t\r")
		if strings.TrimSpace(trimmedRight) == "" {
			flush()
			continue
		}
		if bulletPrefixRE.MatchString(trimmedRight) {
			flush()
			stripped := bulletPrefixRE.ReplaceAllString(trimmedRight, "")
			current.WriteString(strings.TrimSpace(stripped))
			continue
		}
		// Continuation line: append to current with a single space.
		if current.Len() > 0 {
			current.WriteString(" ")
			current.WriteString(strings.TrimSpace(trimmedRight))
		}
	}
	flush()

	if len(entries) == 0 {
		return nil
	}
	return entries
}

// extractHyllaFeedbackBody returns the verbatim trimmed body of the
// "## Hylla Feedback" section, with one normalization: the canonical
// "None — Hylla answered everything needed." stanza (and its ASCII-hyphen
// variant) is collapsed to "" so the dispatcher's HyllaFeedback field
// distinguishes "no feedback recorded" from "explicit None stanza."
// Cascade-end aggregation only cares about non-empty bodies.
func extractHyllaFeedbackBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	// Match the canonical None-stanza in both em-dash and ASCII-hyphen
	// forms. The check is prefix-based so an agent that appends extra
	// commentary after the stanza retains the commentary.
	for _, prefix := range []string{
		"none — hylla answered everything needed",
		"none - hylla answered everything needed",
		"none. hylla answered everything needed",
	} {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	return trimmed
}
