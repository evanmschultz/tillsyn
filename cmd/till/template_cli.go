package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/evanmschultz/laslig"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

type childRuleRow struct {
	nodeTemplateName string
	nodeTemplateID   string
	rule             domain.TemplateChildRule
}

// writeTemplateLibraryList renders template libraries as a stable human-readable table.
func writeTemplateLibraryList(stdout io.Writer, libraries []domain.TemplateLibrary) error {
	ordered := append([]domain.TemplateLibrary(nil), libraries...)
	slices.SortFunc(ordered, compareTemplateLibrariesForCLI)
	rows := make([][]string, 0, len(ordered))
	for _, library := range ordered {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(library.Name, library.ID),
			firstNonEmptyTrimmed(library.ID, "-"),
			fmt.Sprintf("%d", max(library.Revision, 1)),
			firstNonEmptyTrimmed(string(library.Scope), "-"),
			firstNonEmptyTrimmed(library.ProjectID, "-"),
			firstNonEmptyTrimmed(string(library.Status), "-"),
			firstNonEmptyTrimmed(library.SourceLibraryID, "-"),
			fmt.Sprintf("%d", len(library.NodeTemplates)),
		})
	}
	return writeCLITable(
		stdout,
		"Template Libraries",
		[]string{"NAME", "LIBRARY ID", "REV", "SCOPE", "PROJECT", "STATUS", "SOURCE", "NODE TEMPLATES"},
		rows,
		"No template libraries found.",
	)
}

// writeTemplateLibraryDetail renders one template library and its nested rules in a stable human-readable view.
func writeTemplateLibraryDetail(stdout io.Writer, library domain.TemplateLibrary) error {
	printer := newCLIPrinter(stdout)
	rows := [][2]string{
		{"id", firstNonEmptyTrimmed(library.ID, "-")},
		{"name", firstNonEmptyTrimmed(library.Name, "-")},
		{"scope", firstNonEmptyTrimmed(string(library.Scope), "-")},
		{"project", firstNonEmptyTrimmed(library.ProjectID, "-")},
		{"status", firstNonEmptyTrimmed(string(library.Status), "-")},
		{"revision", fmt.Sprintf("%d", max(library.Revision, 1))},
		{"builtin managed", yesNo(library.BuiltinManaged)},
		{"builtin source", firstNonEmptyTrimmed(library.BuiltinSource, "-")},
		{"builtin version", firstNonEmptyTrimmed(library.BuiltinVersion, "-")},
		{"source library", firstNonEmptyTrimmed(library.SourceLibraryID, "-")},
		{"description", firstNonEmptyTrimmed(compactText(library.Description), "-")},
		{"node templates", fmt.Sprintf("%d", len(library.NodeTemplates))},
		{"created by", templateAuditActorLabel(library.CreatedByActorName, library.CreatedByActorID, library.CreatedByActorType)},
		{"created at", formatAuthTime(library.CreatedAt)},
		{"updated at", formatAuthTime(library.UpdatedAt)},
		{"approved by", templateAuditActorLabel(library.ApprovedByActorName, library.ApprovedByActorID, library.ApprovedByActorType)},
		{"approved at", formatAuthOptionalTime(library.ApprovedAt)},
	}
	if err := writeCLIKVWithPrinter(printer, "Template Library", rows); err != nil {
		return err
	}
	if err := writeTemplateNodeTemplateTable(printer, library.NodeTemplates); err != nil {
		return err
	}
	if err := writeTemplateChildRuleTable(printer, library.NodeTemplates); err != nil {
		return err
	}
	return nil
}

