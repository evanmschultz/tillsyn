//go:build wails

package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
)

// TestApp_ListProjects_ReturnsDTOForExistingProject is the Go-side smoke test
// for the Wails IPC bridge. It exercises the full read-only chain:
// App.ListProjects -> *app.Service.ListProjects -> *sqlite.Repository against
// an in-memory SQLite DB seeded via the canonical service layer. Asserts that
// every seeded project surfaces back through the App method as a ProjectDTO
// with non-empty ID and Name.
//
// In-memory DB factory is sqlite.OpenInMemory() (canonical multi-connection-
// safe DSN "file::memory:?cache=shared") rather than raw sqlite.Open(":memory:"),
// per PLAN.md round-2 F2-fals resolution: the canonical helper is the
// future-proof choice if a refactor ever lifts the MaxOpenConns(1) cap.
func TestApp_ListProjects_ReturnsDTOForExistingProject(t *testing.T) {
	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("sqlite.OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	// Deterministic, predictable IDs and clock so the test reads cleanly under
	// failure. Counter-based idGen tolerates auto-create paths (child_rules,
	// twin creation) that may issue more than one ID during the seed.
	idCounter := 0
	idGen := func() string {
		idCounter++
		return strings.Repeat("0", 31-len(itoa(idCounter))) + "p" + itoa(idCounter)
	}
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	clk := func() time.Time {
		now = now.Add(time.Second)
		return now
	}
	svc := app.NewService(repo, idGen, clk, app.ServiceConfig{})

	ctx := context.Background()
	seededName := "Tillsyn FE Smoke"
	seededDescription := "in-memory seed for App.ListProjects bridge test"
	seeded, err := svc.CreateProject(ctx, seededName, seededDescription)
	if err != nil {
		t.Fatalf("svc.CreateProject() error = %v", err)
	}
	if seeded.ID == "" {
		t.Fatalf("seeded project has empty ID; CreateProject must assign one")
	}

	// Construct the App against the seeded service. Wails normally calls
	// startup(ctx) at window-open; for the headless smoke test we set the
	// context directly via startup() so ListProjects() sees a real ctx and
	// not the nil zero-value.
	application := NewApp(svc)
	application.startup(ctx)

	result, err := application.ListProjects()
	if err != nil {
		t.Fatalf("App.ListProjects() error = %v", err)
	}
	if len(result) < 1 {
		t.Fatalf("App.ListProjects() returned %d projects; want >= 1", len(result))
	}

	// Assert (c): every DTO has non-empty ID and Name.
	for i, dto := range result {
		if dto.ID == "" {
			t.Errorf("result[%d].ID is empty; want non-empty", i)
		}
		if dto.Name == "" {
			t.Errorf("result[%d].Name is empty; want non-empty", i)
		}
	}

	// Assert (d): the seeded (ID, Name) pair appears in the result set.
	found := false
	for _, dto := range result {
		if dto.ID == seeded.ID && dto.Name == seededName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("seeded project (ID=%q, Name=%q) not present in App.ListProjects() result %+v",
			seeded.ID, seededName, result)
	}
}

// itoa is a tiny local helper to avoid pulling strconv just for ID padding.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
