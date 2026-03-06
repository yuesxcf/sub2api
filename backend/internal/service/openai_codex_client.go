package service

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
)

func isOfficialCodexClientRequest(c *gin.Context, cfg *config.Config) bool {
	if cfg != nil && cfg.Gateway.ForceCodexCLI {
		return true
	}
	if c == nil || c.Request == nil {
		return false
	}
	return isOfficialCodexClientHeaders(c.Request.Header, cfg)
}

func isOfficialCodexClientHeaders(headers http.Header, cfg *config.Config) bool {
	if cfg != nil && cfg.Gateway.ForceCodexCLI {
		return true
	}
	if headers == nil {
		return false
	}

	userAgent := strings.TrimSpace(headers.Get("User-Agent"))
	originator := strings.TrimSpace(headers.Get("originator"))
	return openai.IsCodexOfficialClientRequest(userAgent) || openai.IsCodexOfficialClientOriginator(originator)
}
