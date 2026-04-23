package handlers

import (
	"net/http"
	"strconv"
	"time"

	"realestate-ai/backend/internal/models"
	"realestate-ai/backend/internal/pkg/ai"
	"realestate-ai/backend/internal/repository"
	"realestate-ai/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	LeadRepo    *repository.LeadRepository
	PropRepo    *repository.PropertyRepository
	ChatRepo    *repository.ChatRepository
	TelegramSvc *service.TelegramService
	AIRouter    *ai.Router
}

func NewHandler(leadRepo *repository.LeadRepository, propRepo *repository.PropertyRepository, chatRepo *repository.ChatRepository, telegramSvc *service.TelegramService, aiRouter *ai.Router) *Handler {
	return &Handler{
		LeadRepo:    leadRepo,
		PropRepo:    propRepo,
		ChatRepo:    chatRepo,
		TelegramSvc: telegramSvc,
		AIRouter:    aiRouter,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	api := r.Group("/api")
	{
		api.GET("/leads", h.ListLeads)
		api.GET("/leads/:id", h.GetLead)
		api.GET("/leads/:id/messages", h.GetLeadMessages)
		api.GET("/stats", h.GetStats)
		api.POST("/test-ai", h.TestAI)

		api.GET("/properties", h.ListProperties)
		api.GET("/properties/:id", h.GetProperty)
		api.POST("/properties", h.CreateProperty)
		api.PUT("/properties/:id", h.UpdateProperty)
		api.DELETE("/properties/:id", h.DeleteProperty)

		api.POST("/webhook", h.Webhook)
	}
}

func (h *Handler) ListLeads(c *gin.Context) {
	leads, err := h.LeadRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, leads)
}

func (h *Handler) GetLead(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	lead, err := h.LeadRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lead not found"})
		return
	}
	c.JSON(http.StatusOK, lead)
}

func (h *Handler) GetLeadMessages(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	messages, err := h.ChatRepo.GetByLeadID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, messages)
}

func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.LeadRepo.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) ListProperties(c *gin.Context) {
	properties, err := h.PropRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, properties)
}

func (h *Handler) GetProperty(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	property, err := h.PropRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *Handler) CreateProperty(c *gin.Context) {
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.PropRepo.Create(&property); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, property)
}

func (h *Handler) UpdateProperty(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var property models.Property
	if err := c.ShouldBindJSON(&property); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	property.ID = uint(id)
	if err := h.PropRepo.Update(&property); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, property)
}

func (h *Handler) DeleteProperty(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.PropRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) TestAI(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start := time.Now()
	resp, err := h.AIRouter.Chat(c.Request.Context(), []ai.Message{{Role: "user", Content: req.Message}})
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "duration": duration.String()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": resp, "duration": duration.String()})
}

func (h *Handler) Webhook(c *gin.Context) {
	var payload struct {
		Message struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
			From struct {
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
			} `json:"from"`
		} `json:"message"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if payload.Message.Text == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	name := payload.Message.From.FirstName
	if name == "" {
		name = payload.Message.From.Username
	}

	reply, err := h.TelegramSvc.HandleMessage(c.Request.Context(), payload.Message.Chat.ID, payload.Message.Text, name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "reply": reply})
}
