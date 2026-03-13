package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/Terminus-Lab/themis/internal/mcpadapter"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	logger := log.Logger

	// Load env
	_ = godotenv.Load()

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load Config
	cfg := setup.LoadConfig()

	// Wire dependencies
	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to load dependencies")
		os.Exit(1)
	}

	// Create MCP Server
	server := createMCPServer(deps)

	// Run over stdio
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		// EOF / "server is closing" is expected when stdin closes (e.g. echo | ./bin/eval-mcp)
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "server is closing") {
			logger.Debug().Err(err).Msg("MCP server stopped")
			return
		}
		logger.Error().Err(err).Msg("Failed to run mcp server")
		os.Exit(1)
	}
}

func createMCPServer(deps *setup.Dependencies) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "eval-agent",
			Version: "1.0.0",
		}, nil,
	)

	// Add Tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "evaluate_response",
		Description: "Evaluate an AI agent response for relevance, faithfulness, coherence, completeness, and instruction-following. Optionally provide conversation_id to group multi-turn interactions.",
	}, mcpadapter.NewEvaluateHandler(deps.Executor))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "evaluate_single_judge",
		Description: "Evaluate with a single judge (relevance, faithfulness, coherence, completeness, instruction, or correctness). Faster than full pipeline. Optionally provide conversation_id to group multi-turn interactions.",
	}, mcpadapter.NewEvaluateSingleJudgeHandler(deps.JudgeExecutor))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_conversation",
		Description: "Retrieve all evaluation turns for a given conversation_id. Returns turn count, average confidence, and detailed results for each turn in chronological order.",
	}, mcpadapter.NewGetConversationHandler(deps.Repository))

	return server
}
