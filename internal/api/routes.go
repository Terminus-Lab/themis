package api

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/Terminus-Lab/themis/internal/api/middleware"
	"github.com/Terminus-Lab/themis/internal/models"
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

	container.Add(ws)
}
