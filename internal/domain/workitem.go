package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// LifecycleState represents canonical lifecycle state values.
type LifecycleState string

// Canonical lifecycle states.
const (
	StateTodo     LifecycleState = "todo"
	StateProgress LifecycleState = "progress"
	StateDone     LifecycleState = "done"
	StateFailed   LifecycleState = "failed"
	StateArchived LifecycleState = "archived"
)

// ActorType describes the actor class that last updated an item.
type ActorType string

// ActorType values.
const (
	ActorTypeUser   ActorType = "user"
	ActorTypeAgent  ActorType = "agent"
	ActorTypeSystem ActorType = "system"
)

// ContextType classifies planning context snippets attached to an item.
type ContextType string

// ContextType values.
const (
	ContextTypeNote       ContextType = "note"
	ContextTypeConstraint ContextType = "constraint"
	ContextTypeDecision   ContextType = "decision"
	ContextTypeReference  ContextType = "reference"
	ContextTypeWarning    ContextType = "warning"
	ContextTypeRunbook    ContextType = "runbook"
)

// ContextImportance represents relative importance for context blocks.
type ContextImportance string

// ContextImportance values.
const (
	ContextImportanceLow      ContextImportance = "low"
	ContextImportanceNormal   ContextImportance = "normal"
	ContextImportanceHigh     ContextImportance = "high"
	ContextImportanceCritical ContextImportance = "critical"
)

// ResourceType defines resource reference categories.
type ResourceType string

// ResourceType values.
const (
	ResourceTypeLocalFile ResourceType = "local_file"
	ResourceTypeLocalDir  ResourceType = "local_dir"
	ResourceTypeURL       ResourceType = "url"
	ResourceTypeDoc       ResourceType = "doc"
	ResourceTypeTicket    ResourceType = "ticket"
	ResourceTypeSnippet   ResourceType = "snippet"
)

// PathMode identifies whether a resource path is relative or absolute.
type PathMode string

// PathMode values.
const (
	PathModeRelative PathMode = "relative"
	PathModeAbsolute PathMode = "absolute"
)

// ChecklistItem describes a completion-contract checklist item.
type ChecklistItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// CompletionPolicy controls parent/child completion requirements.
type CompletionPolicy struct {
	RequireChildrenDone bool `json:"require_children_done"`
}

// CompletionContract stores start/complete checks and completion evidence.
type CompletionContract struct {
	StartCriteria       []ChecklistItem  `json:"start_criteria"`
	CompletionCriteria  []ChecklistItem  `json:"completion_criteria"`
	CompletionChecklist []ChecklistItem  `json:"completion_checklist"`
	CompletionEvidence  []string         `json:"completion_evidence"`
	CompletionNotes     string           `json:"completion_notes"`
	Policy              CompletionPolicy `json:"policy"`
}

// ContextBlock stores typed contextual notes attached to a work item.
type ContextBlock struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Type       ContextType       `json:"type"`
	Importance ContextImportance `json:"importance"`
}

// ResourceRef stores a path/URL reference that supports future context hydration.
type ResourceRef struct {
	ID             string       `json:"id"`
	ResourceType   ResourceType `json:"resource_type"`
	Location       string       `json:"location"`
	PathMode       PathMode     `json:"path_mode"`
	BaseAlias      string       `json:"base_alias"`
	Title          string       `json:"title"`
	Notes          string       `json:"notes"`
	Tags           []string     `json:"tags"`
	LastVerifiedAt *time.Time   `json:"last_verified_at,omitempty"`
}

// ActionItemMetadata stores rich planning context for an item.
type ActionItemMetadata struct {
	Objective                string             `json:"objective"`
	ImplementationNotesUser  string             `json:"implementation_notes_user"`
	ImplementationNotesAgent string             `json:"implementation_notes_agent"`
	AcceptanceCriteria       string             `json:"acceptance_criteria"`
	DefinitionOfDone         string             `json:"definition_of_done"`
	ValidationPlan           string             `json:"validation_plan"`
	BlockedReason            string             `json:"blocked_reason"`
	RiskNotes                string             `json:"risk_notes"`
	CommandSnippets          []string           `json:"command_snippets"`
	ExpectedOutputs          []string           `json:"expected_outputs"`
	DecisionLog              []string           `json:"decision_log"`
	RelatedItems             []string           `json:"related_items"`
	TransitionNotes          string             `json:"transition_notes"`
	DependsOn                []string           `json:"depends_on"`
	BlockedBy                []string           `json:"blocked_by"`
	ContextBlocks            []ContextBlock     `json:"context_blocks"`
	ResourceRefs             []ResourceRef      `json:"resource_refs"`
	KindPayload              json.RawMessage    `json:"kind_payload,omitempty"`
	CompletionContract       CompletionContract `json:"completion_contract"`
	Outcome                  string             `json:"outcome,omitempty"`
}

