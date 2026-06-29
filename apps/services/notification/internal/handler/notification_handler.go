// Package handler provides HTTP handlers for the Notification Service.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/notification/internal/service"
)

// NotificationHandler handles notification endpoints.
type NotificationHandler struct {
	notifSvc *service.NotificationService
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(notifSvc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc}
}

// ListNotifications returns a paginated list of notifications.
// GET /api/v1/notifications
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	notifs, total, err := h.notifSvc.ListNotifications(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": notifs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// SendNotification sends a notification.
// POST /api/v1/notifications/send
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var input service.SendNotificationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	notif, err := h.notifSvc.Send(c.Request.Context(), &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": notif})
}

// MarkAsRead marks a notification as read.
// PATCH /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid notification ID")
		return
	}

	if err := h.notifSvc.MarkAsRead(c.Request.Context(), userID, notifID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "marked as read"}})
}

// MarkAllAsRead marks all notifications as read.
// POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	if err := h.notifSvc.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "all marked as read"}})
}

// UnreadCount returns the unread notification count.
// GET /api/v1/notifications/unread-count
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	count, err := h.notifSvc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": count}})
}

// GetPreferences returns notification preferences.
// GET /api/v1/notifications/preferences
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	prefs, err := h.notifSvc.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": prefs})
}

// UpdatePreferences updates notification preferences.
// PATCH /api/v1/notifications/preferences
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.UpdatePreferenceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	if err := h.notifSvc.UpdatePreference(c.Request.Context(), userID, &input); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "preference updated"}})
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
