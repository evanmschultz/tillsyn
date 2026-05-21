package main

// ProjectDTO is the wire-format projection of domain.Project exposed across
// the Wails IPC boundary as window.go.main.App.ListProjects' element type.
// Fields are capitalized (Go export convention) — Wails serializes Go structs
// to JS as {"ID": "...", "Name": "..."} by default, and the matching JS-side
// TypeScript ambient declaration in ui/frontend/src/types/wails.d.ts (added
// in D1.5) types this as Promise<{ ID: string; Name: string }[]>. The DTO
// lives in this dedicated file rather than ui/main.go to pre-empt entrypoint
// bloat as future drops add more IPC surfaces; both files are package main
// so there is no import-boundary cost (PLAN.md §N2 — F4-fals resolution).
type ProjectDTO struct {
	ID   string
	Name string
}

// ActionItemDTO is the wire-format projection of domain.ActionItem exposed
// across the Wails IPC boundary as window.go.main.App.ListActionItems' element
// type. Every field is `string` to match the FE-friendly serialization
// established by ProjectDTO — the domain's closed-enum types (Kind, Role,
// StructuralType, LifecycleState, Priority) all carry an underlying string
// representation, so each round-trips losslessly as its raw enum-token value
// (e.g. "build", "builder", "droplet", "in_progress", "high"). Fields are
// capitalized (Go export convention); Wails serializes Go structs to JS as
// {"ID": "...", "Title": "...", ...} by default, and the matching JS-side
// types are regenerated into ui/frontend/wailsjs/go/main/App.d.ts on the next
// `wails build`. The DTO intentionally projects only the columns the
// initial action-items list view needs (ID + ProjectID + ParentID for
// navigation, Title for display, Kind/Role/StructuralType/LifecycleState/
// Priority for filter + badge rendering) rather than the full ActionItem
// surface — additional projections can be added in future drops without a
// breaking change because callers consume by field name. Drop FE 2.6 D1.
type ActionItemDTO struct {
	ID             string
	ProjectID      string
	ParentID       string
	Title          string
	Kind           string
	Role           string
	StructuralType string
	LifecycleState string
	Priority       string
}
