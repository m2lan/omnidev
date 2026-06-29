// Package handler provides HTTP handlers for the Billing Service.
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/billing/internal/service"
)

// BillingHandler handles billing endpoints.
type BillingHandler struct {
	billingSvc *service.BillingService
}

// NewBillingHandler creates a new billing handler.
func NewBillingHandler(billingSvc *service.BillingService) *BillingHandler {
	return &BillingHandler{billingSvc: billingSvc}
}

// GetAccount returns the billing account for the current user.
// GET /api/v1/billing/account
func (h *BillingHandler) GetAccount(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	account, err := h.billingSvc.GetAccount(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// GetUsage returns usage summary.
// GET /api/v1/billing/usage
func (h *BillingHandler) GetUsage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}

	summary, err := h.billingSvc.GetUsage(c.Request.Context(), userID, days)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}

// RecordUsage records a usage event.
// POST /api/v1/billing/usage/record
func (h *BillingHandler) RecordUsage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.RecordUsageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	record, err := h.billingSvc.RecordUsage(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": record})
}

// ListInvoices returns a paginated list of invoices.
// GET /api/v1/billing/invoices
func (h *BillingHandler) ListInvoices(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	invoices, total, err := h.billingSvc.ListInvoices(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": invoices,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// GetInvoice returns an invoice by ID.
// GET /api/v1/billing/invoices/:id
func (h *BillingHandler) GetInvoice(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid invoice ID")
		return
	}

	invoice, err := h.billingSvc.GetInvoice(c.Request.Context(), userID, invoiceID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoice})
}

// Subscribe subscribes to a plan.
// POST /api/v1/billing/subscribe
func (h *BillingHandler) Subscribe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.SubscribeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.billingSvc.Subscribe(c.Request.Context(), userID, &input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "subscribed to " + input.Plan}})
}

// AddPaymentMethod adds a payment method.
// POST /api/v1/billing/payment-method
func (h *BillingHandler) AddPaymentMethod(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.AddPaymentMethodInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.billingSvc.AddPaymentMethod(c.Request.Context(), userID, &input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "payment method added"}})
}

// --- Helpers ---

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
}

func badRequest(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": detail}})
}

func handleError(c *gin.Context, err error) {
	if e, ok := err.(*appErr.AppError); ok {
		c.JSON(e.Code, gin.H{"error": gin.H{"code": e.Code, "message": e.Message, "detail": e.Detail}})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": "internal server error"}})
}

// unused helper to avoid compile error
var _ = time.Now
