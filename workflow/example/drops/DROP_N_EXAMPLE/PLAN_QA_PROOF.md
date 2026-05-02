# DROP_N — Plan QA Proof — Round 1

> **Pedagogical note.** This file demonstrates one plan-QA proof round. In a
> real drop, this file persists in tree as `PLAN_QA_PROOF.md` for round 1;
> a round 2 review would write a sibling file `PLAN_QA_PROOF_R2.md`, leaving
> round 1 visible. **Plan-QA files are never `git rm`d** — every adversarial
> round stays in the tree for audit.

**Verdict:** pass-with-notes (two minor findings; both accepted and folded into a Round 2 planner revision).

**Scope reviewed:** `DROP_N_EXAMPLE/PLAN.md` Round 1. Reviewed against `<PROJECT>/CLAUDE.md`, `<PROJECT>/PLAN.md`, `drops/WORKFLOW.md`, `AGENT_CASCADE_DESIGN.md`.

## Findings

### Finding 1 — Acceptance for N.1 defers the compile check, which is correct but unstated

- **Evidence:** N.1 acceptance bullet referred to a drop-end compile check via `mage ci`. This is correct per `CLAUDE.md` § "Build Verification" rule 2 (never raw `go build`). The deferral is right; what was missing in Round 1 was the positive confirmation the planner considered and chose it.
- **Claim:** the acceptance criterion is deferrable because the package gate (mage-backed) is the authoritative compile check, and it cannot exist until N.2.
- **Trace:** N.1 touches only files in `cmd/<PROJECT>/`. N.2 adds `magefile.go`. The `mage build` target that wraps `go build ./...` only becomes runnable at N.2's completion. A premature raw `go build` would contradict `CLAUDE.md`.
- **Conclusion:** the deferral is sound. Planner Round 2 should add one sentence explicitly noting the deferral is intentional, so a future reader does not mistake it for an omission.
- **Unknowns:** none.

### Finding 2 — `mage test` acceptance in N.2 is vague when "no tests yet"

- **Evidence:** N.2 Round 1 acceptance said "`mage test` succeeds (no tests yet, but the target runs clean)." This is ambiguous — `go test ./...` with no test files per package returns `? <pkg> [no test files]` which is an exit-0 state but noisy. A reader might interpret "succeeds" as "prints green" and be surprised by the `[no test files]` banner.
- **Claim:** "succeeds" should mean "exit code 0"; the banner is fine.
- **Trace:** go stdlib behavior documented in `go help test`. Exit codes 0 for pass (including "no test files"), 1 for fail, 2 for setup error. Verified by inspecting `go help test` output and exit-code semantics.
- **Conclusion:** planner Round 2 should clarify: "`mage test` exits 0 (no test files yet is expected — banner `[no test files]` per package is acceptable)". One additional sentence, no structural change to the droplet.
- **Unknowns:** none.

## Evidence Completeness Check

- All three droplets have `paths`, `packages`, `acceptance`, `blocked_by` per `drops/WORKFLOW.md` § "Phase 1 — Plan" step 3.
- `blocked_by` chain is explicit (`N.1 → N.2 → N.3`) and tight (no missing links; no spurious links).
- No droplets share a package — no implicit same-package blockers needed.
- Acceptance criteria are yes/no-verifiable: a QA agent can grep for `func main`, run `mage -l`, check workflow file YAML keys, and inspect `gh run watch` exit code.

## Route

Two findings, both small, both accepted. Dev confirmed in chat. Orch writes planner brief: "Planner Round 2: address finding 1 (one-sentence justification of the N.1 compile-check deferral) and finding 2 (clarify N.2 `mage test` exit-code semantics). No structural changes."

Orch leaves this file + `PLAN_QA_FALSIFICATION.md` in tree; re-spawns planner for Round 2; Phase 2 repeats against Round 2 by writing fresh `PLAN_QA_PROOF_R2.md` + `PLAN_QA_FALSIFICATION_R2.md` files. Round 2 passed clean — no further findings.
