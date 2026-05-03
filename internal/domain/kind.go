package domain

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// KindID identifies one reusable kind definition in the global kind catalog.
type KindID string

// Kind represents the closed 12-value enum of action-item kinds.
type Kind string

// Built-in kind values. Scope mirrors kind per row.
const (
	KindPlan                 Kind = "plan"
	KindResearch             Kind = "research"
	KindBuild                Kind = "build"
	KindPlanQAProof          Kind = "plan-qa-proof"
	KindPlanQAFalsification  Kind = "plan-qa-falsification"
	KindBuildQAProof         Kind = "build-qa-proof"
	KindBuildQAFalsification Kind = "build-qa-falsification"
	KindCloseout             Kind = "closeout"
	KindCommit               Kind = "commit"
	KindRefinement           Kind = "refinement"
	KindDiscussion           Kind = "discussion"
	KindHumanVerify          Kind = "human-verify"
)

// validKinds stores every member of the closed 12-value Kind enum.
var validKinds = []Kind{
	KindPlan,
	KindResearch,
	KindBuild,
	KindPlanQAProof,
	KindPlanQAFalsification,
	KindBuildQAProof,
	KindBuildQAFalsification,
	KindCloseout,
	KindCommit,
	KindRefinement,
	KindDiscussion,
	KindHumanVerify,
}

// IsValidKind reports whether kind is a member of the closed Kind enum.
func IsValidKind(kind Kind) bool {
	return slices.Contains(validKinds, Kind(strings.TrimSpace(strings.ToLower(string(kind)))))
}

// KindAppliesTo identifies the node types a kind can be used for. Scope mirrors
// kind per row, so the applies-to vocabulary is the 12-value Kind enum.
type KindAppliesTo string

// KindAppliesTo values mirror the 12 Kind values exactly.
const (
	KindAppliesToPlan                 KindAppliesTo = KindAppliesTo(KindPlan)
	KindAppliesToResearch             KindAppliesTo = KindAppliesTo(KindResearch)
	KindAppliesToBuild                KindAppliesTo = KindAppliesTo(KindBuild)
	KindAppliesToPlanQAProof          KindAppliesTo = KindAppliesTo(KindPlanQAProof)
	KindAppliesToPlanQAFalsification  KindAppliesTo = KindAppliesTo(KindPlanQAFalsification)
	KindAppliesToBuildQAProof         KindAppliesTo = KindAppliesTo(KindBuildQAProof)
	KindAppliesToBuildQAFalsification KindAppliesTo = KindAppliesTo(KindBuildQAFalsification)
	KindAppliesToCloseout             KindAppliesTo = KindAppliesTo(KindCloseout)
	KindAppliesToCommit               KindAppliesTo = KindAppliesTo(KindCommit)
	KindAppliesToRefinement           KindAppliesTo = KindAppliesTo(KindRefinement)
	KindAppliesToDiscussion           KindAppliesTo = KindAppliesTo(KindDiscussion)
	KindAppliesToHumanVerify          KindAppliesTo = KindAppliesTo(KindHumanVerify)
)

// validKindAppliesTo stores all supported applies_to values. Because scope
// mirrors kind, this is the single valid set for both catalog definitions and
// work-item rows; IsValidKindAppliesTo and IsValidWorkItemAppliesTo both
// delegate here.
var validKindAppliesTo = []KindAppliesTo{
	KindAppliesToPlan,
	KindAppliesToResearch,
	KindAppliesToBuild,
	KindAppliesToPlanQAProof,
	KindAppliesToPlanQAFalsification,
	KindAppliesToBuildQAProof,
	KindAppliesToBuildQAFalsification,
	KindAppliesToCloseout,
	KindAppliesToCommit,
	KindAppliesToRefinement,
	KindAppliesToDiscussion,
	KindAppliesToHumanVerify,
}

// KindDefinition stores one reusable kind definition.
//
// Per Drop 3 droplet 3.15 the legacy KindTemplate surface
// (AutoCreateChildren / ProjectMetadataDefaults / ActionItemMetadataDefaults /
// CompletionChecklist / AllowedParentScopes / AllowsParentScope) was removed.
// Parent/child nesting now flows through templates.Template.AllowsNesting +
// the project's baked KindCatalog (per fix L5). KindDefinition keeps only the
// surface still needed by the catalog list/get + JSON-payload schema gate.
type KindDefinition struct {
	ID                  KindID
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []KindAppliesTo
	PayloadSchemaJSON   string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ArchivedAt          *time.Time
}

