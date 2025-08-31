package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/you/swarmone/internal/orch"
)

// HTTP server exposing /v1/ask (judge-only consensus) and /health.

type Server struct {
	Router *gin.Engine
	Cfg    *orch.Config
	Keys   orch.Keys
}

func New(cfg *orch.Config, keys orch.Keys) *Server {
	r := gin.Default()
	s := &Server{Router: r, Cfg: cfg, Keys: keys}

	r.POST("/v1/ask", s.ask)
	r.GET("/health", s.health)

	return s
}

// Serve is the entry used by main.
func Serve(cfg *orch.Config, keys orch.Keys) error {
	s := New(cfg, keys)
	return s.Router.Run(cfg.Server.Addr)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"runners": len(s.Cfg.Runners),
	})
}

type askReq struct {
	TemplateID  *string `json:"template_id"`
	Instruction string  `json:"instruction" binding:"required"`
}

func (s *Server) ask(c *gin.Context) {
	var req askReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	ctx := c.Request.Context()

	answer, meta, err := orch.Execute(ctx, s.Cfg, s.Keys, req.Instruction)
	if err != nil && answer == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"detail":           err.Error(),
			"winner_index":     meta.WinnerIndex,
			"runners":          meta.Runners,
			"scores":           meta.Scores,
			"included_indices": meta.IncludedIndices,
			"runner_errors":    meta.RunnerErrors,
			"consensus_id":     meta.ConsensusID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answer":           answer,
		"winner_index":     meta.WinnerIndex,
		"runners":          meta.Runners,
		"scores":           meta.Scores,
		"included_indices": meta.IncludedIndices,
		"runner_errors":    meta.RunnerErrors,
		"consensus_id":     meta.ConsensusID,
	})
}
