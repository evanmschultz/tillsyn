//go:build wails

package main

import (
	"context"
	"embed"
	"fmt"
	"log"

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
	repo, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite repository %q: %w", cfg.Database.Path, err)
	}
	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		DefaultDeleteMode: app.DeleteMode(cfg.Delete.DefaultMode),
	})
	cleanup := func() {
		if closeErr := repo.Close(); closeErr != nil {
			log.Printf("warning: close sqlite repository: %v", closeErr)
		}
	}
	return svc, cleanup, nil
}

func main() {
	svc, cleanup, err := newServiceFromConfig()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
}
