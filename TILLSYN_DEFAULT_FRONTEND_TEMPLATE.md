# Default Frontend Builtin Template

Status: shipped builtin contract summary

Primary builtin source:
- [templates/builtin/default-frontend.json](/Users/evanschultz/Documents/Code/hylla/tillsyn/main/templates/builtin/default-frontend.json)

Builtin library id:
- `default-frontend`

Project kind:
- `frontend-project`

Additional reusable kinds referenced by the builtin:
- `visual-qa`
- `a11y-check`
- `design-review`

Shared phase and task kinds referenced by the builtin:
- `project-setup-phase`
- `plan-phase`
- `build-phase`
- `closeout-phase`
- `branch-cleanup-phase`
- `build-task`
- `refactor-phase`
- `dogfood-refactor-phase`
- `refactor-task`
- `dogfood-refactor-task`
- `qa-check`
- `commit-and-reingest`

Lifecycle summary:
- project creation auto-generates one root `PROJECT SETUP` phase
- each branch lane auto-generates `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`
- each generated phase also auto-generates `PHASE PUSH AND REINGEST CONFIRMATION`
- each `build-task` auto-generates:
  - `QA PROOF REVIEW`
  - `QA FALSIFICATION REVIEW`
  - `VISUAL QA`
  - `ACCESSIBILITY CHECK`
  - `COMMIT PUSH AND REINGEST`
- each `refactor-task` adds:
  - `PARITY VALIDATION IN ACTION`
  - `METRICS CAPTURE AND REPORT`
- each `dogfood-refactor-task` adds:
  - `TEST AGAINST DEV VERSION`
  - `CONFIRM LOCAL USED VERSION UPDATED`
  - `METRICS CAPTURE AND REPORT`
- for refactor and dogfood-refactor work, the builder should update the slice task description after QA and parity or dev validation with truthful `git diff` deltas, before/after repo and Hylla counts, timing windows, and cleanup/security findings, and the orchestrator should roll those values up into the parent phase description plus the report artifact

Design intent:
- generic frontend workflow contract, not Astro-specific and not design-tool-specific
- broad standards around semantic HTML, accessibility, responsive behavior, minimal browser JavaScript, and visual verification
- no implementation, cleanup, QA, parity-check, visual review, or repair work should happen outside explicit tasks or subtasks
- failing tests, CI, or QA should create a new explicit follow-up item before repair begins, including after a previously completed item needs more fixes
- refactor and dogfood-refactor are first-class default kinds, and their slice or phase metrics should stay truthful in task and phase descriptions plus the orchestrator report artifact
- tool-specific workflows such as Astro/Solid or Paper.design should live in specialized template libraries layered on top of this baseline, not in the builtin itself
