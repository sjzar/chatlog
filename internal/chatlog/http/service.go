package http

import (
	"context"
	"net/http"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/chatlog/database"
	"github.com/sjzar/chatlog/internal/chatlog/mcp"
	"github.com/sjzar/chatlog/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const (
	DefalutHTTPAddr = "127.0.0.1:5030"
)

type Service struct {
	ctx *ctx.Context
	db  *database.Service
	mcp *mcp.Service

	router *gin.Engine
	server *http.Server
}

func NewService(ctx *ctx.Context, db *database.Service, mcp *mcp.Service) *Service {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Handle error from SetTrustedProxies
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Err(err).Msg("Failed to set trusted proxies")
	}

	// Middleware
	router.Use(
		errors.RecoveryMiddleware(),
		errors.ErrorHandlerMiddleware(),
		gin.LoggerWithWriter(log.Logger),
	)

	s := &Service{
		ctx:    ctx,
		db:     db,
		mcp:    mcp,
		router: router,
	}

	s.initRouter()
	return s
}

func (s *Service) Start() error {

	if s.ctx.HTTPAddr == "" {
		s.ctx.HTTPAddr = DefalutHTTPAddr
	}

	s.server = &http.Server{
		Addr:    s.ctx.HTTPAddr,
		Handler: s.router,
	}

	go func() {
		// Handle error from Run
		if err := s.server.ListenAndServe(); err != nil {
			log.Err(err).Msg("Failed to start HTTP server")
		}
	}()

	log.Info().Msg("Starting HTTP server on " + s.ctx.HTTPAddr)

	return nil
}

func (s *Service) ListenAndServe() error {

	if s.ctx.HTTPAddr == "" {
		s.ctx.HTTPAddr = DefalutHTTPAddr
	}

	s.server = &http.Server{
		Addr:    s.ctx.HTTPAddr,
		Handler: s.router,
	}

	log.Info().Msg("Starting HTTP server on " + s.ctx.HTTPAddr)
	return s.server.ListenAndServe()
}

func (s *Service) Stop() error {

	if s.server == nil {
		return nil
	}

	// 使用超时上下文优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Debug().Err(err).Msg("Failed to shutdown HTTP server")
		return nil
	}

	log.Info().Msg("HTTP server stopped")
	return nil
}

func (s *Service) GetRouter() *gin.Engine {
	return s.router
}
