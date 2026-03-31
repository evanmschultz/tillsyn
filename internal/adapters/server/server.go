// Package server composes HTTP API and MCP transports into one process handler.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/adapters/server/httpapi"
	"github.com/hylla/tillsyn/internal/adapters/server/mcpapi"
)

// defaultBindAddress defines the localhost-first serve default.
const defaultBindAddress = "127.0.0.1:5437"

// defaultShutdownTimeout bounds graceful shutdown time once context cancellation starts.
const defaultShutdownTimeout = 5 * time.Second

// Config defines serve-mode endpoint configuration.
type Config struct {
	HTTPBind               string
	APIEndpoint            string
	MCPEndpoint            string
	ServerName             string
	ServerVersion          string
	ExposeLegacyLeaseTools bool
}

// Dependencies defines app-facing adapters required by server transports.
type Dependencies struct {
	CaptureState common.CaptureStateReader
	Attention    common.AttentionService
}

// NewHandler composes one root HTTP mux containing health, REST API, and MCP endpoints.
func NewHandler(cfg Config, deps Dependencies) (http.Handler, Config, error) {
	normalizedCfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, Config{}, err
	}
	if deps.CaptureState == nil {
		return nil, Config{}, fmt.Errorf("capture_state dependency is required")
	}

	mcpHandler, err := mcpapi.NewHandler(
		mcpapi.Config{
			ServerName:             normalizedCfg.ServerName,
			ServerVersion:          normalizedCfg.ServerVersion,
			EndpointPath:           normalizedCfg.MCPEndpoint,
			ExposeLegacyLeaseTools: normalizedCfg.ExposeLegacyLeaseTools,
		},
		deps.CaptureState,
		deps.Attention,
	)
	if err != nil {
		return nil, Config{}, fmt.Errorf("configure mcp handler: %w", err)
	}
	apiHandler := httpapi.NewHandler(deps.CaptureState, deps.Attention)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", writeHealthStatus)
	mux.HandleFunc("/readyz", writeHealthStatus)
	mux.Handle(normalizedCfg.MCPEndpoint, mcpHandler)
	mux.Handle(normalizedCfg.APIEndpoint, http.StripPrefix(normalizedCfg.APIEndpoint, apiHandler))
	mux.Handle(normalizedCfg.APIEndpoint+"/", http.StripPrefix(normalizedCfg.APIEndpoint, apiHandler))
	return mux, normalizedCfg, nil
}

// Run starts the composed HTTP server and blocks until shutdown or startup failure.
func Run(ctx context.Context, cfg Config, deps Dependencies) error {
	if ctx == nil {
		ctx = context.Background()
	}

	handler, normalizedCfg, err := NewHandler(cfg, deps)
	if err != nil {
		return fmt.Errorf("build server handler: %w", err)
	}
	httpServer := &http.Server{
		Addr:    normalizedCfg.HTTPBind,
		Handler: handler,
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErrCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		shutdownErr := httpServer.Shutdown(shutdownCtx)
		serveErr := <-serveErrCh
		if shutdownErr != nil && !errors.Is(shutdownErr, context.Canceled) {
			return fmt.Errorf("shutdown server: %w", shutdownErr)
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve after shutdown: %w", serveErr)
		}
		return nil
	}
}

// RunStdio starts the MCP server over stdio and blocks until shutdown or startup failure.
func RunStdio(ctx context.Context, cfg Config, deps Dependencies) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	normalizedCfg, err := normalizeConfig(cfg)
	if err != nil {
		return err
	}
	if deps.CaptureState == nil {
		return fmt.Errorf("capture_state dependency is required")
	}
	return mcpapi.ServeStdio(
		mcpapi.Config{
			ServerName:             normalizedCfg.ServerName,
			ServerVersion:          normalizedCfg.ServerVersion,
			EndpointPath:           normalizedCfg.MCPEndpoint,
			ExposeLegacyLeaseTools: normalizedCfg.ExposeLegacyLeaseTools,
		},
		deps.CaptureState,
		deps.Attention,
	)
}

// normalizeConfig applies defaults and validates endpoint collisions.
func normalizeConfig(cfg Config) (Config, error) {
	cfg.HTTPBind = strings.TrimSpace(cfg.HTTPBind)
	if cfg.HTTPBind == "" {
		cfg.HTTPBind = defaultBindAddress
	}

	cfg.APIEndpoint = normalizeEndpoint(cfg.APIEndpoint, "/api/v1")
	cfg.MCPEndpoint = normalizeEndpoint(cfg.MCPEndpoint, "/mcp")
	if cfg.APIEndpoint == cfg.MCPEndpoint {
		return Config{}, fmt.Errorf("api and mcp endpoints must differ")
	}

	cfg.ServerName = strings.TrimSpace(cfg.ServerName)
	if cfg.ServerName == "" {
		cfg.ServerName = "tillsyn"
	}
	cfg.ServerVersion = strings.TrimSpace(cfg.ServerVersion)
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "dev"
	}
	return cfg, nil
}

// normalizeEndpoint normalizes one endpoint path and applies fallback defaults.
func normalizeEndpoint(path string, fallback string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = fallback
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = "/" + strings.Trim(path, "/")
	if path == "/" {
		return fallback
	}
	return path
}

// writeHealthStatus responds with a deterministic readiness payload.
func writeHealthStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
