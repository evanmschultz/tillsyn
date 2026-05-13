// Package server composes HTTP API and MCP transports into one process handler.
// W7.D2: non-HTTP symbols extracted to mcp_common/, mcp_rpc/, mcp_stdio/.
// This file retains only HTTP-residue (Run, NewHandler, writeHealthStatus plus
// the HTTP-only constants). W7.D3 deletes this package entirely.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"
	mcprpc "github.com/evanmschultz/tillsyn/internal/adapters/mcp_rpc"
	"github.com/evanmschultz/tillsyn/internal/adapters/server/httpapi"
)

// defaultShutdownTimeout bounds graceful shutdown time once context cancellation starts.
const defaultShutdownTimeout = 5 * time.Second

// NewHandler composes one root HTTP mux containing health, REST API, and MCP endpoints.
func NewHandler(cfg mcpcommon.Config, deps mcpcommon.Dependencies) (http.Handler, mcpcommon.Config, error) {
	normalizedCfg, err := mcpcommon.NormalizeConfig(cfg)
	if err != nil {
		return nil, mcpcommon.Config{}, err
	}
	if deps.CaptureState == nil {
		return nil, mcpcommon.Config{}, fmt.Errorf("capture_state dependency is required")
	}

	mcpHandler, err := mcprpc.NewHandler(
		mcprpc.Config{
			ServerName:                    normalizedCfg.ServerName,
			ServerVersion:                 normalizedCfg.ServerVersion,
			EndpointPath:                  normalizedCfg.MCPEndpoint,
			ExposeLegacyLeaseTools:        normalizedCfg.ExposeLegacyLeaseTools,
			ExposeLegacyCoordinationTools: normalizedCfg.ExposeLegacyCoordinationTools,
			ExposeLegacyProjectTools:      normalizedCfg.ExposeLegacyProjectTools,
			ExposeLegacyActionItemTools:   normalizedCfg.ExposeLegacyActionItemTools,
		},
		deps.CaptureState,
		deps.Attention,
	)
	if err != nil {
		return nil, mcpcommon.Config{}, fmt.Errorf("configure mcp handler: %w", err)
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
func Run(ctx context.Context, cfg mcpcommon.Config, deps mcpcommon.Dependencies) error {
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

// writeHealthStatus responds with a deterministic readiness payload.
func writeHealthStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
