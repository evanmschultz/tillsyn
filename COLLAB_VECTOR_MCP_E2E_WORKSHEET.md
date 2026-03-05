# Collaborative Vector + MCP E2E Worksheet

Created: 2026-03-04
Status: Active
Owner: User + Codex (single-writer updates by Codex)

## 1) Purpose

Validate the vector-search wave end-to-end together, with:
1. user-driven TUI validation,
2. agent-driven MCP validation,
3. shared pass/fail decisions,
4. strict fail-stop remediation loop before moving to the next item.

## 2) Locked Collaboration Protocol

For every failing step, use this exact sequence before any next step:
1. Stop progression immediately on first fail.
2. Record the user's exact wording in this worksheet.
3. Spawn subagents to investigate and propose fix options.
4. Discuss options with user and reach explicit consensus.
5. Implement fix with scoped worker subagents.
6. Run required package tests (`just test-pkg ...`) and required gates as applicable.
7. Run independent QA pass 1 and QA pass 2.
8. Re-run the same collaborative step and capture fresh evidence.
9. Only then mark that step complete and proceed.

## 3) Evidence Destinations

Primary collaborative evidence files:
1. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` (this file)
2. `COLLAB_TEST_2026-03-02_DOGFOOD.md` (collaborative dogfood record)
3. `MCP_DOGFOODING_WORKSHEET.md` (MCP transport corroboration)

Command/test artifacts:
1. `.tmp/vec-collab-e2e-<timestamp>/...`
2. `.tmp/vec-wavef-evidence/20260303_175936/...`
3. `.tmp/vec-wavef-evidence/20260303_180827/...`

## 4) Session Setup

### 4.1 Agent Preflight

| ID | Command | Expected | Status | Evidence | Notes |
|---|---|---|---|---|---|
| S-01 | `just build` | Build succeeds | PASS | `.tmp/vec-collab-e2e-20260304_191626/just_build.txt` | Initial sandbox run hit cache permission; rerun outside sandbox passed. |
| S-02 | `./till serve --help` | Serve help is visible | PASS | `.tmp/vec-collab-e2e-20260304_191626/till_serve_help.txt` | Help output includes serve flags and endpoints. |
| S-03 | `just check` | PASS | PASS | `.tmp/vec-collab-e2e-20260304_191626/just_check.txt` | Cross-package check suite passed for this session. |

### 4.2 User Runtime Setup

| ID | Action | Expected | Status | Evidence | Notes |
|---|---|---|---|---|---|
| U-01 | Start TUI runtime (`just run` or built binary flow) | App opens without panic | PENDING_USER | | |
| U-02 | Open project/board that contains vector-indexed tasks | Board loads with tasks | PENDING_USER | | |

## 5) Collaborative TUI E2E Queue (Run In Order)

### Section T1: Metadata Accessibility (Wave E)

| ID | Step | Expected | Status | User Exact Wording | Evidence | Notes |
|---|---|---|---|---|---|---|
| T1-01 | Open edit task form for existing task | `objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`, `blocked_reason` are visible/editable | PENDING_USER | | | |
| T1-02 | Save updates for all above fields | Values persist after save/re-open | PENDING_USER | | | |
| T1-03 | Open task info overlay | All above fields render in info view | PENDING_USER | | | |

### Section T2: TUI Search Behavior

| ID | Step | Expected | Status | User Exact Wording | Evidence | Notes |
|---|---|---|---|---|---|---|
| T2-01 | Run task search for known keyword in title/description | Relevant matches returned and stable ordering | PENDING_USER | | | |
| T2-02 | Run search for metadata text (`objective` etc.) | Match includes task containing metadata phrase | PENDING_USER | | | |
| T2-03 | Navigate multi-result search pages | Deterministic behavior with explicit limit/offset defaults | PENDING_USER | | | |
| T2-04 | Use dependency inspector search | Results remain consistent with explicit mode/sort/limit/offset defaults | PENDING_USER | | | |

### Section T3: Regression Safety in TUI Flows

| ID | Step | Expected | Status | User Exact Wording | Evidence | Notes |
|---|---|---|---|---|---|---|
| T3-01 | Edit task metadata, then open thread/comments flow | No overlay/layout regression | PENDING_USER | | | |
| T3-02 | Switch projects/scopes and repeat search | No stale/incorrect search carryover | PENDING_USER | | | |

## 6) Collaborative MCP E2E Queue (Run In Order)

### Section M1: Tool Schema + Guardrails

| ID | MCP Check | Expected | Status | User Exact Wording | Evidence | Notes |
|---|---|---|---|---|---|---|
| M1-01 | `till.search_task_matches` tool schema inspection | Contains `mode`, `sort`, `levels`, `kinds`, `labels_any`, `labels_all`, `limit`, `offset` | PENDING_AGENT | | | |
| M1-02 | Schema numeric constraints | `limit` default 50, min 0, max 200; `offset` default 0, min 0 | PENDING_AGENT | | | |
| M1-03 | Invalid pagination input check | Invalid values fail with deterministic validation behavior | PENDING_AGENT | | | |

### Section M2: Query Mode Behavior

| ID | MCP Check | Expected | Status | User Exact Wording | Evidence | Notes |
|---|---|---|---|---|---|---|
| M2-01 | `mode=keyword` call | Returns lexical matches for query | PENDING_AGENT | | | |
| M2-02 | `mode=semantic` call | Returns semantic matches or keyword fallback when semantic unavailable | PENDING_AGENT | | | |
| M2-03 | `mode=hybrid` call | Combined behavior with stable ranking response shape | PENDING_AGENT | | | |

### Section M3: Filters, Sorting, Pagination

| ID | MCP Check | Expected | Status | User Exact Wording | Evidence | Notes |
|---|---|---|---|---|---|---|
| M3-01 | `levels` + `kinds` filters | Result set constrained correctly | PENDING_AGENT | | | |
| M3-02 | `labels_any` + `labels_all` filters | Taxonomy filter behavior correct | PENDING_AGENT | | | |
| M3-03 | `sort=rank_desc|title_asc|created_at_desc|updated_at_desc` | Sort order deterministic and valid | PENDING_AGENT | | | |
| M3-04 | `limit` + `offset` paging calls | Stable slices of total candidate set | PENDING_AGENT | | | |

## 7) Findings + Remediation Ledger

| Finding ID | Section/Step | Severity | User Exact Wording | Agent Rephrase | Decision | Status | Evidence |
|---|---|---|---|---|---|---|---|
| FR-001 | | | | | | OPEN | |

## 8) Subagent Fix Planning Record (Populate On First Failure)

| Fix ID | Finding ID | Planning Subagents | Candidate Options | User-Selected Option | Notes |
|---|---|---|---|---|---|
| FX-001 | | | | | |

## 9) Validation After Fix Record

| Fix ID | Package Tests | QA Pass 1 | QA Pass 2 | Collaborative Re-test Step(s) | Final Result |
|---|---|---|---|---|---|
| FX-001 | | | | | |

## 10) Sign-Off

- Collaborative TUI queue complete: `PENDING`
- Collaborative MCP queue complete: `PENDING`
- Open High findings: `PENDING`
- Open Medium findings: `PENDING`
- Final user+agent dogfood verdict: `PENDING`
