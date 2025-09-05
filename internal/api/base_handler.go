package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kingrain94/audit-log-api/internal/utils"
)

type BaseHandler struct{}

func (h *BaseHandler) RequestCtx(ginCtx *gin.Context) context.Context {
	ctx := ginCtx.Request.Context()
	for k, v := range ginCtx.Keys {
		// Convert string keys to proper context key types to avoid collisions
		contextKey := utils.ContextKey(k)
		ctx = context.WithValue(ctx, contextKey, v)
	}
	return ctx
}
