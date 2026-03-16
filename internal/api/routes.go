package api

import (
	"github.com/Terminus-Lab/themis/internal/api/middleware"
	"github.com/Terminus-Lab/themis/internal/models"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

func RegisterRoutes(container *restful.Container, handler *Handler) {
	ws := new(restful.WebService)

	ws.
		Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// Health endpoint
	ws.
		Route(ws.GET("health").
			To(handler.Health).
			Doc("Health check").
			Metadata(restfulspec.KeyOpenAPITags, []string{"health"}).
			Writes(HealthResponse{}).
			Returns(200, "OK", HealthResponse{}))

	ws.
		Route(ws.POST("/evaluate").
			To(handler.Evaluate).
			Doc("Evaluate agent response").
			Metadata(restfulspec.KeyOpenAPITags, []string{"evaluate"}).
			Reads(models.EvaluationRequest{}).
			Writes(models.EvaluationResult{}).
			Returns(200, "OK", models.EvaluationResult{}).
			Returns(400, "Bad Request", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.POST("/evaluate/judge/{judge_name}").
			To(handler.EvaluateSingleJudge).
			Doc("Evaluate with a single judge").
			Metadata(restfulspec.KeyOpenAPITags, []string{"evaluate"}).
			Param(ws.PathParameter("judge_name", "Judge name (relevance, faithfulness, coherence, completeness, instruction)").DataType("string")).
			Param(ws.QueryParameter("threshold", "Pass/fail threshold (0.0-1.0, default: 0.7)").DataType("number").Required(false)).
			Reads(models.EvaluationRequest{}).
			Writes(models.EvaluationResult{}).
			Returns(200, "OK", models.EvaluationResult{}).
			Returns(400, "Bad Request", middleware.ErrorResponse{}).
			Returns(404, "Judge Not Found", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.GET("/results").
			To(handler.QueryResults).
			Doc("Query evaluation results with filters").
			Metadata(restfulspec.KeyOpenAPITags, []string{"results"}).
			Param(ws.QueryParameter("agent_name", "Filter by agent name").DataType("string").Required(false)).
			Param(ws.QueryParameter("verdict", "Filter by verdict (pass, review, fail)").DataType("string").Required(false)).
			Param(ws.QueryParameter("limit", "Number of results to return (default: 50)").DataType("number").Required(false)).
			Param(ws.QueryParameter("offset", "Number of results to skip (default: 0)").DataType("number").Required(false)).
			Writes(QueryResultsResponse{}).
			Returns(200, "OK", QueryResultsResponse{}).
			Returns(400, "Bad Request", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.GET("/results/{event_id}").
			To(handler.GetResultByID).
			Doc("Get evaluation result by event ID").
			Metadata(restfulspec.KeyOpenAPITags, []string{"results"}).
			Param(ws.PathParameter("event_id", "Event ID").DataType("string")).
			Writes(EvaluationResponse{}).
			Returns(200, "OK", EvaluationResponse{}).
			Returns(404, "Not Found", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.GET("/conversations/{conversation_id}").
			To(handler.GetConversationID).
			Doc("Get conversation details with all turns").
			Metadata(restfulspec.KeyOpenAPITags, []string{"conversations"}).
			Param(ws.PathParameter("conversation_id", "Conversation ID").DataType("string")).
			Writes(ConversationDetailResponse{}).
			Returns(200, "OK", ConversationDetailResponse{}).
			Returns(404, "Not Found", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.GET("/conversations").
			To(handler.ListConversations).
			Doc("List all conversations with summary metrics").
			Metadata(restfulspec.KeyOpenAPITags, []string{"conversations"}).
			Writes(ConversationListResponse{}).
			Returns(200, "OK", ConversationListResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.GET("/metrics/health").
			To(handler.HealthMetrics).
			Doc("Proxy health metrics for drift detection (no human annotation required)").
			Metadata(restfulspec.KeyOpenAPITags, []string{"metrics"}).
			Param(ws.QueryParameter("window", "Time window: e.g. 7d, 30d, 24h (default: 7d)").DataType("string").Required(false)).
			Writes(HealthMetricsResponse{}).
			Returns(200, "OK", HealthMetricsResponse{}).
			Returns(400, "Bad Request", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	ws.
		Route(ws.POST("/validation/sample/download").
			To(handler.DownloadSample).
			Doc("Sample evaluation results from a date range and download as JSONL").
			Metadata(restfulspec.KeyOpenAPITags, []string{"validation"}).
			Reads(SampleRequest{}).
			Returns(200, "OK (JSONL stream)", nil).
			Returns(400, "Bad Request", middleware.ErrorResponse{}).
			Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	container.Add(ws)
}