// writeBuiltinTemplateLibraryStatusDetail renders one builtin template lifecycle summary in a stable human-readable view.
func writeBuiltinTemplateLibraryStatusDetail(stdout io.Writer, status domain.BuiltinTemplateLibraryStatus) error {
	return writeCLIKV(stdout, "Builtin Template Library", [][2]string{
		{"library id", firstNonEmptyTrimmed(status.LibraryID, "-")},
		{"name", firstNonEmptyTrimmed(status.Name, "-")},
		{"state", firstNonEmptyTrimmed(string(status.State), "-")},
		{"builtin source", firstNonEmptyTrimmed(status.BuiltinSource, "-")},
		{"builtin version", firstNonEmptyTrimmed(status.BuiltinVersion, "-")},
		{"builtin digest", firstNonEmptyTrimmed(status.BuiltinRevisionDigest, "-")},
		{"required kinds", renderKindIDs(status.RequiredKindIDs)},
		{"missing kinds", renderKindIDs(status.MissingKindIDs)},
		{"installed", yesNo(status.Installed)},
		{"installed name", firstNonEmptyTrimmed(status.InstalledLibraryName, "-")},
		{"installed status", firstNonEmptyTrimmed(string(status.InstalledStatus), "-")},
		{"installed revision", firstNonEmptyTrimmed(renderOptionalPositiveInt(status.InstalledRevision), "-")},
		{"installed digest", firstNonEmptyTrimmed(status.InstalledDigest, "-")},
		{"installed builtin", yesNo(status.InstalledBuiltin)},
		{"installed updated at", formatAuthOptionalTime(status.InstalledUpdatedAt)},
	})
}

// writeBuiltinTemplateLibraryEnsureDetail renders one explicit builtin ensure result plus the resulting status.
func writeBuiltinTemplateLibraryEnsureDetail(stdout io.Writer, result domain.BuiltinTemplateLibraryEnsureResult) error {
	printer := newCLIPrinter(stdout)
	if err := writeCLIKVWithPrinter(printer, "Builtin Template Ensure", [][2]string{
		{"changed", yesNo(result.Changed)},
		{"library id", firstNonEmptyTrimmed(result.Library.ID, "-")},
		{"revision", firstNonEmptyTrimmed(renderOptionalPositiveInt(result.Library.Revision), "-")},
		{"status", firstNonEmptyTrimmed(string(result.Library.Status), "-")},
		{"builtin version", firstNonEmptyTrimmed(result.Library.BuiltinVersion, "-")},
	}); err != nil {
		return err
	}
	return writeBuiltinTemplateLibraryStatusDetail(stdout, result.Status)
}

// writeProjectTemplateBindingDetail renders one project binding in a stable human-readable detail block.
func writeProjectTemplateBindingDetail(stdout io.Writer, binding domain.ProjectTemplateBinding) error {
	return writeCLIKV(stdout, "Project Template Binding", [][2]string{
		{"project id", firstNonEmptyTrimmed(binding.ProjectID, "-")},
		{"library id", firstNonEmptyTrimmed(binding.LibraryID, "-")},
		{"library name", firstNonEmptyTrimmed(binding.LibraryName, "-")},
		{"bound revision", fmt.Sprintf("%d", max(binding.BoundRevision, 1))},
		{"drift status", firstNonEmptyTrimmed(binding.DriftStatus, "-")},
		{"latest revision", firstNonEmptyTrimmed(renderOptionalPositiveInt(binding.LatestRevision), "-")},
		{"bound by", templateAuditActorLabel(binding.BoundByActorName, binding.BoundByActorID, binding.BoundByActorType)},
		{"bound at", formatAuthTime(binding.BoundAt)},
	})
}

// writeProjectTemplateReapplyPreviewDetail renders one explicit template reapply preview in a stable human-readable view.
func writeProjectTemplateReapplyPreviewDetail(stdout io.Writer, preview domain.ProjectTemplateReapplyPreview) error {
	printer := newCLIPrinter(stdout)
	if err := writeCLIKVWithPrinter(printer, "Project Template Reapply Preview", [][2]string{
		{"project id", firstNonEmptyTrimmed(preview.ProjectID, "-")},
		{"library id", firstNonEmptyTrimmed(preview.LibraryID, "-")},
		{"library name", firstNonEmptyTrimmed(preview.LibraryName, "-")},
		{"drift status", firstNonEmptyTrimmed(preview.DriftStatus, "-")},
		{"bound revision", firstNonEmptyTrimmed(renderOptionalPositiveInt(preview.BoundRevision), "-")},
		{"latest revision", firstNonEmptyTrimmed(renderOptionalPositiveInt(preview.LatestRevision), "-")},
		{"eligible migrations", firstNonEmptyTrimmed(renderOptionalPositiveInt(preview.EligibleMigrationCount), "0")},
		{"ineligible migrations", firstNonEmptyTrimmed(renderOptionalPositiveInt(preview.IneligibleMigrationCount), "0")},
		{"review required", yesNo(preview.ReviewRequired)},
	}); err != nil {
		return err
	}
	if err := writeProjectTemplateDefaultChangeTable(printer, preview.ProjectDefaultChanges); err != nil {
		return err
	}
	if err := writeProjectTemplateChildRuleChangeTable(printer, preview.ChildRuleChanges); err != nil {
		return err
	}
	return writeProjectTemplateMigrationCandidateTable(printer, preview.MigrationCandidates)
}

