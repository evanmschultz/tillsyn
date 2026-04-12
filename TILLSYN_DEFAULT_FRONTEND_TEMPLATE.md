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
- `qa-check`
- `commit-and-reingest`

Lifecycle summary:
- project creation auto-generates one root `PROJECT SETUP` phase
- each branch lane auto-generates `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`
- each `build-task` auto-generates:
  - `QA PROOF REVIEW`
  - `QA FALSIFICATION REVIEW`
  - `VISUAL QA`
  - `ACCESSIBILITY CHECK`
  - `COMMIT AND REINGEST`

Design intent:
- generic frontend workflow contract, not Astro-specific and not design-tool-specific
- broad standards around semantic HTML, accessibility, responsive behavior, minimal browser JavaScript, and visual verification
- tool-specific workflows such as Astro/Solid or Paper.design should live in specialized template libraries layered on top of this baseline, not in the builtin itself
