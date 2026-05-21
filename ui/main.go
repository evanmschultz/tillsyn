package main

import (
	"context"
	"embed"
	"fmt"
	"strings"

	charmLog "github.com/charmbracelet/log"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/platform"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// App is the Wails application struct bound to the frontend via IPC.
// Public methods on App are accessible from JavaScript as window.go.main.App.*
// All business logic delegates to the underlying *app.Service.
type App struct {
	ctx context.Context
	svc *app.Service
}

// NewApp creates a new App with the given service.
func NewApp(svc *app.Service) *App {
	return &App{svc: svc}
}

// startup is called by the Wails runtime when the application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ListProjects is the Wails IPC method exposed to the frontend as
// window.go.main.App.ListProjects(). Returns every non-archived project on
// the underlying SQLite store projected into the JS-friendly ProjectDTO
// shape. Read-only — never mutates the store. Errors from the service layer
// surface verbatim (Wails serializes (T, error) returns as a JS promise that
// rejects on non-nil error).
func (a *App) ListProjects() ([]ProjectDTO, error) {
	projects, err := a.svc.ListProjects(a.ctx, false)
	if err != nil {
		return nil, err
	}
	dtos := make([]ProjectDTO, 0, len(projects))
	for _, p := range projects {
		dtos = append(dtos, ProjectDTO{ID: p.ID, Name: p.Name})
	}
	return dtos, nil
}

// ListActionItems is the Wails IPC method exposed to the frontend as
// window.go.main.App.ListActionItems(projectID). Returns every non-archived
// action item belonging to projectID, projected into the JS-friendly
// ActionItemDTO shape. Read-only — never mutates the store.
//
// projectID is whitespace-trimmed before delegation. If the trimmed value is
// empty, the method returns (empty slice, nil) — NOT an error — so the
// frontend can render the "Select a project" empty state cleanly without
// catching a rejected promise. Callers that need to distinguish "no project
// selected" from "project selected but empty" must look at the projectID they
// passed, not at the return value.
//
// On a non-empty projectID, the method delegates to Service.ListActionItems
// with includeArchived=false (archived items live outside the primary
// browse view; a future projection can opt-in via a follow-up IPC method).
// Service-layer errors surface verbatim — Wails serializes (T, error)
// returns as a JS promise that rejects on non-nil error.
//
// The result is ALWAYS a non-nil slice (empty when the service returns zero
// rows), matching ListProjects' contract so the FE never has to distinguish
// null from []. Each domain.ActionItem maps field-for-field via plain string
// conversion on the enum-typed columns (Kind, Role, StructuralType,
// LifecycleState, Priority) — the underlying domain types are all `type X
// string` so the conversion is allocation-free and round-trips the raw enum
// token (e.g. "build", "builder", "droplet"). Drop FE 2.6 D1.
func (a *App) ListActionItems(projectID string) ([]ActionItemDTO, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return []ActionItemDTO{}, nil
	}
	items, err := a.svc.ListActionItems(a.ctx, projectID, false)
	if err != nil {
		return nil, err
	}
	dtos := make([]ActionItemDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, ActionItemDTO{
			ID:             item.ID,
			ProjectID:      item.ProjectID,
			ParentID:       item.ParentID,
			Title:          item.Title,
			Kind:           string(item.Kind),
			Role:           string(item.Role),
			StructuralType: string(item.StructuralType),
			LifecycleState: string(item.LifecycleState),
			Priority:       string(item.Priority),
		})
	}
	return dtos, nil
}

// newServiceFromConfig constructs a live *app.Service against the same SQLite
// database the CLI opens, resolved through the canonical platform/config chain
// (mirrors cmd/till/main.go:2244-2314). Returns the service plus a cleanup
// callback that closes the underlying repository. Callers MUST defer the
// cleanup func before passing the service to the Wails runtime.
func newServiceFromConfig() (*app.Service, func(), error) {
	paths, err := platform.DefaultPaths()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve runtime paths: %w", err)
	}
	defaultCfg := config.Default(paths.DBPath)
	cfg, err := config.Load(paths.ConfigPath, defaultCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("load config %q: %w", paths.ConfigPath, err)
	}
	// Startup observability: emit the exact SQLite path the Wails host will
	// open so future "the FE shows no data" debugging is one log line away.
	// The Wails host and the `till` CLI / `till mcp` stdio runtime MUST resolve
	// to the same DB file (both go through platform.DefaultPaths() +
	// config.Load(); divergence here is a path-drift bug). Logged at Info so
	// it surfaces in the `wails dev` console without needing debug flags.
	charmLog.Info("ui host startup",
		"config_path", paths.ConfigPath,
		"db_path", cfg.Database.Path,
		"db_default", paths.DBPath,
	)
	repo, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite repository %q: %w", cfg.Database.Path, err)
	}
	charmLog.Info("ui host sqlite open", "db_path", cfg.Database.Path)
	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		DefaultDeleteMode: app.DeleteMode(cfg.Delete.DefaultMode),
	})
	cleanup := func() {
		if closeErr := repo.Close(); closeErr != nil {
			charmLog.Warn("ui host sqlite close failed", "err", closeErr)
		}
	}
	return svc, cleanup, nil
}

func main() {
	svc, cleanup, err := newServiceFromConfig()
	if err != nil {
		charmLog.Fatal(err)
	}
	defer cleanup()

	application := NewApp(svc)

	err = wails.Run(&options.App{
		Title:  "Tillsyn",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 255},
		OnStartup:        application.startup,
		Bind: []interface{}{
			application,
		},
	})
	if err != nil {
		charmLog.Fatal(err)
	}
}
