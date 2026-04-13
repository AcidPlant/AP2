package http

import (
	"errors"
	"net/http"

	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc usecase.PaymentUseCase
}

func NewHandler(uc usecase.PaymentUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/payments", h.Authorize)
	r.GET("/payments/:order_id", h.GetByOrderID)
}

type authorizeRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount"   binding:"required"`
}

func (h *Handler) Authorize(c *gin.Context) {
	var req authorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.uc.Authorize(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *Handler) GetByOrderID(c *gin.Context) {
	payment, err := h.uc.GetByOrderID(c.Request.Context(), c.Param("order_id"))
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, payment)
}
