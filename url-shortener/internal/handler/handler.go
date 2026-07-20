package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AlexTihonow/url-shortener/internal/repository"
	"github.com/AlexTihonow/url-shortener/internal/service"
)

type Handler struct {
	svc     *service.Service
	baseURL string
}

func New(svc *service.Service, baseURL string) *Handler {
	return &Handler{svc: svc, baseURL: baseURL}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.health)
	r.GET("/:code", h.redirect)

	api := r.Group("/api/v1")
	{
		api.POST("/links", h.create)
		api.GET("/links/:code/stats", h.stats)
		api.DELETE("/links/:code", h.delete)
	}
}

type createRequest struct {
	URL        string     `json:"url" binding:"required"`
	CustomCode string     `json:"custom_code"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	link, err := h.svc.Create(c.Request.Context(), service.CreateInput{
		OriginalURL: req.URL,
		CustomCode:  req.CustomCode,
		ExpiresAt:   req.ExpiresAt,
	})
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	case errors.Is(err, repository.ErrCodeTaken):
		c.JSON(http.StatusConflict, gin.H{"error": "custom code already taken"})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"short_code": link.ShortCode,
		"short_url":  h.baseURL + "/" + link.ShortCode,
		"original":   link.OriginalURL,
		"expires_at": link.ExpiresAt,
	})
}

func (h *Handler) redirect(c *gin.Context) {
	code := c.Param("code")
	target, err := h.svc.Resolve(
		c.Request.Context(), code,
		c.Request.UserAgent(), c.Request.Referer(),
	)
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Redirect(http.StatusFound, target)
}

func (h *Handler) stats(c *gin.Context) {
	s, err := h.svc.Stats(c.Request.Context(), c.Param("code"))
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) delete(c *gin.Context) {
	err := h.svc.Delete(c.Request.Context(), c.Param("code"))
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
