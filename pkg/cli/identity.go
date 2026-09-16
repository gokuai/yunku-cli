package cli

import "github.com/gokuai/yunku-cli/internal/app"

// MCPIdentityHeaders returns HTTP headers aligned with MCP tool calls
// (identity + edition merge). Overlays may pass this to auxiliary clients.
func MCPIdentityHeaders() map[string]string {
	return app.MCPIdentityHeaders()
}
