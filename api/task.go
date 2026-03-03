package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/model"
	"ai-monitor-backend/pyservice"
	"ai-monitor-backend/store"
)

type TaskHandler struct {
	store *store.Store
}

func NewTaskHandler(s *store.Store) *TaskHandler {
	return &TaskHandler{store: s}
}

func (h *TaskHandler) ListAlgorithms(c *gin.Context) {
	algos, err := h.store.ListAlgorithms()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": algos})
}

func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := h.store.ListTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": tasks})
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req model.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	taskID, err := h.store.CreateTask(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	task, _ := h.store.GetTask(taskID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": task})
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}

	task, err := h.store.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "task not found"})
		return
	}
	// Stop running task before deleting
	if task.Status == 1 {
		_ = pyservice.StopTask(id)
	}

	if err := h.store.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}

func (h *TaskHandler) Start(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}

	task, err := h.store.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "task not found"})
		return
	}
	if task.Status == 1 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "already running"})
		return
	}

	if err := pyservice.StartTask(id); err != nil {
		_ = h.store.UpdateTaskStatus(id, 2, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "python service error: " + err.Error()})
		return
	}
	_ = h.store.UpdateTaskStatus(id, 1, "")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "task started", "data": gin.H{"task_id": id}})
}

func (h *TaskHandler) Stop(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}

	task, err := h.store.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "task not found"})
		return
	}
	if task.Status == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "already stopped"})
		return
	}

	if err := pyservice.StopTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "python service error: " + err.Error()})
		return
	}
	_ = h.store.UpdateTaskStatus(id, 0, "")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "task stopped", "data": gin.H{"task_id": id}})
}
