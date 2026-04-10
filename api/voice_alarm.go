package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/config"
	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
)

// VoiceAlarmHandler handles all voice-alarm management endpoints.
type VoiceAlarmHandler struct {
	store *store.Store
}

func NewVoiceAlarmHandler(s *store.Store) *VoiceAlarmHandler {
	return &VoiceAlarmHandler{store: s}
}

// ─── Settings ──────────────────────────────────────────────────

// GetSettings returns the global voice alarm switch + device info.
func (h *VoiceAlarmHandler) GetSettings(c *gin.Context) {
	settings, err := h.store.GetVoiceAlarmSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, settings)
}

// SaveSettings updates the global switch and device info, and syncs config.properties.
func (h *VoiceAlarmHandler) SaveSettings(c *gin.Context) {
	var req model.UpdateVoiceAlarmSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SaveVoiceAlarmSettings(req); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	// Sync config.properties
	if err := writeConfigProperties(req.DeviceIP, req.DeviceUser, req.DevicePass); err != nil {
		// Non-fatal: log the error but still return success
		_ = err
	}
	ok(c, nil)
}

// writeConfigProperties overwrites the runtime audio config file.
func writeConfigProperties(ip, user, pass string) error {
	content := fmt.Sprintf("ip=%s\nname=%s\npass=%s\n", ip, user, pass)
	if err := os.MkdirAll(filepath.Dir(config.AudioConfigPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(config.AudioConfigPath, []byte(content), 0644)
}

// ─── Algo Map ──────────────────────────────────────────────────

// ListAlgoMap returns all algorithms with their optional audio file mapping.
func (h *VoiceAlarmHandler) ListAlgoMap(c *gin.Context) {
	maps, err := h.store.ListVoiceAlarmAlgoMaps()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if maps == nil {
		maps = []model.VoiceAlarmAlgoMap{}
	}
	ok(c, maps)
}

// SetAlgoMap sets or replaces the audio file for a given algo_id.
func (h *VoiceAlarmHandler) SetAlgoMap(c *gin.Context) {
	algoID, err := strconv.ParseInt(c.Param("algo_id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid algo_id")
		return
	}
	var req model.SetVoiceAlarmAlgoMapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetVoiceAlarmAlgoMap(algoID, req.AudioFile); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// DeleteAlgoMap removes the audio file mapping for a given algo_id.
func (h *VoiceAlarmHandler) DeleteAlgoMap(c *gin.Context) {
	algoID, err := strconv.ParseInt(c.Param("algo_id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid algo_id")
		return
	}
	if err := h.store.DeleteVoiceAlarmAlgoMap(algoID); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// ─── Audio Files ───────────────────────────────────────────────

// ListAudioFiles lists all .pcm files in the AudioFile directory.
func (h *VoiceAlarmHandler) ListAudioFiles(c *gin.Context) {
	entries, err := os.ReadDir(config.AudioFilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			ok(c, []string{})
			return
		}
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	var files []gin.H
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pcm") {
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, gin.H{
			"name":     strings.TrimSuffix(e.Name(), ".pcm"),
			"filename": e.Name(),
			"size":     size,
		})
	}
	if files == nil {
		files = []gin.H{}
	}
	ok(c, files)
}

// UploadAudioFile saves an uploaded .pcm file to the AudioFile directory.
func (h *VoiceAlarmHandler) UploadAudioFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "missing file field")
		return
	}
	if !strings.HasSuffix(file.Filename, ".pcm") {
		fail(c, http.StatusBadRequest, "只允许上传 .pcm 文件")
		return
	}
	// Sanitise filename: no path traversal
	filename := filepath.Base(file.Filename)
	dest := filepath.Join(config.AudioFilesDir, filename)

	src, err := file.Open()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer src.Close()

	if err := os.MkdirAll(config.AudioFilesDir, 0755); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	out, err := os.Create(dest)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"filename": filename,
		"name":     strings.TrimSuffix(filename, ".pcm"),
		"size":     file.Size,
	})
}

// DeleteAudioFile removes a .pcm file from the AudioFile directory.
func (h *VoiceAlarmHandler) DeleteAudioFile(c *gin.Context) {
	name := c.Param("name")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		fail(c, http.StatusBadRequest, "invalid file name")
		return
	}
	filename := name
	if !strings.HasSuffix(filename, ".pcm") {
		filename += ".pcm"
	}
	dest := filepath.Join(config.AudioFilesDir, filename)
	if err := os.Remove(dest); err != nil {
		if os.IsNotExist(err) {
			fail(c, http.StatusNotFound, "文件不存在")
		} else {
			fail(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	ok(c, nil)
}
