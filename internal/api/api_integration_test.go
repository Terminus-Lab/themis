package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Terminus-Lab/themis/internal/aggregator"
	"github.com/Terminus-Lab/themis/internal/api"
	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/judge"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/Terminus-Lab/themis/internal/llm/aws"
	"github.com/Terminus-Lab/themis/internal/llm/azure"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/prechecks"
	"github.com/Terminus-Lab/themis/internal/storage/sqlite"
	"github.com/emicklei/go-restful/v3"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// Custom flag for running integration tests with real LLM calls
var runIntegration = flag.Bool("integration", false, "Run integration tests with real LLM API calls")

/*
TEST 1: Health Check
Purpose: Verify the API is running and responds to health checks
*/
func TestAPI_Health(t *testing.T) {
	// Build real API with REAL LLM client
	container := setupTestAPI(t)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	// Create recorder to capture response
	recorder := httptest.NewRecorder()

	// Execute: Send request through real API
	container.ServeHTTP(recorder, req)

	// Assert: Check response
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	var response api.HealthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response.Status)
	}
}

/*
TEST 2: Full Evaluation - Happy Path
Purpose: Test complete evaluation pipeline with all judges
*/
func TestAPI_Evaluate_FullPipeline(t *testing.T) {
	// Setup
	container := setupTestAPI(t)

	// Create evaluation request (happy case: good answer)
	evalRequest := models.EvaluationRequest{
		EventID: "test-001",
		Interaction: models.Interaction{
			UserQuery: "What is the capital of France?",
			Answer:    "The capital of France is Paris.",
			Context:   "France is a country in Europe. Paris is its capital city.",
		},
	}

	// Marshal to JSON
	body, err := json.Marshal(evalRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Create recorder
	recorder := httptest.NewRecorder()

	// Execute
	container.ServeHTTP(recorder, req)

	// Assert
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var result models.EvaluationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify result structure
	if result.ID != "test-001" {
		t.Errorf("Expected ID 'test-001', got '%s'", result.ID)
	}

	if len(result.Stages) == 0 {
		t.Error("Expected stages in result, got none")
	}

	// Verify both prechecks and judges ran
	hasPrechecks := false
	hasJudges := false
	for _, stage := range result.Stages {
		if stage.Name == "length-checker" || stage.Name == "overlap-checker" || stage.Name == "format-checker" {
			hasPrechecks = true
		}
		if stage.Name == "relevance-judge" || stage.Name == "coherence-judge" {
			hasJudges = true
		}
	}

	if !hasPrechecks {
		t.Error("Expected precheck stages")
	}
	if !hasJudges {
		t.Error("Expected judge stages")
	}

	// Verify verdict is set
	if result.Verdict == "" {
		t.Error("Expected verdict to be set")
	}

	// Verify confidence is in valid range
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Expected confidence in [0,1], got %f", result.Confidence)
	}

	// Log results
	t.Logf("Full Pipeline Result: verdict=%s, confidence=%.3f, stages=%d",
		result.Verdict, result.Confidence, len(result.Stages))
	t.Logf("Metrics: weighted_avg=%.3f, harmonic_mean=%.3f, median=%.3f, product=%.3f",
		result.Metrics.Stage2WeightedAvg, result.Metrics.Stage2HarmonicMean,
		result.Metrics.Stage2Median, result.Metrics.Stage2WeightedProduct)
}

/*
TEST 3: Single Judge Evaluation
Purpose: Test evaluating with only one judge (faster endpoint)
*/
func TestAPI_EvaluateSingleJudge_Relevance(t *testing.T) {
	// Setup
	container := setupTestAPI(t)

	// Create evaluation request
	evalRequest := models.EvaluationRequest{
		EventID: "test-002",
		Interaction: models.Interaction{
			UserQuery: "What is AI?",
			Answer:    "AI stands for Artificial Intelligence.",
		},
	}

	body, _ := json.Marshal(evalRequest)

	// Create HTTP request for relevance judge only
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/evaluate/judge/relevance?threshold=0.7",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	// Execute
	container.ServeHTTP(recorder, req)

	// Assert
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var result models.EvaluationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify only relevance judge ran
	if len(result.Stages) != 1 {
		t.Errorf("Expected 1 stage (relevance-judge), got %d", len(result.Stages))
	}

	if len(result.Stages) > 0 && result.Stages[0].Name != "relevance-judge" {
		t.Errorf("Expected 'relevance-judge', got '%s'", result.Stages[0].Name)
	}

	// Log results
	if len(result.Stages) > 0 {
		t.Logf("Relevance Judge: verdict=%s, confidence=%.3f, score=%.3f",
			result.Verdict, result.Confidence, result.Stages[0].Score)
	}
}

