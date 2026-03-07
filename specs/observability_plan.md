  ---                                                                                                                                                                                                 
  1. Current Architecture                                                                                                                                                                             
                  
  API Structure:
  - Framework: go-restful/v3
  - Entry point: cmd/api/main.go
  - Handlers: internal/api/handler.go
  - Middleware: internal/api/middleware/ (logging, error handling)
  - Endpoints:
    - GET /api/v1/health - Health check
    - POST /api/v1/evaluate - Full evaluation pipeline
    - POST /api/v1/evaluate/judge/{judge_name} - Single judge evaluation

  ---
  2. Metrics to Track

  HTTP Layer Metrics (Standard)

  1. Request counter - Total HTTP requests
    - Labels: method, path, status_code
    - Metric: themis_http_requests_total
  2. Request duration histogram - Response latency
    - Labels: method, path, status_code
    - Buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10] (seconds)
    - Metric: themis_http_request_duration_seconds
  3. Request size histogram - Request body size
    - Labels: method, path
    - Metric: themis_http_request_size_bytes
  4. Response size histogram - Response body size
    - Labels: method, path
    - Metric: themis_http_response_size_bytes

  Evaluation Pipeline Metrics (Business Logic)

  5. Evaluation counter - Total evaluations run
    - Labels: verdict (pass/review/fail), agent_name, early_exit (true/false)
    - Metric: themis_evaluations_total
  6. Evaluation duration histogram - Time to complete evaluation
    - Labels: verdict, early_exit
    - Buckets: [0.1, 0.5, 1, 2, 5, 10, 15, 30] (seconds)
    - Metric: themis_evaluation_duration_seconds
  7. Stage execution counter - Stage-level tracking
    - Labels: stage_type (precheck/judge), stage_name, result (success/failure/skipped)
    - Metric: themis_stage_executions_total
  8. Judge execution counter - Individual judge tracking
    - Labels: judge_name, result (success/failure/skipped)
    - Metric: themis_judge_executions_total
  9. Confidence score histogram - Final confidence distribution
    - Labels: verdict
    - Buckets: [0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0]
    - Metric: themis_evaluation_confidence
  10. LLM API calls counter - Provider usage tracking
    - Labels: provider (anthropic/openai/openai_platform), model_id, result (success/error)
    - Metric: themis_llm_api_calls_total
  11. LLM API latency histogram - Provider performance
    - Labels: provider, model_id
    - Buckets: [0.1, 0.5, 1, 2, 5, 10, 15] (seconds)
    - Metric: themis_llm_api_duration_seconds

  Error Metrics

  12. Validation errors counter
    - Labels: error_type (missing_field/invalid_format)
    - Metric: themis_validation_errors_total
  13. LLM errors counter
    - Labels: provider, model_id, error_type (timeout/api_error/parse_error)
    - Metric: themis_llm_errors_total

  ---
  3. Implementation Structure

  Package Organization

  internal/
  ├── metrics/
  │   ├── metrics.go           # Metric definitions and initialization
  │   ├── http_metrics.go      # HTTP layer instrumentation
  │   ├── evaluation_metrics.go # Evaluation pipeline instrumentation
  │   └── collector.go         # Custom collectors if needed
  ├── api/
  │   └── middleware/
  │       ├── logging.go       # Existing
  │       ├── errors.go        # Existing
  │       └── prometheus.go    # NEW - Prometheus middleware

  ---
  4. Implementation Steps

  Step 1: Add Prometheus Dependencies

  go get github.com/prometheus/client_golang/prometheus
  go get github.com/prometheus/client_golang/prometheus/promhttp

  Step 2: Create Metrics Package (internal/metrics/)

  File: internal/metrics/metrics.go
  - Define all Prometheus metric collectors (counters, histograms, gauges)
  - Initialize metrics registry
  - Export function: MustRegister() to register all metrics

  File: internal/metrics/http_metrics.go
  - HTTP-level metrics helpers
  - Functions: RecordHTTPRequest(), RecordHTTPDuration()

  File: internal/metrics/evaluation_metrics.go
  - Evaluation pipeline metrics helpers
  - Functions: RecordEvaluation(), RecordStageExecution(), RecordJudgeExecution(), RecordLLMCall()

  Step 3: Create Prometheus Middleware (internal/api/middleware/prometheus.go)

  - go-restful compatible filter
  - Captures: method, path, status code, duration, request/response sizes
  - Integrates with internal/metrics/http_metrics.go

  Step 4: Instrument Handler (internal/api/handler.go)

  - Add metric recording in Evaluate() method
  - Add metric recording in EvaluateSingleJudge() method
  - Record: verdict, confidence, duration, early exit

  Step 5: Instrument Executors (optional, deeper tracking)

  File: internal/executor/agent_executor.go
  - Add metric recording in Execute() method
  - Record stage-level metrics

  File: internal/judge/runner.go
  - Add metric recording per judge execution
  - Record judge success/failure/skip

  File: internal/llm/**/client.go (all providers)
  - Add metric recording in InvokeModel() and InvokeModelWithRetry()
  - Record LLM call duration, success, errors

  Step 6: Expose Metrics Endpoint (cmd/api/main.go)

  - Add new route: GET /metrics (standard Prometheus path)
  - Use promhttp.Handler() to serve metrics

  Step 7: Update Configuration

  - Add to .env: METRICS_ENABLED=true (feature flag)
  - Add to .env: METRICS_PATH=/metrics (configurable endpoint path)
