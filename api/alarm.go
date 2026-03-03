package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
)

type AlarmHandler struct {
	store *store.Store
}

func NewAlarmHandler(s *store.Store) *AlarmHandler {
	return &AlarmHandler{store: s}
}

func (h *AlarmHandler) List(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Query("task_id"), 10, 64)
	// status=-1 means no filter
	status := -1
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	alarms, total, err := h.store.ListAlarms(taskID, status, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	if alarms == nil {
		alarms = []model.Alarm{}
	}
	for i := range alarms {
		if url := alarms[i].ImageURL; url != "" && strings.HasPrefix(url, "/") {
			alarms[i].ImageURL = "/snapshots/" + filepath.Base(url)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":  alarms,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

func (h *AlarmHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}
	var req model.UpdateAlarmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}
	if err := h.store.UpdateAlarmStatus(id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "updated"})
}
