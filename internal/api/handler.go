package api

import (
	"database/sql"
	"errors"
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
	executor      *executor.Executor
	judgeExecutor *executor.JudgeExecutor
	repository    storage.Repository
	logger        *zerolog.Logger
}

func NewHandler(
	executor *executor.Executor,
	judgeExecutor *executor.JudgeExecutor,
	repository storage.Repository,
	logger *zerolog.Logger,
) *Handler {
	return &Handler{
		executor:      executor,
		judgeExecutor: judgeExecutor,
		repository:    repository,
		logger:        logger,
	}
}

// POST /api/v1/evaluate
// Body: EvaluateRequest
// Returns: EvaluationResult
func (h *Handler) Evaluate(req *restful.Request, resp *restful.Response) {
	var evalRequest models.EvaluationRequest
	if err := req.ReadEntity(&evalRequest); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse request body")
		middleware.HandleError(resp, err, http.StatusBadRequest)
		return
	}

	// EvaluationRequest validation
	if err := validateEvaluationRequest(evalRequest); err != nil {
		h.logger.Warn().Err(err).Msg("Request validation failed")
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("event_id", evalRequest.EventID).
		Str("conversation_id", evalRequest.ConversationID).
		Str("event_type", string(evalRequest.EventType)).
		Str("agent_name", string(evalRequest.Agent.Name)).
		Msg("Start evaluation")

	ctx := req.Request.Context()
	evaluationContext := normalize(evalRequest)

	evalResult := h.executor.Execute(ctx, evaluationContext)

	h.logger.Info().
		Str("event_id", evalResult.ID).
		Str("conversation_id", evalResult.ConversationID).
		Str("verdict", string(evalResult.Verdict)).
		Float64("confidence", evalResult.Confidence).
		Msg("Evaluation complete")

	_ = resp.WriteHeaderAndEntity(http.StatusOK, evalResult)
}

// POST /api/v1/evaluate/judge/{judge_name}
func (h *Handler) EvaluateSingleJudge(req *restful.Request, resp *restful.Response) {
	judgeName := req.PathParameter("judge_name")
	thresholdStr := req.QueryParameter("threshold")
	threshold := 0.7
	if thresholdStr != "" {
		if parsedThreshold, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			if parsedThreshold >= 0.0 && parsedThreshold <= 1.0 {
				threshold = parsedThreshold
			} else {
				h.logger.Warn().Str("threshold", thresholdStr).Msg("Invalid threshold, using default 0.7")
			}
		}
	}

	var evalRequest models.EvaluationRequest

	if err := req.ReadEntity(&evalRequest); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse request body")
		middleware.HandleError(resp, err, http.StatusBadRequest)
		return
	}

	// EvaluationRequest validation
	if err := validateEvaluationRequest(evalRequest); err != nil {
		h.logger.Warn().Err(err).Msg("Request validation failed")
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("event_id", evalRequest.EventID).
		Str("conversation_id", evalRequest.ConversationID).
		Str("judge_name", judgeName).
		Float64("threshold", threshold).
		Str("event_type", string(evalRequest.EventType)).
		Str("agent_name", string(evalRequest.Agent.Name)).
		Msg("Start evaluation")

	ctx := req.Request.Context()
	evalContext := normalize(evalRequest)

	evalResult, err := h.judgeExecutor.Execute(ctx, judgeName, threshold, evalContext)

	if err != nil {
		if errors.Is(err, executor.ErrJudgeNotFound) {
			h.logger.Warn().Str("judge_name", judgeName).Msg("Judge not found")
			_ = resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{
				"error": "judge not found: " + judgeName,
			})
			return
		}

		h.logger.Error().Err(err).Msg("Evaluation failed")
		_ = resp.WriteHeaderAndEntity(http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
		return
	}

	h.logger.Info().
		Str("judge_name", judgeName).
		Float64("threshold", threshold).
		Str("event_id", evalResult.ID).
		Str("conversation_id", evalResult.ConversationID).
		Str("verdict", string(evalResult.Verdict)).
		Float64("confidence", evalResult.Confidence).
		Msg("Evaluation complete")

	_ = resp.WriteHeaderAndEntity(http.StatusOK, evalResult)

}

