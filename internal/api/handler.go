package api

import (
	"database/sql"
	"encoding/json"
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
	executor             *executor.Executor
	judgeExecutor        *executor.JudgeExecutor
	conversationExecutor *executor.ConversationExecutor
	repository           storage.Repository
	logger               *zerolog.Logger
}

func NewHandler(
	executor *executor.Executor,
	judgeExecutor *executor.JudgeExecutor,
	conversationExecutor *executor.ConversationExecutor,
	repository storage.Repository,
	logger *zerolog.Logger,
) *Handler {
	return &Handler{
		executor:             executor,
		judgeExecutor:        judgeExecutor,
		conversationExecutor: conversationExecutor,
		repository:           repository,
		logger:               logger,
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
	turns, err := h.repository.GetConversation(ctx, conversationID)
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

	if len(turns) == 0 {
		h.logger.Warn().Str("conversation_id", conversationID).Msg("Conversation not found")
		_ = resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{
			"error": "conversation not found",
		})
		return
	}

	// Calculate average confidence
	totalConfidence := 0.0
	for _, turn := range turns {
		totalConfidence += turn.Confidence
	}
	avgConfidence := totalConfidence / float64(len(turns))

	// Build conversation detail
	detail := storage.ConversationDetail{
		ConversationID: conversationID,
		TurnCount:      len(turns),
		AvgConfidence:  avgConfidence,
		AgentName:      turns[0].AgentName,
		AgentVersion:   turns[0].AgentVersion,
		Turns:          turns,
	}

	// Convert to DTO
	conversationDetailResponse := toConversationDetailResponse(detail)

	_ = resp.WriteHeaderAndEntity(http.StatusOK, conversationDetailResponse)
}

// GET /api/v1/conversations
func (h *Handler) ListConversations(req *restful.Request, resp *restful.Response) {

	h.logger.Info().
		Msg("Listing all conversations")

	ctx := req.Request.Context()
	conversationSummaries, err := h.repository.ListConversations(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list conversations")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	// Convert to DTO
	conversationSummaryDTOs := toConversationSummaryDTOs(conversationSummaries)

	response := ConversationListResponse{
		Conversations: conversationSummaryDTOs,
		Total:         len(conversationSummaryDTOs),
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, response)
}

// GET /api/v1/metrics/health?window=7d
// Supported window units: h (hours), d (days). Default: 7d.
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
		h.logger.Error().Err(err).Msg("Failed to compute health metrics")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, HealthMetricsResponse{
		Window:           window,
		TotalEvaluations: data.TotalEvaluations,
		AvgConfidence:    data.AvgConfidence,
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

// POST /api/v1/validation/sample/events/download
// Samples a percentage of individual event evaluations from a date range and returns them as JSONL.
func (h *Handler) DownloadEventsSample(req *restful.Request, resp *restful.Response) {
	filters, ok := h.parseSampleRequest(req, resp)
	if !ok {
		return
	}

	ctx := req.Request.Context()
	evaluations, err := h.repository.Sample(ctx, filters)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to sample events")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	h.logger.Info().Int("sampled_count", len(evaluations)).Msg("Events sample complete, streaming JSONL")

	resp.Header().Set("Content-Type", "application/x-ndjson")
	resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"events-sample-%s.jsonl\"", time.Now().Format("20060102-150405")))
	resp.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(resp.ResponseWriter)
	for _, e := range evaluations {
		record := EventSampleRecord{
			EventID:        e.EventID,
			ConversationID: e.ConversationID,
		}
		record.Agent.Name = e.AgentName
		record.Agent.Version = e.AgentVersion
		record.Interaction.UserQuery = e.UserQuery
		record.Interaction.Context = e.Context
		record.Interaction.Answer = e.Answer

		if err := encoder.Encode(record); err != nil {
			h.logger.Error().Err(err).Msg("Failed to encode event record to JSONL")
			return
		}
	}
}

