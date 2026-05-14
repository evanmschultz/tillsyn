# BUILD-QA-PROOF — W3.D2 (project delete/archive/restore/rename CLIs)

## Verdict: PASS

## Scope

W3.D2 specify block (PLAN.md L282-L353). Four lifecycle subcommands wired in `cmd/till/project_cli.go` with table tests in `cmd/till/project_cli_test.go`. `mage test-pkg ./cmd/till/...` returns 370/370 pass.

## Acceptance Criteria — Evidence + Verdict

### AC1: `till project delete --project-id <id> --confirm` calls `(*Service).DeleteProject` and writes a confirmation line on stdout.

**Evidence:** `cmd/till/project_cli.go:335-351`

```go
func runProjectDelete(ctx context.Context, svc *app.Service, cfg config.Config, opts projectDeleteCommandOptions, stdout io.Writer) error {
    if err := requireProjectID("project delete", opts.projectID); err != nil {
        return err
    }
    if svc == nil {
        return fmt.Errorf("app service is not configured")
    }
    if !opts.confirm {
        return fmt.Errorf("till project delete requires --confirm flag; hard delete is irreversible")
    }
    ctx = cliMutationContext(ctx, cfg)
    if err := svc.DeleteProject(ctx, opts.projectID); err != nil {
        return fmt.Errorf("delete project: %w", err)
    }
    _, err := fmt.Fprintf(stdout, "Project %q deleted.\n", opts.projectID)
    return err
}
```

Test: `TestRunProjectDelete_SuccessPath` (project_cli_test.go:1129-1154) runs with `confirm: true`, asserts the project is gone post-call, and that stdout contains either `project.ID` or `"deleted"`.

**Verdict:** PASS.

### AC2: `till project delete --project-id <id>` (missing `--confirm`) fails with a clear error explaining `--confirm` is required for hard-delete.

**Evidence:** `project_cli.go:342-344` — `if !opts.confirm { return fmt.Errorf("till project delete requires --confirm flag; hard delete is irreversible") }`.

Test: `TestRunProjectDelete_RequiresConfirm` (project_cli_test.go:1108-1125) asserts the error contains both `"--confirm"` and `"hard delete is irreversible"`.

**Verdict:** PASS.

### AC3: `till project archive --project-id <id>` calls `(*Service).ArchiveProject` and returns the archived project detail.

**Evidence:** `project_cli.go:354-367`

```go
func runProjectArchive(...) error {
    if err := requireProjectID("project archive", opts.projectID); err != nil { return err }
    if svc == nil { return fmt.Errorf("app service is not configured") }
    ctx = cliMutationContext(ctx, cfg)
    project, err := svc.ArchiveProject(ctx, opts.projectID)
    if err != nil { return fmt.Errorf("archive project: %w", err) }
    return writeProjectDetail(stdout, project, "Archived Project")
}
```

Test: `TestRunProjectArchive_ArchivesProject` (project_cli_test.go:1175-1191) asserts output contains both `"Archived Project"` title and the project name.

**Verdict:** PASS.

### AC4: `till project restore --project-id <id>` calls `(*Service).RestoreProject` and returns the restored project detail.

**Evidence:** `project_cli.go:370-383`

```go
func runProjectRestore(...) error {
    if err := requireProjectID("project restore", opts.projectID); err != nil { return err }
    if svc == nil { return fmt.Errorf("app service is not configured") }
    ctx = cliMutationContext(ctx, cfg)
    project, err := svc.RestoreProject(ctx, opts.projectID)
    if err != nil { return fmt.Errorf("restore project: %w", err) }
    return writeProjectDetail(stdout, project, "Restored Project")
}
```

Test: `TestRunProjectRestore_RestoresProject` (project_cli_test.go:1210-1231) archives first via `svc.ArchiveProject`, then asserts the restore output contains `"Restored Project"` and the project name.

**Verdict:** PASS.

### AC5: `till project rename --project-id <id> --name <new-name>` calls `(*Service).UpdateProject` with the new name (preserving all other existing fields) and returns the renamed project detail. New name must be non-empty.

