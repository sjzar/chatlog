package errors

import "github.com/mark3labs/mcp-go/mcp"

func ErrMCPTool(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: err.Error(),
			},
		},
		IsError: true,
	}
}
