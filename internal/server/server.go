package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"authservice/internal/config"
	"authservice/internal/user"
	"github.com/gin-gonic/gin"
)

type authService interface {
	Register(ctx context.Context, username, password string) (user.User, string, error)
	Login(ctx context.Context, username, password string) (user.User, string, error)
	VerifyToken(token string) (user.User, error)
}

type Server struct {
	cfg     config.Config
	engine  *gin.Engine
	httpSrv *http.Server
}

func New(cfg config.Config, svc authService) *Server {
	router := gin.Default()

	api := router.Group("/")
	api.POST("/register", registerHandler(svc))
	api.POST("/login", loginHandler(svc))
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	protected := api.Group("/")
	protected.Use(authMiddleware(svc))
	protected.GET("/me", meHandler())

	httpSrv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	return &Server{cfg: cfg, engine: router, httpSrv: httpSrv}
}

func (s *Server) Run() error {
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

type credentials struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

func registerHandler(svc authService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req credentials
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		u, token, err := svc.Register(ctx, req.Username, req.Password)
		if err != nil {
			if errors.Is(err, user.ErrDuplicateUser) {
				c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not register"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"user": u, "token": token})
	}
}

func loginHandler(svc authService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req credentials
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		u, token, err := svc.Login(ctx, req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": u, "token": token})
	}
}

func meHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing user context"})
			return
		}
		u, ok := val.(user.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": u})
	}
}

func authMiddleware(svc authService) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawAuth := c.GetHeader("Authorization")
		if rawAuth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(rawAuth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		u, err := svc.VerifyToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