**Evidence:** `project_cli.go:388-419`

```go
func runProjectRename(ctx context.Context, svc *app.Service, cfg config.Config, opts projectRenameCommandOptions, stdout io.Writer) error {
    if err := requireProjectID("project rename", opts.projectID); err != nil { return err }
    if svc == nil { return fmt.Errorf("app service is not configured") }
    if strings.TrimSpace(opts.newName) == "" {
        return fmt.Errorf("project rename requires --name <new-name>; new name cannot be empty")
    }
    existing, err := locateProjectForCLI(ctx, svc, opts.projectID, false, "project rename")
    if err != nil { return err }
    ctx = cliMutationContext(ctx, cfg)
    project, err := svc.UpdateProject(ctx, app.UpdateProjectInput{
        ProjectID:           opts.projectID,
        Name:                strings.TrimSpace(opts.newName),
        Description:         existing.Description,
        Metadata:            existing.Metadata,
        HyllaArtifactRef:    existing.HyllaArtifactRef,
        RepoBareRoot:        existing.RepoBareRoot,
        RepoPrimaryWorktree: existing.RepoPrimaryWorktree,
        Language:            existing.Language,
        BuildTool:           existing.BuildTool,
        DevMcpServerName:    existing.DevMcpServerName,
    })
    if err != nil { return fmt.Errorf("rename project: %w", err) }
    return writeProjectDetail(stdout, project, "Renamed Project")
}
```

The function: (1) trims and rejects empty newName, (2) calls `locateProjectForCLI` to fetch the existing project, (3) constructs `UpdateProjectInput` copying every first-class field (Description, Metadata, HyllaArtifactRef, RepoBareRoot, RepoPrimaryWorktree, Language, BuildTool, DevMcpServerName) from `existing`, with only `Name` overridden. This matches the RiskNotes constraint at PLAN.md L306 verbatim.

Test: `TestRunProjectRename_PreservesAllOtherFields` (project_cli_test.go:1286-1350) seeds a fully-populated project (description, HyllaArtifactRef, RepoBareRoot, RepoPrimaryWorktree, Language=go, BuildTool=mage, DevMcpServerName=tillsyn-dev, Metadata.Owner=original-owner), runs rename to "RenamedProject", then reads back and asserts each preserved field is unchanged.

**Verdict:** PASS.

### AC6: `till project rename --project-id <id>` (missing `--name`) fails with a clear error.

**Evidence:** `project_cli.go:395-397` — `if strings.TrimSpace(opts.newName) == "" { return fmt.Errorf("project rename requires --name <new-name>; new name cannot be empty") }`.

Test: `TestRunProjectRename_MissingNameReturnsError` (project_cli_test.go:1250-1265) asserts the error contains `"--name"`.

**Verdict:** PASS.

### AC7: `mage test-pkg ./cmd/till/...` passes.

**Evidence:** `mage test-pkg ./cmd/till/...` output: `370 tests passed across 1 package. [SUCCESS] All tests passed`.

**Verdict:** PASS.

## Cross-Cutting Checks

### Mutation context propagation

All four functions (`runProjectDelete:345`, `runProjectArchive:361`, `runProjectRestore:377`, `runProjectRename:402`) call `ctx = cliMutationContext(ctx, cfg)` BEFORE the corresponding `(*Service)` call. This matches the ContextBlocks reference at PLAN.md L323-L327.

**Verdict:** PASS.

### Missing-project-id discovery error

Spec requires "missing-project-id discovery error per command" per builder claim. Verified four dedicated tests:

- `TestRunProjectDelete_MissingProjectIDReturnsDiscoveryError` (project_cli_test.go:1158-1171)
- `TestRunProjectArchive_MissingProjectIDReturnsDiscoveryError` (project_cli_test.go:1195-1206)
- `TestRunProjectRestore_MissingProjectIDReturnsDiscoveryError` (project_cli_test.go:1235-1246)
- `TestRunProjectRename_MissingProjectIDReturnsDiscoveryError` (project_cli_test.go:1269-1282)

Each asserts the error contains `"--project-id is required"`.