// GET /api/v1/results
func (h *Handler) QueryResults(req *restful.Request, resp *restful.Response) {
	agentName := req.QueryParameter("agent_name")
	verdict := req.QueryParameter("verdict")
	limitStr := req.QueryParameter("limit")
	offsetStr := req.QueryParameter("offset")

	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	filters := models.QueryFilters{
		AgentName: agentName,
		Verdict:   verdict,
		Limit:     limit,
		Offset:    offset,
	}

	h.logger.Info().
		Str("agent_name", agentName).
		Str("verdict", verdict).
		Int("limit", limit).
		Int("offset", offset).
		Msg("Query results")

	ctx := req.Request.Context()
	evaluations, total, err := h.repository.Query(ctx, filters)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to query results")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	// Convert to DTOs
	dtos := toEvaluationDTOs(evaluations)

	response := QueryResultsResponse{
		Results: dtos,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Count:   len(dtos),
		HasMore: offset+len(dtos) < total,
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, response)
}

// GET /api/v1/results/{event_id}
func (h *Handler) GetResultByID(req *restful.Request, resp *restful.Response) {
	eventID := req.PathParameter("event_id")

	h.logger.Info().
		Str("event_id", eventID).
		Msg("Get result by ID")

	ctx := req.Request.Context()
	evaluation, err := h.repository.QueryById(ctx, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn().Str("event_id", eventID).Msg("Result not found")
			_ = resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{
				"error": "result not found",
			})
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get result")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	// Convert to DTO
	dto := toEvaluationDTO(*evaluation)

	response := EvaluationResponse{
		Evaluation: dto,
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, response)
}

// GET /api/v1/conversations/{conversation_id}
func (h *Handler) GetConversationID(req *restful.Request, resp *restful.Response) {
	conversationID := req.PathParameter("conversation_id")

	h.logger.Info().
		Str("conversation_id", conversationID).
		Msg("Get conversations by ID")

	ctx := req.Request.Context()
	evaluations, err := h.repository.GetConversation(ctx, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn().Str("conversation_id", conversationID).Msg("Result not found")
			_ = resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{
				"error": "result not found",
			})
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get result")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	// Convert to DTO
	conversationDetailResponse := toConversationDetailResponse(evaluations, conversationID)

	_ = resp.WriteHeaderAndEntity(http.StatusOK, conversationDetailResponse)
}

// GET /api/v1/conversations/{conversation_id}
func (h *Handler) ListConversations(req *restful.Request, resp *restful.Response) {

	h.logger.Info().
		Msg("Getting all conversations")

	ctx := req.Request.Context()
	conversationSummaries, err := h.repository.ListConversations(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{
				"error": "couldn't find any conversation",
			})
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get result")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	// Convert to DTO
	conversationSummaryDTOs := toConversationSummaryDTOs(conversationSummaries)

	_ = resp.WriteHeaderAndEntity(http.StatusOK, conversationSummaryDTOs)
}

// Health handler GET API /api/v1/health
func (h *Handler) Health(req *restful.Request, resp *restful.Response) {
	healthResponse := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, healthResponse)
}

func normalize(req models.EvaluationRequest) models.EvaluationContext {
	return models.EvaluationContext{
		RequestID:      req.EventID,
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		Query:          req.Interaction.UserQuery,
		Context:        req.Interaction.Context,
		Answer:         req.Interaction.Answer,
		ExpectedOutput: req.Interaction.ExpectedOutput,
		CreatedAt:      time.Now(),
	}
}

func validateEvaluationRequest(evalRequest models.EvaluationRequest) error {
	if evalRequest.EventID == "" {
		return errors.New("event_id is required")
	}
	if evalRequest.Interaction.UserQuery == "" {
		return errors.New("user_query is required")
	}
	if evalRequest.Interaction.Answer == "" {
		return errors.New("answer is required")
	}
	return nil
}