// writeProjectTemplateMigrationApprovalResultDetail renders one explicit migration-approval result in a stable human-readable view.
func writeProjectTemplateMigrationApprovalResultDetail(stdout io.Writer, result domain.ProjectTemplateMigrationApprovalResult) error {
	printer := newCLIPrinter(stdout)
	if err := writeCLIKVWithPrinter(printer, "Project Template Migration Approval", [][2]string{
		{"project id", firstNonEmptyTrimmed(result.ProjectID, "-")},
		{"library id", firstNonEmptyTrimmed(result.LibraryID, "-")},
		{"library name", firstNonEmptyTrimmed(result.LibraryName, "-")},
		{"drift status", firstNonEmptyTrimmed(result.DriftStatus, "-")},
		{"approved all", yesNo(result.ApprovedAll)},
		{"applied count", firstNonEmptyTrimmed(renderOptionalPositiveInt(result.AppliedCount), "0")},
		{"remaining eligible", firstNonEmptyTrimmed(renderOptionalPositiveInt(result.RemainingEligibleCount), "0")},
		{"remaining ineligible", firstNonEmptyTrimmed(renderOptionalPositiveInt(result.RemainingIneligibleCount), "0")},
	}); err != nil {
		return err
	}
	if len(result.Approvals) == 0 {
		return writeCLIPanelWithPrinter(printer, "Approved Migrations", "No template migrations were applied.", "")
	}
	rows := make([][]string, 0, len(result.Approvals))
	for _, approval := range result.Approvals {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(approval.Title, approval.TaskID, "-"),
			firstNonEmptyTrimmed(approval.TaskID, "-"),
			firstNonEmptyTrimmed(renderAuthStringList(approval.ChangeKinds), "-"),
			firstNonEmptyTrimmed(compactText(approval.NewTitle), "-"),
		})
	}
	return writeCLITableWithPrinter(
		printer,
		"Approved Migrations",
		[]string{"TASK", "TASK ID", "CHANGES", "NEW TITLE"},
		rows,
		"No template migrations were applied.",
	)
}

// writeNodeContractSnapshotDetail renders one generated-node contract snapshot in a stable human-readable detail block.
func writeNodeContractSnapshotDetail(stdout io.Writer, snapshot domain.NodeContractSnapshot) error {
	return writeCLIKV(stdout, "Node Contract", [][2]string{
		{"node id", firstNonEmptyTrimmed(snapshot.NodeID, "-")},
		{"project id", firstNonEmptyTrimmed(snapshot.ProjectID, "-")},
		{"source library", firstNonEmptyTrimmed(snapshot.SourceLibraryID, "-")},
		{"source node template", firstNonEmptyTrimmed(snapshot.SourceNodeTemplateID, "-")},
		{"source child rule", firstNonEmptyTrimmed(snapshot.SourceChildRuleID, "-")},
		{"generated by", templateAuditActorLabel("", snapshot.CreatedByActorID, snapshot.CreatedByActorType)},
		{"responsible actor", firstNonEmptyTrimmed(string(snapshot.ResponsibleActorKind), "-")},
		{"editable by", renderTemplateActorKinds(snapshot.EditableByActorKinds)},
		{"completable by", renderTemplateActorKinds(snapshot.CompletableByActorKinds)},
		{"orchestrator may complete", yesNo(snapshot.OrchestratorMayComplete)},
		{"required for parent done", yesNo(snapshot.RequiredForParentDone)},
		{"required for containing done", yesNo(snapshot.RequiredForContainingDone)},
		{"created at", formatAuthTime(snapshot.CreatedAt)},
	})
}

func renderOptionalPositiveInt(v int) string {
	if v <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", v)
}

