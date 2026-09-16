package app

// MCPIdentityHeaders returns the same header map used for MCP HTTP requests
// (agent identity, env trace headers, edition MergeHeaders). Intended for
// non-MCP transports such as the A2A gateway client.
func MCPIdentityHeaders() map[string]string {
	return resolveIdentityHeaders()
}