// normalizeLifecycleState canonicalizes lifecycle state aliases.
func normalizeLifecycleState(state LifecycleState) LifecycleState {
	switch strings.TrimSpace(strings.ToLower(string(state))) {
	case "to-do", "todo":
		return StateTodo
	case "in-progress", "progress", "doing":
		return StateProgress
	case "done", "complete", "completed":
		return StateDone
	case "failed", "fail":
		return StateFailed
	case "archived", "archive":
		return StateArchived
	default:
		return LifecycleState(strings.TrimSpace(strings.ToLower(string(state))))
	}
}

// isValidLifecycleState reports whether the lifecycle state is canonical.
func isValidLifecycleState(state LifecycleState) bool {
	state = normalizeLifecycleState(state)
	return slices.Contains([]LifecycleState{StateTodo, StateProgress, StateDone, StateFailed, StateArchived}, state)
}

// IsTerminalState reports whether a lifecycle state is terminal (done or failed).
func IsTerminalState(state LifecycleState) bool {
	state = normalizeLifecycleState(state)
	return state == StateDone || state == StateFailed
}

// isValidActorType reports whether actor type is supported.
func isValidActorType(actorType ActorType) bool {
	actorType = ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	return slices.Contains([]ActorType{ActorTypeUser, ActorTypeAgent, ActorTypeSystem}, actorType)
}

// normalizeActionItemMetadata trims and validates rich metadata.
func normalizeActionItemMetadata(meta ActionItemMetadata) (ActionItemMetadata, error) {
	meta.Objective = strings.TrimSpace(meta.Objective)
	meta.ImplementationNotesUser = strings.TrimSpace(meta.ImplementationNotesUser)
	meta.ImplementationNotesAgent = strings.TrimSpace(meta.ImplementationNotesAgent)
	meta.AcceptanceCriteria = strings.TrimSpace(meta.AcceptanceCriteria)
	meta.DefinitionOfDone = strings.TrimSpace(meta.DefinitionOfDone)
	meta.ValidationPlan = strings.TrimSpace(meta.ValidationPlan)
	meta.BlockedReason = strings.TrimSpace(meta.BlockedReason)
	meta.RiskNotes = strings.TrimSpace(meta.RiskNotes)
	meta.TransitionNotes = strings.TrimSpace(meta.TransitionNotes)
	meta.Outcome = strings.TrimSpace(meta.Outcome)
	meta.CommandSnippets = normalizeStringList(meta.CommandSnippets)
	meta.ExpectedOutputs = normalizeStringList(meta.ExpectedOutputs)
	meta.DecisionLog = normalizeStringList(meta.DecisionLog)
	meta.RelatedItems = normalizeStringList(meta.RelatedItems)
	meta.DependsOn = normalizeStringList(meta.DependsOn)
	meta.BlockedBy = normalizeStringList(meta.BlockedBy)
	meta.KindPayload = bytes.TrimSpace(meta.KindPayload)
	if len(meta.KindPayload) > 0 && !json.Valid(meta.KindPayload) {
		return ActionItemMetadata{}, ErrInvalidKindPayload
	}
	var err error
	meta.CompletionContract, err = normalizeCompletionContract(meta.CompletionContract)
	if err != nil {
		return ActionItemMetadata{}, err
	}

	contextBlocks := make([]ContextBlock, 0, len(meta.ContextBlocks))
	for i, block := range meta.ContextBlocks {
		block.Title = strings.TrimSpace(block.Title)
		block.Body = strings.TrimSpace(block.Body)
		if block.Body == "" {
			continue
		}
		block.Type = ContextType(strings.TrimSpace(strings.ToLower(string(block.Type))))
		if block.Type == "" {
			block.Type = ContextTypeNote
		}
		if !slices.Contains([]ContextType{
			ContextTypeNote,
			ContextTypeConstraint,
			ContextTypeDecision,
			ContextTypeReference,
			ContextTypeWarning,
			ContextTypeRunbook,
		}, block.Type) {
			return ActionItemMetadata{}, fmt.Errorf("invalid context block type at index %d", i)
		}
		block.Importance = ContextImportance(strings.TrimSpace(strings.ToLower(string(block.Importance))))
		if block.Importance == "" {
			block.Importance = ContextImportanceNormal
		}
		if !slices.Contains([]ContextImportance{
			ContextImportanceLow,
			ContextImportanceNormal,
			ContextImportanceHigh,
			ContextImportanceCritical,
		}, block.Importance) {
			return ActionItemMetadata{}, fmt.Errorf("invalid context block importance at index %d", i)
		}
		contextBlocks = append(contextBlocks, block)
	}
	meta.ContextBlocks = contextBlocks

	resourceRefs := make([]ResourceRef, 0, len(meta.ResourceRefs))
	for i, ref := range meta.ResourceRefs {
		ref.ID = strings.TrimSpace(ref.ID)
		ref.Location = strings.TrimSpace(ref.Location)
		ref.BaseAlias = strings.TrimSpace(ref.BaseAlias)
		ref.Title = strings.TrimSpace(ref.Title)
		ref.Notes = strings.TrimSpace(ref.Notes)
		ref.Tags = normalizeLabels(ref.Tags)
		if ref.Location == "" {
			continue
		}
		ref.ResourceType = ResourceType(strings.TrimSpace(strings.ToLower(string(ref.ResourceType))))
		if ref.ResourceType == "" {
			ref.ResourceType = ResourceTypeDoc
		}
		if !slices.Contains([]ResourceType{
			ResourceTypeLocalFile,
			ResourceTypeLocalDir,
			ResourceTypeURL,
			ResourceTypeDoc,
			ResourceTypeTicket,
			ResourceTypeSnippet,
		}, ref.ResourceType) {
			return ActionItemMetadata{}, fmt.Errorf("invalid resource type at index %d", i)
		}
		ref.PathMode = PathMode(strings.TrimSpace(strings.ToLower(string(ref.PathMode))))
		if ref.PathMode == "" {
			ref.PathMode = PathModeRelative
		}
		if !slices.Contains([]PathMode{PathModeRelative, PathModeAbsolute}, ref.PathMode) {
			return ActionItemMetadata{}, fmt.Errorf("invalid path mode at index %d", i)
		}
		if ref.LastVerifiedAt != nil {
			ts := ref.LastVerifiedAt.UTC().Truncate(time.Second)
			ref.LastVerifiedAt = &ts
		}
		resourceRefs = append(resourceRefs, ref)
	}
	meta.ResourceRefs = resourceRefs

	return meta, nil
}

