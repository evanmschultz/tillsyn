//go:build wails

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
