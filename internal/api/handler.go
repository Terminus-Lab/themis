package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Terminus-Lab/themis/internal/api/middleware"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/emicklei/go-restful/v3"
	"github.com/rs/zerolog"
)

type Handler struct {
	evaluator  *executor.ConversationEvaluator
	repository storage.Repository
	logger     *zerolog.Logger
}

func NewHandler(
	evaluator *executor.ConversationEvaluator,
	repository storage.Repository,
	logger *zerolog.Logger,
) *Handler {
	return &Handler{
		evaluator:  evaluator,
		repository: repository,
		logger:     logger,
	}
}

// POST /api/v1/conversations/evaluate
func (h *Handler) EvaluateConversation(req *restful.Request, resp *restful.Response) {
	var evalReq ConversationEvalRequest
	if err := req.ReadEntity(&evalReq); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse conversation evaluation request")
		middleware.HandleError(resp, err, http.StatusBadRequest)
		return
	}

	if evalReq.ConversationID == "" {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}
	if len(evalReq.Turns) == 0 {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "turns must not be empty"})
		return
	}

	turns := make([]models.ConversationTurn, len(evalReq.Turns))
	for i, t := range evalReq.Turns {
		turns[i] = models.ConversationTurn{
			TurnIndex:      t.TurnIndex,
			UserQuery:      t.UserQuery,
			Answer:         t.Answer,
			Context:        t.Context,
			ExpectedOutput: t.ExpectedOutput,
		}
	}

	convReq := models.ConversationEvaluationRequest{
		ConversationID: evalReq.ConversationID,
		Agent: models.Agent{
			Name:    evalReq.Agent.Name,
			Version: evalReq.Agent.Version,
		},
		Turns: turns,
	}

	h.logger.Info().
		Str("conversation_id", convReq.ConversationID).
		Int("turn_count", len(convReq.Turns)).
		Msg("start conversation evaluation")

	ctx := req.Request.Context()
	result := h.evaluator.Execute(ctx, convReq)

	_ = resp.WriteHeaderAndEntity(http.StatusOK, toConversationEvalResponse(result))
}

// GET /api/v1/conversations/{conversation_id}
func (h *Handler) GetConversation(req *restful.Request, resp *restful.Response) {
	conversationID := req.PathParameter("conversation_id")

	h.logger.Info().
		Str("conversation_id", conversationID).
		Msg("get conversation by ID")

	ctx := req.Request.Context()
	record, err := h.repository.GetConversation(ctx, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{"error": "conversation not found"})
			return
		}
		h.logger.Error().Err(err).Msg("failed to get conversation")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, toConversationDetailResponse(record))
}

// GET /api/v1/conversations
func (h *Handler) ListConversations(req *restful.Request, resp *restful.Response) {
	h.logger.Info().Msg("listing all conversations")

	ctx := req.Request.Context()
	summaries, err := h.repository.ListConversations(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list conversations")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	dtos := make([]ConversationSummaryDTO, len(summaries))
	for i, s := range summaries {
		dtos[i] = ConversationSummaryDTO{
			ConversationID: s.ConversationID,
			AgentName:      s.AgentName,
			AgentVersion:   s.AgentVersion,
			TurnCount:      s.TurnCount,
			FinalScore:     s.FinalScore,
			Verdict:        s.Verdict,
			CreatedAt:      s.CreatedAt,
		}
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, ConversationListResponse{
		Conversations: dtos,
		Total:         len(dtos),
	})
}

// GET /api/v1/metrics/health?window=7d
func (h *Handler) HealthMetrics(req *restful.Request, resp *restful.Response) {
	window := req.QueryParameter("window")
	if window == "" {
		window = "7d"
	}

	dur, err := parseWindow(window)
	if err != nil {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid window %q: use format like 7d or 24h", window),
		})
		return
	}

	since := time.Now().UTC().Add(-dur)
	ctx := req.Request.Context()
	data, err := h.repository.HealthMetrics(ctx, since)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to compute health metrics")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, HealthMetricsResponse{
		Window:           window,
		TotalEvaluations: data.TotalEvaluations,
		AvgConfidence:    data.AvgConfidence,
	})
}

// GET /api/v1/health
func (h *Handler) Health(req *restful.Request, resp *restful.Response) {
	_ = resp.WriteHeaderAndEntity(http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: "2.0.0",
	})
}

// parseWindow parses a duration string like "7d" or "24h".
func parseWindow(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("too short")
	}
	unit := s[len(s)-1]
	value := s[:len(s)-1]
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid number")
	}
	switch unit {
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q, use h or d", string(unit))
	}
}