// KindDefinitionInput holds write-time values for creating/updating a kind definition.
type KindDefinitionInput struct {
	ID                  KindID
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []KindAppliesTo
	PayloadSchemaJSON   string
}

// NewKindDefinition validates and normalizes one kind definition.
func NewKindDefinition(in KindDefinitionInput, now time.Time) (KindDefinition, error) {
	in.ID = NormalizeKindID(in.ID)
	if in.ID == "" {
		return KindDefinition{}, ErrInvalidKindID
	}

	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = string(in.ID)
	}

	appliesTo, err := normalizeKindAppliesToList(in.AppliesTo)
	if err != nil {
		return KindDefinition{}, err
	}
	if len(appliesTo) == 0 {
		return KindDefinition{}, ErrInvalidKindAppliesTo
	}

	schemaJSON := strings.TrimSpace(in.PayloadSchemaJSON)
	if schemaJSON != "" {
		if !json.Valid([]byte(schemaJSON)) {
			return KindDefinition{}, ErrInvalidKindPayloadSchema
		}
	}

	ts := now.UTC()
	return KindDefinition{
		ID:                  in.ID,
		DisplayName:         displayName,
		DescriptionMarkdown: strings.TrimSpace(in.DescriptionMarkdown),
		AppliesTo:           appliesTo,
		PayloadSchemaJSON:   schemaJSON,
		CreatedAt:           ts,
		UpdatedAt:           ts,
	}, nil
}

// AppliesToScope reports whether the kind allows the given target scope.
func (k KindDefinition) AppliesToScope(scope KindAppliesTo) bool {
	scope = NormalizeKindAppliesTo(scope)
	for _, candidate := range k.AppliesTo {
		if NormalizeKindAppliesTo(candidate) == scope {
			return true
		}
	}
	return false
}

// NormalizeKindID canonicalizes kind identifiers for storage/lookup by
// trimming whitespace and lowercasing the input.
func NormalizeKindID(id KindID) KindID {
	trimmed := strings.TrimSpace(string(id))
	if trimmed == "" {
		return ""
	}
	return KindID(strings.ToLower(trimmed))
}

// NormalizeKindAppliesTo canonicalizes applies_to values. Inputs are matched
// case-insensitively against the supported 12-value set and returned in their
// canonical form; unknown values are returned lowercased so callers can still
// detect invalid inputs.
func NormalizeKindAppliesTo(scope KindAppliesTo) KindAppliesTo {
	lowered := strings.TrimSpace(strings.ToLower(string(scope)))
	if lowered == "" {
		return ""
	}
	for _, candidate := range validKindAppliesTo {
		if strings.ToLower(string(candidate)) == lowered {
			return candidate
		}
	}
	return KindAppliesTo(lowered)
}

// IsValidKindAppliesTo reports whether a value is supported for catalog definitions.
func IsValidKindAppliesTo(scope KindAppliesTo) bool {
	scope = NormalizeKindAppliesTo(scope)
	return slices.Contains(validKindAppliesTo, scope)
}

// IsValidWorkItemAppliesTo reports whether a value is supported for work-item rows.
// Scope mirrors kind, so the work-item applies-to set is identical to the
// catalog applies-to set; both helpers delegate to the same 12-value list.
func IsValidWorkItemAppliesTo(scope KindAppliesTo) bool {
	return IsValidKindAppliesTo(scope)
}

// normalizeKindAppliesToList trims, validates, and de-duplicates applies_to values.
func normalizeKindAppliesToList(in []KindAppliesTo) ([]KindAppliesTo, error) {
	out := make([]KindAppliesTo, 0, len(in))
	seen := map[KindAppliesTo]struct{}{}
	for _, raw := range in {
		scope := NormalizeKindAppliesTo(raw)
		if scope == "" {
			continue
		}
		if !IsValidKindAppliesTo(scope) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidKindAppliesTo, scope)
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out, nil
}