/*
TEST 4: Faithfulness Judge (requires context)
Purpose: Test a judge that requires context field
*/
func TestAPI_EvaluateSingleJudge_Faithfulness(t *testing.T) {
	// Setup
	container := setupTestAPI(t)

	// Create evaluation request WITH context
	evalRequest := models.EvaluationRequest{
		EventID: "test-003",
		Interaction: models.Interaction{
			UserQuery: "What does the documentation say about Redis?",
			Answer:    "Redis is used for streaming messages.",
			Context:   "The system uses Redis Streams for message queue functionality.", // Required!
		},
	}

	body, _ := json.Marshal(evalRequest)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/evaluate/judge/faithfulness",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	// Execute
	container.ServeHTTP(recorder, req)

	// Assert
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var result models.EvaluationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(result.Stages) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(result.Stages))
	}

	if len(result.Stages) > 0 && result.Stages[0].Name != "faithfulness-judge" {
		t.Errorf("Expected 'faithfulness-judge', got '%s'", result.Stages[0].Name)
	}

	// Log results
	if len(result.Stages) > 0 {
		t.Logf("Faithfulness Judge: verdict=%s, confidence=%.3f, score=%.3f",
			result.Verdict, result.Confidence, result.Stages[0].Score)
	}
}

/*
TEST 5: Multiple Judges at Once
Purpose: Test evaluating with multiple specific judges
*/
func TestAPI_Evaluate_MultipleJudges(t *testing.T) {
	// Setup
	container := setupTestAPI(t)

	// Test different judges with the same request
	judges := []string{"relevance", "coherence", "completeness", "instruction"}

	evalRequest := models.EvaluationRequest{
		EventID: "test-004",
		Interaction: models.Interaction{
			UserQuery: "Explain Go interfaces in one sentence.",
			Answer:    "Go interfaces define method signatures that types must implement.",
		},
	}

	for _, judgeName := range judges {
		body, _ := json.Marshal(evalRequest)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/evaluate/judge/"+judgeName,
			bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		container.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Judge %s failed with status %d", judgeName, recorder.Code)
			continue
		}

		var result models.EvaluationResult
		json.Unmarshal(recorder.Body.Bytes(), &result)

		// Log results
		if len(result.Stages) > 0 {
			t.Logf("Judge %s: verdict=%s, confidence=%.3f, score=%.3f",
				judgeName, result.Verdict, result.Confidence, result.Stages[0].Score)
		}
	}
}

/*
TEST 6: Early Exit Scenario
Purpose: Test that poor responses trigger early exit (skip LLM judges)
*/
func TestAPI_Evaluate_EarlyExit(t *testing.T) {
	// Setup
	container := setupTestAPI(t)

	// Create request with very poor answer (should fail prechecks)
	evalRequest := models.EvaluationRequest{
		EventID: "test-005",
		Interaction: models.Interaction{
			UserQuery: "Explain quantum computing, its applications, and future implications?",
			Answer:    "Yes.", // Very short answer = early exit
		},
	}

	body, _ := json.Marshal(evalRequest)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	// Execute
	container.ServeHTTP(recorder, req)

	// Assert
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	var result models.EvaluationResult
	json.Unmarshal(recorder.Body.Bytes(), &result)

	// Verify early exit: should have prechecks but NO judges
	hasPrechecks := false
	hasJudges := false
	for _, stage := range result.Stages {
		if stage.Name == "length-checker" {
			hasPrechecks = true
		}
		if stage.Name == "relevance-judge" {
			hasJudges = true
		}
	}

	if !hasPrechecks {
		t.Error("Expected prechecks to run")
	}

	if hasJudges {
		t.Error("Expected early exit - judges should NOT run for poor answer")
	}

	// Should be a fail verdict
	if result.Verdict != models.VerdictFail {
		t.Errorf("Expected 'fail' verdict for early exit, got '%s'", result.Verdict)
	}

	// Log results
	t.Logf("Early Exit Result: verdict=%s, confidence=%.3f, stages=%d (prechecks only)",
		result.Verdict, result.Confidence, len(result.Stages))
}