func renderKindIDs(kindIDs []domain.KindID) string {
	if len(kindIDs) == 0 {
		return "-"
	}
	values := make([]string, 0, len(kindIDs))
	for _, kindID := range kindIDs {
		values = append(values, string(kindID))
	}
	return renderAuthStringList(values)
}

func writeProjectTemplateDefaultChangeTable(printer *laslig.Printer, changes []domain.ProjectTemplateDefaultChange) error {
	if len(changes) == 0 {
		return writeCLIPanelWithPrinter(printer, "Project Default Changes", "No project-level default changes detected.", "")
	}
	rows := make([][]string, 0, len(changes))
	for _, change := range changes {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(change.Field, "-"),
			firstNonEmptyTrimmed(compactText(change.Previous), "-"),
			firstNonEmptyTrimmed(compactText(change.Current), "-"),
		})
	}
	return writeCLITableWithPrinter(
		printer,
		"Project Default Changes",
		[]string{"FIELD", "BOUND REVISION", "LATEST REVISION"},
		rows,
		"No project-level default changes detected.",
	)
}

func writeProjectTemplateChildRuleChangeTable(printer *laslig.Printer, changes []domain.ProjectTemplateChildRuleChange) error {
	if len(changes) == 0 {
		return writeCLIPanelWithPrinter(printer, "Child Rule Changes", "No generated child-rule changes detected.", "")
	}
	rows := make([][]string, 0, len(changes))
	for _, change := range changes {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(change.NodeTemplateName, change.NodeTemplateID),
			firstNonEmptyTrimmed(change.ChildRuleID, "-"),
			renderAuthStringList(change.ChangeKinds),
			firstNonEmptyTrimmed(change.PreviousTitleTemplate, "-"),
			firstNonEmptyTrimmed(change.CurrentTitleTemplate, "-"),
		})
	}
	return writeCLITableWithPrinter(
		printer,
		"Child Rule Changes",
		[]string{"NODE TEMPLATE", "RULE ID", "CHANGES", "BOUND TITLE", "LATEST TITLE"},
		rows,
		"No generated child-rule changes detected.",
	)
}

func writeProjectTemplateMigrationCandidateTable(printer *laslig.Printer, candidates []domain.ProjectTemplateMigrationCandidate) error {
	if len(candidates) == 0 {
		return writeCLIPanelWithPrinter(printer, "Migration Candidates", "No existing generated nodes are affected by the current drift.", "")
	}
	rows := make([][]string, 0, len(candidates))
	for _, candidate := range candidates {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(candidate.Title, candidate.TaskID),
			firstNonEmptyTrimmed(candidate.TaskID, "-"),
			firstNonEmptyTrimmed(string(candidate.Status), "-"),
			firstNonEmptyTrimmed(renderAuthStringList(candidate.ChangeKinds), "-"),
			firstNonEmptyTrimmed(compactText(candidate.Reason), "-"),
		})
	}
	return writeCLITableWithPrinter(
		printer,
		"Migration Candidates",
		[]string{"TASK", "TASK ID", "STATUS", "CHANGES", "REASON"},
		rows,
		"No existing generated nodes are affected by the current drift.",
	)
}

func writeTemplateNodeTemplateTable(printer *laslig.Printer, nodeTemplates []domain.NodeTemplate) error {
	if len(nodeTemplates) == 0 {
		return writeCLIPanelWithPrinter(printer, "Node Templates", "No node templates defined.", "")
	}
	ordered := append([]domain.NodeTemplate(nil), nodeTemplates...)
	slices.SortFunc(ordered, compareNodeTemplatesForCLI)
	rows := make([][]string, 0, len(ordered))
	for _, nodeTemplate := range ordered {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(nodeTemplate.DisplayName, nodeTemplate.ID),
			firstNonEmptyTrimmed(nodeTemplate.ID, "-"),
			firstNonEmptyTrimmed(string(nodeTemplate.ScopeLevel), "-"),
			firstNonEmptyTrimmed(string(nodeTemplate.NodeKindID), "-"),
			fmt.Sprintf("%d", len(nodeTemplate.ChildRules)),
		})
	}
	return writeCLITableWithPrinter(
		printer,
		"Node Templates",
		[]string{"NAME", "TEMPLATE ID", "SCOPE", "NODE KIND", "CHILD RULES"},
		rows,
		"No node templates defined.",
	)
}

