package http

import (
	"errors"
	"net/http"

	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc usecase.OrderUseCase
}

func NewHandler(uc usecase.OrderUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrder)
	r.PATCH("/orders/:id/cancel", h.CancelOrder)
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	order, err := h.uc.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrPaymentServiceUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service unavailable"})
		case errors.Is(err, usecase.ErrInvalidAmount):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) GetOrder(c *gin.Context) {
	order, err := h.uc.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *Handler) CancelOrder(c *gin.Context) {
	err := h.uc.CancelOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, usecase.ErrCannotCancel):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}