// MergeActionItemMetadata applies optional defaults to actionItem metadata without weakening explicit values.
func MergeActionItemMetadata(base ActionItemMetadata, defaults *ActionItemMetadata) (ActionItemMetadata, error) {
	normalizedBase, err := normalizeActionItemMetadata(base)
	if err != nil {
		return ActionItemMetadata{}, err
	}
	if defaults == nil {
		return normalizedBase, nil
	}

	normalizedDefaults, err := normalizeActionItemMetadata(*defaults)
	if err != nil {
		return ActionItemMetadata{}, err
	}

	merged := normalizedBase
	if merged.Objective == "" {
		merged.Objective = normalizedDefaults.Objective
	}
	if merged.ImplementationNotesUser == "" {
		merged.ImplementationNotesUser = normalizedDefaults.ImplementationNotesUser
	}
	if merged.ImplementationNotesAgent == "" {
		merged.ImplementationNotesAgent = normalizedDefaults.ImplementationNotesAgent
	}
	if merged.AcceptanceCriteria == "" {
		merged.AcceptanceCriteria = normalizedDefaults.AcceptanceCriteria
	}
	if merged.DefinitionOfDone == "" {
		merged.DefinitionOfDone = normalizedDefaults.DefinitionOfDone
	}
	if merged.ValidationPlan == "" {
		merged.ValidationPlan = normalizedDefaults.ValidationPlan
	}
	if merged.BlockedReason == "" {
		merged.BlockedReason = normalizedDefaults.BlockedReason
	}
	if merged.RiskNotes == "" {
		merged.RiskNotes = normalizedDefaults.RiskNotes
	}
	merged.CommandSnippets = mergeStringLists(merged.CommandSnippets, normalizedDefaults.CommandSnippets)
	merged.ExpectedOutputs = mergeStringLists(merged.ExpectedOutputs, normalizedDefaults.ExpectedOutputs)
	merged.DecisionLog = mergeStringLists(merged.DecisionLog, normalizedDefaults.DecisionLog)
	merged.RelatedItems = mergeStringLists(merged.RelatedItems, normalizedDefaults.RelatedItems)
	if merged.TransitionNotes == "" {
		merged.TransitionNotes = normalizedDefaults.TransitionNotes
	}
	if merged.Outcome == "" {
		merged.Outcome = normalizedDefaults.Outcome
	}
	merged.DependsOn = mergeStringLists(merged.DependsOn, normalizedDefaults.DependsOn)
	merged.BlockedBy = mergeStringLists(merged.BlockedBy, normalizedDefaults.BlockedBy)
	merged.ContextBlocks = mergeContextBlocks(merged.ContextBlocks, normalizedDefaults.ContextBlocks)
	merged.ResourceRefs = mergeResourceRefs(merged.ResourceRefs, normalizedDefaults.ResourceRefs)
	mergedPayload, err := mergeKindPayloadDefaults(merged.KindPayload, normalizedDefaults.KindPayload)
	if err != nil {
		return ActionItemMetadata{}, err
	}
	merged.KindPayload = mergedPayload
	merged.CompletionContract, err = MergeCompletionContract(merged.CompletionContract, &normalizedDefaults.CompletionContract)
	if err != nil {
		return ActionItemMetadata{}, err
	}

	return normalizeActionItemMetadata(merged)
}

