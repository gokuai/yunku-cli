package app

import (
	"context"
	"encoding/json"

	"github.com/gokuai/yunku-cli/internal/executor"
	"github.com/gokuai/yunku-cli/pkg/edition"
)

// toolCallerAdapter bridges executor.Runner to the public edition.ToolCaller
// interface so that private overlays can invoke MCP tools without importing
// internal packages.
type toolCallerAdapter struct {
	runner executor.Runner
	flags  *GlobalFlags
}

func newToolCallerAdapter(runner executor.Runner, flags *GlobalFlags) edition.ToolCaller {
	return &toolCallerAdapter{runner: runner, flags: flags}
}

func (a *toolCallerAdapter) CallTool(ctx context.Context, productID, toolName string, args map[string]any) (*edition.ToolResult, error) {
	inv := executor.NewHelperInvocation("overlay."+productID+"."+toolName, productID, toolName, args)
	result, err := a.runner.Run(ctx, inv)
	if err != nil {
		return nil, err
	}
	return convertResult(result), nil
}

func (a *toolCallerAdapter) Format() string {
	if a.flags != nil {
		return a.flags.Format
	}
	return "json"
}

func (a *toolCallerAdapter) DryRun() bool {
	return a.flags != nil && a.flags.DryRun
}

func convertResult(r executor.Result) *edition.ToolResult {
	resp := r.Response
	if resp == nil {
		return &edition.ToolResult{}
	}

	// The runtime runner stores MCP response content under "content".
	contentRaw, ok := resp["content"]
	if !ok {
		// Dry-run or echo mode: serialize the whole response as text.
		data, _ := json.Marshal(resp)
		return &edition.ToolResult{
			Content: []edition.ContentBlock{{Type: "text", Text: string(data)}},
		}
	}

	// Content may be a []any of {type, text} blocks from the MCP response,
	// or a single map for mock mode.
	switch v := contentRaw.(type) {
	case []any:
		blocks := make([]edition.ContentBlock, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				blocks = append(blocks, edition.ContentBlock{
					Type: strVal(m, "type"),
					Text: strVal(m, "text"),
				})
			}
		}
		return &edition.ToolResult{Content: blocks}
	case map[string]any:
		data, _ := json.Marshal(v)
		return &edition.ToolResult{
			Content: []edition.ContentBlock{{Type: "text", Text: string(data)}},
		}
	default:
		data, _ := json.Marshal(contentRaw)
		return &edition.ToolResult{
			Content: []edition.ContentBlock{{Type: "text", Text: string(data)}},
		}
	}
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
