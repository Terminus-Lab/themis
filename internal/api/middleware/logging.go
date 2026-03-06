package middleware

import (
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/rs/zerolog/log"
)

func Logger(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()

	// Log incoming request
	log.Info().
		Str("method", req.Request.Method).
		Str("path", req.Request.URL.Path).
		Str("remote_addr", req.Request.RemoteAddr).
		Msg("Request Started")

	// Process request
	chain.ProcessFilter(req, resp)

	// Log Response
	duration := time.Since(start)
	log.Info().
		Str("method", req.Request.Method).
		Str("path", req.Request.URL.Path).
		Int("status", resp.StatusCode()).
		Dur("duration_ms", duration).
		Msg("Request completed")
}