// MergeCompletionContract applies optional defaults to a completion contract without weakening explicit values.
func MergeCompletionContract(base CompletionContract, defaults *CompletionContract) (CompletionContract, error) {
	normalizedBase, err := normalizeCompletionContract(base)
	if err != nil {
		return CompletionContract{}, err
	}
	if defaults == nil {
		return normalizedBase, nil
	}

	normalizedDefaults, err := normalizeCompletionContract(*defaults)
	if err != nil {
		return CompletionContract{}, err
	}

	merged := CompletionContract{
		StartCriteria:       mergeChecklistItems(normalizedBase.StartCriteria, normalizedDefaults.StartCriteria),
		CompletionCriteria:  mergeChecklistItems(normalizedBase.CompletionCriteria, normalizedDefaults.CompletionCriteria),
		CompletionChecklist: mergeChecklistItems(normalizedBase.CompletionChecklist, normalizedDefaults.CompletionChecklist),
		CompletionEvidence:  mergeStringLists(normalizedBase.CompletionEvidence, normalizedDefaults.CompletionEvidence),
		CompletionNotes:     normalizedBase.CompletionNotes,
		Policy: CompletionPolicy{
			RequireChildrenDone: normalizedBase.Policy.RequireChildrenDone || normalizedDefaults.Policy.RequireChildrenDone,
		},
	}
	if merged.CompletionNotes == "" {
		merged.CompletionNotes = normalizedDefaults.CompletionNotes
	}

	return normalizeCompletionContract(merged)
}

