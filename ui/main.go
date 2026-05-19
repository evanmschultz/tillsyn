//go:build wails

package main

import (
	"context"
	"embed"
	"log"

	"github.com/evanmschultz/tillsyn/internal/app"
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

func main() {
	// D2 (app.go) fills in the IPC method bodies and adds service construction.
	// v1: placeholder service; real construction wired in D2.
	application := NewApp(nil)

	err := wails.Run(&options.App{
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
