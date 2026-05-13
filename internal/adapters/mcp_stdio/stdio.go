// Package mcpstdio provides the stdio MCP transport entry point.
// RunStdio is the "till mcp" engine — it delegates to the mcprpc server's
// ServeStdio function after normalizing configuration defaults.
package mcpstdio

import (
	"context"
	"fmt"

	"github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"
	"github.com/evanmschultz/tillsyn/internal/adapters/mcp_rpc"
)

// RunStdio starts the MCP server over stdio and blocks until shutdown or startup failure.
func RunStdio(ctx context.Context, cfg mcpcommon.Config, deps mcpcommon.Dependencies) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	normalizedCfg, err := mcpcommon.NormalizeConfig(cfg)
	if err != nil {
		return err
	}
	if deps.CaptureState == nil {
		return fmt.Errorf("capture_state dependency is required")
	}
	return mcprpc.ServeStdio(
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
}
