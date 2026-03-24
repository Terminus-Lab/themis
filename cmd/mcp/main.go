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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	logger := log.Logger

	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := setup.LoadConfig()

	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		logger.Error().Err(err).Msg("unable to load dependencies")
		os.Exit(1)
	}

	server := createMCPServer(deps)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "server is closing") {
			logger.Debug().Err(err).Msg("MCP server stopped")
			return
		}
		logger.Error().Err(err).Msg("failed to run mcp server")
		os.Exit(1)
	}
}

func createMCPServer(deps *setup.Dependencies) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "themis",
			Version: "2.0.0",
		}, nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "evaluate_conversation",
		Description: "Evaluate a full multi-turn AI agent conversation. Runs per-turn judges (relevance, coherence, completeness) and a holistic conversation-flow judge, returning a final_score and verdict.",
	}, mcpadapter.NewEvaluateConversationHandler(deps.ConversationEvaluator))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_conversation",
		Description: "Retrieve the stored evaluation for a given conversation_id. Returns turn-by-turn scores, holistic score, final_score, and verdict.",
	}, mcpadapter.NewGetConversationHandler(deps.Repository))

	return server
}
