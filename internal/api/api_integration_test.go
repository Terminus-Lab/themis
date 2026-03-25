package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Terminus-Lab/themis/internal/api"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/Terminus-Lab/themis/internal/storage/sqlite"
	"github.com/emicklei/go-restful/v3"
	"github.com/rs/zerolog"
)

// setupRealContainer wires a full in-memory setup (requires no LLM env vars).
func setupRealContainer(t *testing.T) (*restful.Container, *sqlite.EvalRepository) {
	t.Helper()

	ctx := context.Background()
	logger := zerolog.Nop()

	db, err := sqlite.New(ctx, ":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite: %v", err)
	}
	if err := db.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
	repo := sqlite.NewEvalRepository(db, &logger)

	// We only need a non-nil handler — evaluator can be nil for endpoints that
	// don't trigger evaluation (health, list, get).
	handler := api.NewHandler(nil, repo, &logger)
	container := restful.NewContainer()
	api.RegisterRoutes(container, handler)
	return container, repo
}

// ─── Tests ─────────────────────────────────────────────────────────────────

func TestAPI_Health(t *testing.T) {
	container, _ := setupRealContainer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp api.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
}

func TestAPI_ListConversations_Empty(t *testing.T) {
	container, _ := setupRealContainer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. body: %s", rec.Code, rec.Body.String())
	}

	var resp api.ConversationListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 conversations, got %d", resp.Total)
	}
}

func TestAPI_GetConversation_NotFound(t *testing.T) {
	container, _ := setupRealContainer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/does-not-exist", nil)
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d. body: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_EvaluateConversation_ValidationErrors(t *testing.T) {
	container, _ := setupRealContainer(t)

	cases := []struct {
		name string
		body string
		code int
	}{
		{
			name: "missing conversation_id",
			body: `{"turns":[{"turn_index":1,"user_query":"hi","answer":"hello"}]}`,
			code: http.StatusBadRequest,
		},
		{
			name: "empty turns",
			body: `{"conversation_id":"conv-1","turns":[]}`,
			code: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/conversations/evaluate",
				bytes.NewBufferString(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			container.ServeHTTP(rec, req)

			if rec.Code != tc.code {
				t.Errorf("expected %d, got %d. body: %s", tc.code, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAPI_HealthMetrics(t *testing.T) {
	container, _ := setupRealContainer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/health?window=7d", nil)
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. body: %s", rec.Code, rec.Body.String())
	}

	var resp api.HealthMetricsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Window != "7d" {
		t.Errorf("expected window '7d', got '%s'", resp.Window)
	}
}

func TestAPI_HealthMetrics_InvalidWindow(t *testing.T) {
	container, _ := setupRealContainer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/health?window=invalid", nil)
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestAPI_EvaluateConversation_Integration tests the full pipeline with real LLM calls.
// Only runs when LLM credentials are available.
func TestAPI_EvaluateConversation_Integration(t *testing.T) {
	cfg := setup.LoadConfig()
	if cfg.OpenAIKey == "" {
		t.Skip("OPEN_AI_KEY not set — skipping integration test")
	}

	ctx := context.Background()
	logger := zerolog.Nop()

	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		t.Skipf("wiring failed (LLM credentials may be missing): %v", err)
	}

	handler := api.NewHandler(deps.ConversationEvaluator, deps.Repository, &logger)
	container := restful.NewContainer()
	api.RegisterRoutes(container, handler)

	body := api.ConversationEvalRequest{
		ConversationID: "integration-test-001",
		Agent: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{Name: "test-agent", Version: "1.0"},
		Turns: []api.ConversationTurnRequest{
			{TurnIndex: 1, UserQuery: "What is the capital of France?", Answer: "The capital of France is Paris."},
			{TurnIndex: 2, UserQuery: "And Germany?", Answer: "The capital of Germany is Berlin."},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/evaluate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. body: %s", rec.Code, rec.Body.String())
	}

	var result api.ConversationEvalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.ConversationID != "integration-test-001" {
		t.Errorf("expected conversation_id 'integration-test-001', got '%s'", result.ConversationID)
	}
	if len(result.TurnResults) != 2 {
		t.Errorf("expected 2 turn results, got %d", len(result.TurnResults))
	}
	if result.FinalScore < 0 || result.FinalScore > 1 {
		t.Errorf("expected final_score in [0,1], got %f", result.FinalScore)
	}
	if result.Verdict == "" {
		t.Error("expected verdict to be set")
	}

	t.Logf("Integration result: verdict=%s, turn_avg=%.3f, holistic=%.3f, final=%.3f",
		result.Verdict, result.TurnAvg, result.HolisticScore, result.FinalScore)
}

// unused import guard
var _ = models.VerdictPass