// normalizeChecklist trims checklist ids/text and removes empty rows.
func normalizeChecklist(in []ChecklistItem) ([]ChecklistItem, error) {
	out := make([]ChecklistItem, 0, len(in))
	seen := map[string]struct{}{}
	for i, item := range in {
		item.ID = strings.TrimSpace(item.ID)
		item.Text = strings.TrimSpace(item.Text)
		if item.Text == "" {
			continue
		}
		if item.ID == "" {
			item.ID = fmt.Sprintf("item-%d", i+1)
		}
		if _, exists := seen[item.ID]; exists {
			return nil, fmt.Errorf("duplicate checklist id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

// normalizeCompletionContract trims and validates completion-contract fields.
func normalizeCompletionContract(contract CompletionContract) (CompletionContract, error) {
	contract.CompletionEvidence = normalizeStringList(contract.CompletionEvidence)
	contract.CompletionNotes = strings.TrimSpace(contract.CompletionNotes)

	var err error
	contract.StartCriteria, err = normalizeChecklist(contract.StartCriteria)
	if err != nil {
		return CompletionContract{}, err
	}
	contract.CompletionCriteria, err = normalizeChecklist(contract.CompletionCriteria)
	if err != nil {
		return CompletionContract{}, err
	}
	contract.CompletionChecklist, err = normalizeChecklist(contract.CompletionChecklist)
	if err != nil {
		return CompletionContract{}, err
	}
	return contract, nil
}

// mergeChecklistItems merges checklist rows by ID while preserving the base order.
func mergeChecklistItems(base, defaults []ChecklistItem) []ChecklistItem {
	out := make([]ChecklistItem, 0, len(base)+len(defaults))
	seen := map[string]struct{}{}
	appendItem := func(item ChecklistItem) {
		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			return
		}
		if _, ok := seen[item.ID]; ok {
			return
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	for _, item := range base {
		appendItem(item)
	}
	for _, item := range defaults {
		appendItem(item)
	}
	return out
}

// mergeStringLists merges normalized string slices while preserving the base order.
func mergeStringLists(base, defaults []string) []string {
	return normalizeStringList(append(append([]string(nil), base...), defaults...))
}

// mergeKindPayloadDefaults deep-merges object-shaped defaults into a caller payload.
func mergeKindPayloadDefaults(base, defaults json.RawMessage) (json.RawMessage, error) {
	base = bytes.TrimSpace(base)
	defaults = bytes.TrimSpace(defaults)
	if len(defaults) == 0 {
		return append(json.RawMessage(nil), base...), nil
	}
	if len(base) == 0 {
		return append(json.RawMessage(nil), defaults...), nil
	}
	var baseValue any
	if err := json.Unmarshal(base, &baseValue); err != nil {
		return nil, ErrInvalidKindPayload
	}
	var defaultValue any
	if err := json.Unmarshal(defaults, &defaultValue); err != nil {
		return nil, ErrInvalidKindPayload
	}
	merged, ok := mergeKindPayloadValue(baseValue, defaultValue)
	if !ok {
		return append(json.RawMessage(nil), base...), nil
	}
	encoded, err := json.Marshal(merged)
	if err != nil {
		return nil, ErrInvalidKindPayload
	}
	return bytes.TrimSpace(encoded), nil
}

// mergeKindPayloadValue recursively fills missing object fields from defaults.
func mergeKindPayloadValue(base, defaults any) (any, bool) {
	baseObject, baseOK := base.(map[string]any)
	defaultObject, defaultOK := defaults.(map[string]any)
	if !baseOK || !defaultOK {
		return base, false
	}
	merged := make(map[string]any, len(defaultObject)+len(baseObject))
	for key, value := range defaultObject {
		merged[key] = value
	}
	for key, value := range baseObject {
		if currentDefault, ok := merged[key]; ok {
			if nested, nestedOK := mergeKindPayloadValue(value, currentDefault); nestedOK {
				merged[key] = nested
				continue
			}
		}
		merged[key] = value
	}
	return merged, true
}

// mergeContextBlocks merges context blocks by normalized identity while preserving the base order.
func mergeContextBlocks(base, defaults []ContextBlock) []ContextBlock {
	out := make([]ContextBlock, 0, len(base)+len(defaults))
	seen := map[string]struct{}{}
	appendBlock := func(block ContextBlock) {
		block.Title = strings.TrimSpace(block.Title)
		block.Body = strings.TrimSpace(block.Body)
		block.Type = ContextType(strings.TrimSpace(strings.ToLower(string(block.Type))))
		if block.Type == "" {
			block.Type = ContextTypeNote
		}
		block.Importance = ContextImportance(strings.TrimSpace(strings.ToLower(string(block.Importance))))
		if block.Importance == "" {
			block.Importance = ContextImportanceNormal
		}
		if block.Body == "" {
			return
		}
		key := strings.Join([]string{
			string(block.Type),
			string(block.Importance),
			block.Title,
			block.Body,
		}, "\x1f")
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, block)
	}
	for _, block := range base {
		appendBlock(block)
	}
	for _, block := range defaults {
		appendBlock(block)
	}
	return out
}

// mergeResourceRefs merges resource refs by normalized identity while preserving the base order.
func mergeResourceRefs(base, defaults []ResourceRef) []ResourceRef {
	out := make([]ResourceRef, 0, len(base)+len(defaults))
	seen := map[string]struct{}{}
	appendRef := func(ref ResourceRef) {
		ref.ID = strings.TrimSpace(ref.ID)
		ref.ResourceType = ResourceType(strings.TrimSpace(strings.ToLower(string(ref.ResourceType))))
		if ref.ResourceType == "" {
			ref.ResourceType = ResourceTypeDoc
		}
		ref.Location = strings.TrimSpace(ref.Location)
		ref.PathMode = PathMode(strings.TrimSpace(strings.ToLower(string(ref.PathMode))))
		if ref.PathMode == "" {
			ref.PathMode = PathModeRelative
		}
		ref.BaseAlias = strings.TrimSpace(ref.BaseAlias)
		ref.Title = strings.TrimSpace(ref.Title)
		ref.Notes = strings.TrimSpace(ref.Notes)
		ref.Tags = normalizeLabels(ref.Tags)
		if ref.Location == "" {
			return
		}
		key := ref.ID
		if key == "" {
			key = strings.Join([]string{
				string(ref.ResourceType),
				ref.Location,
				string(ref.PathMode),
				ref.BaseAlias,
				ref.Title,
				ref.Notes,
				strings.Join(ref.Tags, "\x1f"),
			}, "\x1f")
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	for _, ref := range base {
		appendRef(ref)
	}
	for _, ref := range defaults {
		appendRef(ref)
	}
	return out
}

// normalizeStringList trims and deduplicates string slices.
func normalizeStringList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
