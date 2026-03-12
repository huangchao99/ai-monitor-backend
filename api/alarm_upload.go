package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
	"ai-monitor-backend/uploader"
)

// AlarmUploadHandler 报警上传管理处理器
type AlarmUploadHandler struct {
	store    *store.Store
	uploader *uploader.Uploader
}

func NewAlarmUploadHandler(s *store.Store, u *uploader.Uploader) *AlarmUploadHandler {
	return &AlarmUploadHandler{store: s, uploader: u}
}

// GetSettings 获取报警上传配置
func (h *AlarmUploadHandler) GetSettings(c *gin.Context) {
	settings, err := h.store.GetAlarmUploadSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, settings)
}

// SaveSettings 保存报警上传配置
func (h *AlarmUploadHandler) SaveSettings(c *gin.Context) {
	var req model.UpdateAlarmUploadSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SaveAlarmUploadSettings(req); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// GetStats 获取上传统计（待上传/已成功/失败数量）
func (h *AlarmUploadHandler) GetStats(c *gin.Context) {
	stats, err := h.store.GetAlarmUploadStats()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, stats)
}

// ListQueue 获取上传队列列表（支持 status 筛选 + 分页）
func (h *AlarmUploadHandler) ListQueue(c *gin.Context) {
	statusFilter := -1 // -1 = 全部
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			statusFilter = v
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	items, total, err := h.store.ListAlarmUploadQueue(statusFilter, page, size)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.AlarmUploadQueueItem{}
	}
	ok(c, gin.H{"list": items, "total": total})
}

// RetryFailed 将所有失败记录重置为待上传，并立即触发一次上传
func (h *AlarmUploadHandler) RetryFailed(c *gin.Context) {
	if err := h.store.ResetFailedUploads(); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.uploader.RunNow()
	ok(c, nil)
}
