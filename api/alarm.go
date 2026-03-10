package api

import (
	"log"
	"net/http"
	"os"
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
	algoName := c.Query("algo_name")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	// status=-1 means no filter
	status := -1
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	alarms, total, err := h.store.ListAlarms(taskID, algoName, startDate, endDate, status, page, size)
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

func (h *AlarmHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "ids 不能为空"})
		return
	}

	imageURLs, err := h.store.BatchDeleteAlarms(req.IDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	// Delete snapshot files
	for _, url := range imageURLs {
		var absPath string
		if strings.HasPrefix(url, "/") {
			absPath = url
		} else {
			absPath = filepath.Join("/home/hzhy/ai-monitor-service/snapshots", filepath.Base(url))
		}
		if removeErr := os.Remove(absPath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("warn: failed to delete snapshot file %s: %v", absPath, removeErr)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted", "data": gin.H{"count": len(req.IDs)}})
}

func (h *AlarmHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}
	imageURL, err := h.store.DeleteAlarm(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	// Delete snapshot file if it exists (imageURL is the raw file path stored in DB)
	if imageURL != "" && strings.HasPrefix(imageURL, "/") {
		if removeErr := os.Remove(imageURL); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("warn: failed to delete snapshot file %s: %v", imageURL, removeErr)
		}
	} else if imageURL != "" {
		// Relative path fallback
		absPath := filepath.Join("/home/hzhy/ai-monitor-service/snapshots", filepath.Base(imageURL))
		if removeErr := os.Remove(absPath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("warn: failed to delete snapshot file %s: %v", absPath, removeErr)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}
