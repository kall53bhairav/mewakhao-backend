package http

import (
	cartHttp "ecom/internal/cart/http"
	categoryHttp "ecom/internal/category/http"
	orderHttp "ecom/internal/order/http"
	productHttp "ecom/internal/product/http"
	userHttp "ecom/internal/user/http"
	"ecom/pkg/config"
	"ecom/pkg/dbs"
	"ecom/pkg/response"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/logger"
	"github.com/quangdangfit/gocommon/validation"
)

type Server struct {
	engine    *gin.Engine
	cfg       *config.Schema
	validator validation.Validation
	db        *dbs.Database
}

func NewServer(validator validation.Validation, db *dbs.Database) *Server {
	engine := gin.Default()
	engine.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000", "https://mewakhao-frontend.vercel.app", "https://mewakhao.in", "https://www.mewakhao.in"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))
	return &Server{
		engine:    engine,
		cfg:       config.GetEnv(),
		validator: validator,
		db:        db,
	}
}

func (s Server) Run() error {
	_ = s.engine.SetTrustedProxies(nil)
	if s.cfg.Environment == config.ProductionEnv {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := s.MapRoutes(); err != nil {
		log.Fatalf("MapRoutes Error: %v", err)
	}
	// s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	s.engine.GET("/health", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, nil)
		return
	})

	// Start http server
	logger.Info("HTTP server is listening on PORT: ", s.cfg.HttpPort)
	if err := s.engine.Run(fmt.Sprintf(":%d", s.cfg.HttpPort)); err != nil {
		log.Fatalf("Running HTTP server: %v", err)
	}

	return nil
}

func (s Server) GetEngine() *gin.Engine {
	return s.engine
}

func (s Server) MapRoutes() error {
	prefix := s.engine.Group("/api/v1")
	userHttp.Routes(prefix, s.db, s.validator)
	categoryHttp.Routes(prefix, s.db, s.validator)
	productHttp.Routes(prefix, s.db, s.validator)
	cartHttp.Routes(prefix, s.db, s.validator)
	orderHttp.Routes(prefix, s.db, s.validator)
	return nil
}
