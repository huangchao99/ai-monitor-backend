package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
)

// PositionHandler handles positioning and navigation-state endpoints.
type PositionHandler struct {
	store *store.Store
}

func NewPositionHandler(s *store.Store) *PositionHandler {
	return &PositionHandler{store: s}
}

func (h *PositionHandler) GetSettings(c *gin.Context) {
	settings, err := h.store.GetPositionSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, settings)
}

func (h *PositionHandler) SaveSettings(c *gin.Context) {
	var req model.UpdatePositionSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SavePositionSettings(req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, nil)
}

func (h *PositionHandler) GetStatus(c *gin.Context) {
	status, err := h.store.GetPositionStatus()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, status)
}
