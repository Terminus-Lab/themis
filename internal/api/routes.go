package api

import (
	"github.com/Terminus-Lab/themis/internal/api/middleware"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

func RegisterRoutes(container *restful.Container, handler *Handler) {
	ws := new(restful.WebService)

	ws.
		Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// Health
	ws.Route(ws.GET("health").
		To(handler.Health).
		Doc("Health check").
		Metadata(restfulspec.KeyOpenAPITags, []string{"health"}).
		Writes(HealthResponse{}).
		Returns(200, "OK", HealthResponse{}))

	// Evaluate a conversation
	ws.Route(ws.POST("/conversations/evaluate").
		To(handler.EvaluateConversation).
		Doc("Evaluate a full multi-turn conversation").
		Metadata(restfulspec.KeyOpenAPITags, []string{"conversations"}).
		Reads(ConversationEvalRequest{}).
		Writes(ConversationEvalResponse{}).
		Returns(200, "OK", ConversationEvalResponse{}).
		Returns(400, "Bad Request", middleware.ErrorResponse{}).
		Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	// Get a conversation by ID
	ws.Route(ws.GET("/conversations/{conversation_id}").
		To(handler.GetConversation).
		Doc("Get conversation evaluation by ID").
		Metadata(restfulspec.KeyOpenAPITags, []string{"conversations"}).
		Param(ws.PathParameter("conversation_id", "Conversation ID").DataType("string")).
		Writes(ConversationDetailResponse{}).
		Returns(200, "OK", ConversationDetailResponse{}).
		Returns(404, "Not Found", middleware.ErrorResponse{}).
		Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	// List conversations
	ws.Route(ws.GET("/conversations").
		To(handler.ListConversations).
		Doc("List all conversation evaluations").
		Metadata(restfulspec.KeyOpenAPITags, []string{"conversations"}).
		Writes(ConversationListResponse{}).
		Returns(200, "OK", ConversationListResponse{}).
		Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	// Health metrics
	ws.Route(ws.GET("/metrics/health").
		To(handler.HealthMetrics).
		Doc("Health metrics for drift detection").
		Metadata(restfulspec.KeyOpenAPITags, []string{"metrics"}).
		Param(ws.QueryParameter("window", "Time window: e.g. 7d, 30d, 24h (default: 7d)").DataType("string").Required(false)).
		Writes(HealthMetricsResponse{}).
		Returns(200, "OK", HealthMetricsResponse{}).
		Returns(400, "Bad Request", middleware.ErrorResponse{}).
		Returns(500, "Internal Server Error", middleware.ErrorResponse{}))

	container.Add(ws)
}