func writeTemplateChildRuleTable(printer *laslig.Printer, nodeTemplates []domain.NodeTemplate) error {
	rowsData := make([]childRuleRow, 0)
	for _, nodeTemplate := range nodeTemplates {
		for _, childRule := range nodeTemplate.ChildRules {
			rowsData = append(rowsData, childRuleRow{
				nodeTemplateName: firstNonEmptyTrimmed(nodeTemplate.DisplayName, nodeTemplate.ID),
				nodeTemplateID:   nodeTemplate.ID,
				rule:             childRule,
			})
		}
	}
	if len(rowsData) == 0 {
		return writeCLIPanelWithPrinter(printer, "Template Child Rules", "No child rules defined.", "")
	}
	slices.SortFunc(rowsData, compareTemplateChildRuleRowsForCLI)
	rows := make([][]string, 0, len(rowsData))
	for _, row := range rowsData {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(row.nodeTemplateName, row.nodeTemplateID),
			firstNonEmptyTrimmed(row.rule.ID, "-"),
			firstNonEmptyTrimmed(string(row.rule.ChildScopeLevel), "-"),
			firstNonEmptyTrimmed(string(row.rule.ChildKindID), "-"),
			firstNonEmptyTrimmed(row.rule.TitleTemplate, "-"),
			firstNonEmptyTrimmed(string(row.rule.ResponsibleActorKind), "-"),
			renderTemplateActorKinds(row.rule.EditableByActorKinds),
			renderTemplateActorKinds(row.rule.CompletableByActorKinds),
			yesNo(row.rule.RequiredForParentDone),
			yesNo(row.rule.RequiredForContainingDone),
		})
	}
	return writeCLITableWithPrinter(
		printer,
		"Template Child Rules",
		[]string{"NODE TEMPLATE", "RULE ID", "CHILD SCOPE", "CHILD KIND", "TITLE", "RESPONSIBLE", "EDITABLE BY", "COMPLETABLE BY", "PARENT BLOCKER", "CONTAINING BLOCKER"},
		rows,
		"No child rules defined.",
	)
}

func compareTemplateLibrariesForCLI(a, b domain.TemplateLibrary) int {
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.Name)), strings.ToLower(strings.TrimSpace(b.Name))); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.ID)), strings.ToLower(strings.TrimSpace(b.ID))); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.ProjectID)), strings.ToLower(strings.TrimSpace(b.ProjectID))); cmp != 0 {
		return cmp
	}
	return strings.Compare(strings.ToLower(strings.TrimSpace(string(a.Scope))), strings.ToLower(strings.TrimSpace(string(b.Scope))))
}

func compareNodeTemplatesForCLI(a, b domain.NodeTemplate) int {
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.DisplayName)), strings.ToLower(strings.TrimSpace(b.DisplayName))); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.ID)), strings.ToLower(strings.TrimSpace(b.ID))); cmp != 0 {
		return cmp
	}
	return strings.Compare(strings.ToLower(strings.TrimSpace(string(a.NodeKindID))), strings.ToLower(strings.TrimSpace(string(b.NodeKindID))))
}

func compareTemplateChildRuleRowsForCLI(a, b childRuleRow) int {
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.nodeTemplateName)), strings.ToLower(strings.TrimSpace(b.nodeTemplateName))); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(strings.ToLower(strings.TrimSpace(a.rule.TitleTemplate)), strings.ToLower(strings.TrimSpace(b.rule.TitleTemplate))); cmp != 0 {
		return cmp
	}
	return strings.Compare(strings.ToLower(strings.TrimSpace(a.rule.ID)), strings.ToLower(strings.TrimSpace(b.rule.ID)))
}

func renderTemplateActorKinds(kinds []domain.TemplateActorKind) string {
	if len(kinds) == 0 {
		return "-"
	}
	values := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		values = append(values, string(kind))
	}
	return renderAuthStringList(values)
}

func templateAuditActorLabel(name, id string, actorType domain.ActorType) string {
	label := firstNonEmptyTrimmed(name, id)
	if label == "" {
		label = "-"
	}
	if actorType == "" {
		return label
	}
	return label + " (" + string(actorType) + ")"
}