**Verdict:** PASS.

### Options structs match KindPayload shape_hint

- `projectDeleteCommandOptions` (project_cli.go:312-315): `{projectID, confirm}` — matches.
- `projectArchiveCommandOptions` (project_cli.go:318-320): `{projectID}` — matches.
- `projectRestoreCommandOptions` (project_cli.go:323-325): `{projectID}` — matches.
- `projectRenameCommandOptions` (project_cli.go:328-331): `{projectID, newName}` — matches.

**Verdict:** PASS.

### writeProjectDetail title strings

- delete: stdout-string `"Project %q deleted.\n"` (compact one-liner, no title because no project detail returned post-delete).
- archive: title `"Archived Project"` (project_cli.go:366).
- restore: title `"Restored Project"` (project_cli.go:382).
- rename: title `"Renamed Project"` (project_cli.go:418).

All three project-detail returns surface the post-mutation state via `writeProjectDetail`, which renders all first-class fields per writeProjectDetail (project_cli.go:544-566).

**Verdict:** PASS.

### Error wrapping with %w

`svc.DeleteProject` (line 347), `svc.ArchiveProject` (line 364), `svc.RestoreProject` (line 380), `svc.UpdateProject` (line 416) errors are wrapped via `fmt.Errorf("... %w", err)`.

**Verdict:** PASS.

## Falsification Attempts (Counterexamples Considered + Rejected)

1. **Confirm bypass via flag default?** Spec requires `--confirm` flag default false (cobra bool). The code path treats `opts.confirm == false` as the rejection case before any service call. Default zero-value of bool is false. **No bypass.**
2. **Rename clobbers Metadata.Tags or Metadata.Groups?** Rename passes `existing.Metadata` as a whole struct (line 407) — every metadata field travels through. `TestRunProjectRename_PreservesAllOtherFields` asserts Owner preservation; Metadata is a value-typed copy, so Tags/Groups/Icon/Color/Homepage all flow through too. **No clobber.**
3. **Archive of an already-archived project?** Service-level concern; CLI delegates to `svc.ArchiveProject` and wraps the returned error. CLI does not have to re-check; service is the authority.
4. **Restore of an active (non-archived) project?** Same — service-level concern, CLI propagates error.
5. **`cliMutationContext` ordering for rename?** Rename calls `locateProjectForCLI` BEFORE `cliMutationContext`. `locateProjectForCLI` is a read path (list projects); mutation context attaches identity to the WRITE (`UpdateProject`). Ordering is correct: read pre-mutation-ctx, write post-mutation-ctx. **No issue.**
6. **`newName` trim asymmetry?** Empty-name guard at line 395 trims before checking. Service call at line 405 also trims. If newName is `"   "`, the guard rejects before reaching `UpdateProject`. **Consistent.**

## NITs

None.

## Hard Rules Compliance

- Hylla OFF: no hylla_* tool used, no `## Hylla Feedback` section.
- mage targets only: `mage test-pkg ./cmd/till/...` used to verify; no raw `go test`.
- No code modification: read-only QA.
- ASCII apostrophe only: verified in this MD body.
- No closeout MD rollups: this file is the verdict only.

## Summary

All 7 acceptance criteria + 5 cross-cutting checks PASS. 6 falsification attempts considered and rejected. 10 new test functions confirmed:

1. `TestRunProjectDelete_RequiresConfirm`
2. `TestRunProjectDelete_SuccessPath`
3. `TestRunProjectDelete_MissingProjectIDReturnsDiscoveryError`
4. `TestRunProjectArchive_ArchivesProject`
5. `TestRunProjectArchive_MissingProjectIDReturnsDiscoveryError`
6. `TestRunProjectRestore_RestoresProject`
7. `TestRunProjectRestore_MissingProjectIDReturnsDiscoveryError`
8. `TestRunProjectRename_MissingNameReturnsError`
9. `TestRunProjectRename_MissingProjectIDReturnsDiscoveryError`
10. `TestRunProjectRename_PreservesAllOtherFields`

**VERDICT: PASS.**
