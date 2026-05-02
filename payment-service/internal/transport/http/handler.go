package http

import (
	"net/http"
	"strconv"

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
	r.POST("/payments", h.CreatePayment)
	r.GET("/payments/order/:orderID", h.GetByOrder)
	r.GET("/payments", h.ListPayments)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req struct {
		OrderID       string `json:"order_id" binding:"required"`
		Amount        int64  `json:"amount"   binding:"required,gt=0"`
		CustomerEmail string `json:"customer_email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CustomerEmail == "" {
		req.CustomerEmail = "user@example.com"
	}

	p, err := h.uc.Authorize(c.Request.Context(), req.OrderID, req.Amount, req.CustomerEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) GetByOrder(c *gin.Context) {
	p, err := h.uc.GetByOrderID(c.Request.Context(), c.Param("orderID"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ListPayments(c *gin.Context) {
	Min, _ := strconv.ParseInt(c.Query("min_amount"), 10, 64)
	Max, _ := strconv.ParseInt(c.Query("max_amount"), 10, 64)
	payments, err := h.uc.ListPayments(c.Request.Context(), Min, Max)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, payments)
}