// POST /api/v1/validation/sample/conversations/download
// Samples a percentage of whole conversations from a date range and returns them as JSONL.
// Each line is a full conversation (all turns grouped), suitable for conversation-level annotation.
func (h *Handler) DownloadConversationsSample(req *restful.Request, resp *restful.Response) {
	filters, ok := h.parseSampleRequest(req, resp)
	if !ok {
		return
	}

	ctx := req.Request.Context()
	conversations, err := h.repository.SampleConversations(ctx, filters)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to sample conversations")
		middleware.HandleError(resp, err, http.StatusInternalServerError)
		return
	}

	h.logger.Info().Int("sampled_count", len(conversations)).Msg("Conversations sample complete, streaming JSONL")

	resp.Header().Set("Content-Type", "application/x-ndjson")
	resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"conversations-sample-%s.jsonl\"", time.Now().Format("20060102-150405")))
	resp.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(resp.ResponseWriter)
	for _, conv := range conversations {
		turns := make([]ConversationSampleTurn, len(conv.Turns))
		for i, t := range conv.Turns {
			turns[i] = ConversationSampleTurn{
				TurnIndex: t.TurnIndex,
				UserQuery: t.UserQuery,
				Answer:    t.Answer,
				Context:   t.Context,
			}
		}
		record := ConversationSampleRecord{Turns: turns}
		record.ConversationID = conv.ConversationID
		record.Agent.Name = conv.AgentName
		record.Agent.Version = conv.AgentVersion

		if err := encoder.Encode(record); err != nil {
			h.logger.Error().Err(err).Msg("Failed to encode conversation record to JSONL")
			return
		}
	}
}

// parseSampleRequest parses and validates the shared SampleRequest body.
// Returns (filters, true) on success or writes the error response and returns (_, false).
func (h *Handler) parseSampleRequest(req *restful.Request, resp *restful.Response) (storage.SampleFilters, bool) {
	var sampleReq SampleRequest
	if err := req.ReadEntity(&sampleReq); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse sample request body")
		middleware.HandleError(resp, err, http.StatusBadRequest)
		return storage.SampleFilters{}, false
	}

	if sampleReq.StartDate == "" || sampleReq.EndDate == "" {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "start_date and end_date are required"})
		return storage.SampleFilters{}, false
	}

	startDate, err := time.Parse(time.RFC3339, sampleReq.StartDate)
	if err != nil {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid start_date: %s", err.Error())})
		return storage.SampleFilters{}, false
	}

	endDate, err := time.Parse(time.RFC3339, sampleReq.EndDate)
	if err != nil {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid end_date: %s", err.Error())})
		return storage.SampleFilters{}, false
	}

	if endDate.Before(startDate) {
		_ = resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "end_date must be after start_date"})
		return storage.SampleFilters{}, false
	}

	percentage := sampleReq.Percentage
	if percentage <= 0 {
		percentage = 25
	}
	if percentage > 100 {
		percentage = 100
	}

	h.logger.Info().
		Str("start_date", sampleReq.StartDate).
		Str("end_date", sampleReq.EndDate).
		Int("percentage", percentage).
		Msg("Sampling request parsed")

	return storage.SampleFilters{
		StartDate:  startDate,
		EndDate:    endDate,
		Percentage: percentage,
		MinSize:    sampleReq.MinSize,
		MaxSize:    sampleReq.MaxSize,
	}, true
}

// POST /api/v1/evaluate/conversation
func (h *Handler) EvaluateConversation(req *restful.Request, resp *restful.Response) {
	if h.conversationExecutor == nil {
		_ = resp.WriteHeaderAndEntity(http.StatusServiceUnavailable, map[string]string{
			"error": "conversation evaluation is not available: no conversation-scoped judges configured",
		})
		return
	}

	var evalReq ConversationEvalRequest
	if err := req.ReadEntity(&evalReq); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse conversation evaluation request body")
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
		Msg("Start conversation evaluation")

	ctx := req.Request.Context()
	result := h.conversationExecutor.Execute(ctx, convReq)

	stageScores := make([]StageScore, len(result.Stages))
	for i, s := range result.Stages {
		stageScores[i] = StageScore{Name: s.Name, Score: s.Score, Reason: s.Reason, Weight: s.Weight}
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, ConversationEvalResponse{
		ConversationID: result.ConversationID,
		AgentName:      result.AgentName,
		AgentVersion:   result.AgentVersion,
		TurnCount:      result.TurnCount,
		Verdict:        string(result.Verdict),
		Confidence:     result.Confidence,
		Stages:         stageScores,
	})
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
	if evalRequest.ConversationID == "" {
		return errors.New("conversation_id is required")
	}
	if evalRequest.Interaction.UserQuery == "" {
		return errors.New("user_query is required")
	}
	if evalRequest.Interaction.Answer == "" {
		return errors.New("answer is required")
	}
	return nil
}
