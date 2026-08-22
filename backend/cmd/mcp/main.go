// Command mcp is an optional stdio-only MCP adapter for hosts that cannot use HTTP.
// Prefer the integrated server: `go run ./cmd/server` exposes REST + MCP at /mcp
// on the same process and port. Do not run this alongside the API unless you need
// pure stdio (e.g. some desktop MCP clients).
//
//	SWYNGORA_API_URL=http://localhost:8080 go run ./cmd/mcp
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/config"
	mcpx "gitlab.com/trace-analysis/swyngora/backend/internal/transport/mcp"
)

func main() {
	config.LoadDotEnv(".env")
	config.LoadDotEnv("backend/.env")

	apiURL := os.Getenv("SWYNGORA_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	fmt.Fprintln(os.Stderr, "note: MCP is integrated in cmd/server at /mcp — this stdio binary is optional")

	token := os.Getenv("API_AUTH_TOKEN")
	if token == "" {
		token = os.Getenv("SWYNGORA_API_TOKEN")
	}

	s := mcpx.NewServer(mcpx.ServerOptions{
		APIBaseURL: apiURL,
		APIToken:   token,
		Name:       "swyngora-mcp",
		Version:    "0.1.0",
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		os.Exit(1)
	}
}