// setupTestAPI creates API with REAL LLM client
func setupTestAPI(t *testing.T) *restful.Container {
	// Check if integration flag is set
	if !*runIntegration {
		t.Skip("Skipping integration test - use 'go test -integration' to run with real LLM API calls")
	}

	// Load environment variables
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Warning: No .env file found, using environment variables")
	}

	// Set config path
	os.Setenv("JUDGES_CONFIG_PATH", "../../configs/judges.yaml")

	// Determine which LLM provider to use
	provider := os.Getenv("DEFAULT_LLM_PROVIDER")
	if provider == "" {
		provider = "bedrock" // Default to Bedrock
	}

	ctx := context.Background()
	logger := zerolog.Nop()

	// Create REAL LLM client (not mocked!)
	var registry *llm.LLMClientRegistry

	switch provider {
	case "bedrock":
		region := os.Getenv("AWS_REGION")
		modelID := os.Getenv("DEFAULT_MODEL_ID")
		modelFamily := os.Getenv("DEFAULT_MODEL_FAMILY")
		if modelFamily == "" {
			modelFamily = "anthropic"
		}

		if region == "" || modelID == "" {
			t.Skip("Skipping real Bedrock integration - AWS_REGION or DEFAULT_MODEL_ID not set")
		}

		llmClient, err := aws.NewClient(ctx, region, modelID)
		if err != nil {
			t.Fatalf("Failed to create Bedrock client: %v", err)
		}
		t.Logf("Using REAL AWS Bedrock: region=%s, model=%s", region, modelID)

		registry = llm.NewLLMClientRegistry(map[llm.LLMFamily]map[string]llm.LLMClient{
			llm.LLMFamily(modelFamily): {
				modelID: llmClient,
			},
		})

	case "openai":
		apiKey := os.Getenv("OPEN_AI_KEY")
		modelID := os.Getenv("OPEN_AI_MODEL_ID")
		azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
		modelFamily := os.Getenv("DEFAULT_MODEL_FAMILY")
		if modelFamily == "" {
			modelFamily = "openai"
		}

		if apiKey == "" || modelID == "" || azureEndpoint == "" {
			t.Skip("Skipping real Azure OpenAI integration - OPEN_AI_KEY, OPEN_AI_MODEL_ID or AZURE_OPENAI_ENDPOINT not set")
		}

		llmClient, err := azure.NewClient(apiKey, modelID, azureEndpoint)
		if err != nil {
			t.Fatalf("Failed to create Azure OpenAI client: %v", err)
		}
		t.Logf("Using REAL Azure OpenAI GPT: model=%s, endpoint=%s", modelID, azureEndpoint)

		registry = llm.NewLLMClientRegistry(map[llm.LLMFamily]map[string]llm.LLMClient{
			llm.LLMFamily(modelFamily): {
				modelID: llmClient,
			},
		})

	default:
		t.Fatalf("Unknown LLM provider: %s (expected 'bedrock' or 'openai')", provider)
	}

	// Judges with REAL LLM client
	judgesConfig, err := config.LoadJudgesConfig()
	if err != nil {
		t.Fatalf("Failed to load judges config: %v", err)
	}

	judgePool := judge.NewJudgePool(registry, &logger)
	judges, err := judgePool.BuildFromConfig(judgesConfig)
	if err != nil {
		t.Fatalf("Failed to build judges: %v", err)
	}

	judgeRunner := judge.NewJudgeRunner(judges, &logger)
	judgeFactory := judge.NewJudgeFactory(judges, &logger)

	// Aggregator Config
	var judgeAggMethod models.AggregationMethod
	aggMethod := os.Getenv("JUDGE_AGGREGATION_METHOD")
	if aggMethod == "" {
		judgeAggMethod = models.MethodWeightedAverage
	} else {
		judgeAggMethod = models.AggregationMethod(aggMethod)
	}

	aggConfig := aggregator.AggregationConfig{
		EnablePrecheck:         true,
		JudgeAggregationMethod: judgeAggMethod,
	}

	// Aggregator
	agg := aggregator.NewAggregator(
		aggregator.Weights{
			PreChecks: 0.3,
			LLMJudge:  0.7,
		},
		aggregator.VerdictThresholds{
			Pass:   0.8,
			Review: 0.5,
		},
		aggConfig,
		&logger,
	)

	// PreChecks
	stageRunner := prechecks.NewStageRunner([]prechecks.Checker{
		&prechecks.LengthChecker{},
		&prechecks.OverlapChecker{MinOverlapThreshold: 0.3},
		&prechecks.FormatChecker{},
	})
	repository := setupTestRepository(t, &logger)
	// Executors
	exec := executor.NewExecutor(stageRunner, repository, judgeRunner, agg, 0.2, &logger)
	judgeExec := executor.NewJudgeExecutor(judgeFactory, repository, &logger)

	// API Handler
	handler := api.NewHandler(exec, judgeExec, &logger)

	// REST Container
	container := restful.NewContainer()
	api.RegisterRoutes(container, handler)

	return container
}

func setupTestRepository(t *testing.T, logger *zerolog.Logger) *sqlite.EvalRepository {
	t.Helper()

	db, err := sqlite.New(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return sqlite.NewEvalRepository(db, logger)
}
