package mcpcommon

import (
	"fmt"
	"strings"
)

// Config defines serve-mode endpoint configuration for both HTTP and stdio transports.
// HTTPBind and APIEndpoint are HTTP-only fields; MCPEndpoint and the Expose* fields apply
// to all transports. HTTP-only fields are trimmed in Drop W7.D3 when the HTTP transport is
// removed.
type Config struct {
	HTTPBind                      string
	APIEndpoint                   string
	MCPEndpoint                   string
	ServerName                    string
	ServerVersion                 string
	ExposeLegacyLeaseTools        bool
	ExposeLegacyCoordinationTools bool
	ExposeLegacyProjectTools      bool
	ExposeLegacyActionItemTools   bool
}

// Dependencies defines app-facing adapters required by server transports.
type Dependencies struct {
	CaptureState CaptureStateReader
	Attention    AttentionService
}

// defaultBindAddress defines the localhost-first serve default.
const defaultBindAddress = "127.0.0.1:5437"

// NormalizeConfig applies defaults and validates endpoint collisions.
func NormalizeConfig(cfg Config) (Config, error) {
	cfg.HTTPBind = strings.TrimSpace(cfg.HTTPBind)
	if cfg.HTTPBind == "" {
		cfg.HTTPBind = defaultBindAddress
	}

	cfg.APIEndpoint = NormalizeEndpoint(cfg.APIEndpoint, "/api/v1")
	cfg.MCPEndpoint = NormalizeEndpoint(cfg.MCPEndpoint, "/mcp")
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

// NormalizeEndpoint normalizes one endpoint path and applies fallback defaults.
func NormalizeEndpoint(path string, fallback string) string {
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
