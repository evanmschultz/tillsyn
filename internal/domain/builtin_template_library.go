package domain

import "time"

// BuiltinTemplateLibraryState identifies whether the installed library matches the builtin contract.
type BuiltinTemplateLibraryState string

// BuiltinTemplateLibraryState values summarize builtin install/refresh state.
const (
	BuiltinTemplateLibraryStateMissing         BuiltinTemplateLibraryState = "missing"
	BuiltinTemplateLibraryStateCurrent         BuiltinTemplateLibraryState = "current"
	BuiltinTemplateLibraryStateUpdateAvailable BuiltinTemplateLibraryState = "update_available"
)

// BuiltinTemplateLibraryStatus stores one builtin library's current install and drift state.
type BuiltinTemplateLibraryStatus struct {
	LibraryID             string                      `json:"library_id"`
	Name                  string                      `json:"name"`
	BuiltinSource         string                      `json:"builtin_source,omitempty"`
	BuiltinVersion        string                      `json:"builtin_version,omitempty"`
	BuiltinRevisionDigest string                      `json:"builtin_revision_digest,omitempty"`
	RequiredKindIDs       []KindID                    `json:"required_kind_ids,omitempty"`
	MissingKindIDs        []KindID                    `json:"missing_kind_ids,omitempty"`
	State                 BuiltinTemplateLibraryState `json:"state"`
	Installed             bool                        `json:"installed"`
	InstalledLibraryName  string                      `json:"installed_library_name,omitempty"`
	InstalledStatus       TemplateLibraryStatus       `json:"installed_status,omitempty"`
	InstalledRevision     int                         `json:"installed_revision,omitempty"`
	InstalledDigest       string                      `json:"installed_digest,omitempty"`
	InstalledUpdatedAt    *time.Time                  `json:"installed_updated_at,omitempty"`
	InstalledBuiltin      bool                        `json:"installed_builtin,omitempty"`
}

// BuiltinTemplateLibraryEnsureResult stores the outcome of one explicit builtin install or refresh.
type BuiltinTemplateLibraryEnsureResult struct {
	Library TemplateLibrary            `json:"library"`
	Status  BuiltinTemplateLibraryStatus `json:"status"`
	Changed bool                       `json:"changed"`
}
