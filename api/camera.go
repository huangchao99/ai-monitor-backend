package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
	"ai-monitor-backend/zlm"
)

var snapClient = &http.Client{Timeout: 12 * time.Second}

type CameraHandler struct {
	store *store.Store
}

func NewCameraHandler(s *store.Store) *CameraHandler {
	return &CameraHandler{store: s}
}

func (h *CameraHandler) List(c *gin.Context) {
	cameras, err := h.store.ListCameras()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	// Enrich with derived URLs
	for i := range cameras {
		if cameras[i].ZlmStream != nil {
			cameras[i].ZlmStream.FlvURL = zlm.BuildFlvURL(cameras[i].ZlmStream.StreamKey)
			cameras[i].ZlmStream.HlsURL = zlm.BuildHlsURL(cameras[i].ZlmStream.StreamKey)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cameras})
}

func (h *CameraHandler) Create(c *gin.Context) {
	var req model.CreateCameraReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	cameraID, err := h.store.CreateCamera(req.Name, req.RtspURL, req.Location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	// Auto-start ZLM proxy stream
	streamKey := fmt.Sprintf("cam%d", cameraID)
	proxyKey, zlmErr := zlm.AddStreamProxy(streamKey, req.RtspURL)
	status := 1
	if zlmErr != nil {
		status = 2
		proxyKey = ""
	}
	_ = h.store.UpsertZlmStream(cameraID, streamKey, proxyKey, status)

	camera, _ := h.store.GetCamera(cameraID)
	if camera != nil && camera.ZlmStream != nil {
		camera.ZlmStream.FlvURL = zlm.BuildFlvURL(streamKey)
		camera.ZlmStream.HlsURL = zlm.BuildHlsURL(streamKey)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": camera})
}

func (h *CameraHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}
	var req model.UpdateCameraReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	existing, err := h.store.GetCamera(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "camera not found"})
		return
	}

	name := req.Name
	if name == "" {
		name = existing.Name
	}
	rtspURL := req.RtspURL
	if rtspURL == "" {
		rtspURL = existing.RtspURL
	}
	location := req.Location
	if location == "" {
		location = existing.Location
	}

	if err := h.store.UpdateCamera(id, name, rtspURL, location, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	// If RTSP URL changed, restart ZLM proxy
	if req.RtspURL != "" && req.RtspURL != existing.RtspURL {
		if existing.ZlmStream != nil && existing.ZlmStream.ProxyKey != "" {
			_ = zlm.DelStreamProxy(existing.ZlmStream.ProxyKey)
		}
		streamKey := fmt.Sprintf("cam%d", id)
		proxyKey, zlmErr := zlm.AddStreamProxy(streamKey, rtspURL)
		status := 1
		if zlmErr != nil {
			status = 2
			proxyKey = ""
		}
		_ = h.store.UpsertZlmStream(id, streamKey, proxyKey, status)
	}

	camera, _ := h.store.GetCamera(id)
	if camera != nil && camera.ZlmStream != nil {
		camera.ZlmStream.FlvURL = zlm.BuildFlvURL(camera.ZlmStream.StreamKey)
		camera.ZlmStream.HlsURL = zlm.BuildHlsURL(camera.ZlmStream.StreamKey)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": camera})
}

func (h *CameraHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid id"})
		return
	}

	camera, err := h.store.GetCamera(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "camera not found"})
		return
	}
	// Stop ZLM proxy
	if camera.ZlmStream != nil && camera.ZlmStream.ProxyKey != "" {
		_ = zlm.DelStreamProxy(camera.ZlmStream.ProxyKey)
	}

	if err := h.store.DeleteCamera(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}

func (h *CameraHandler) StreamStart(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	camera, err := h.store.GetCamera(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "camera not found"})
		return
	}
	// Stop existing proxy if any
	if camera.ZlmStream != nil && camera.ZlmStream.ProxyKey != "" {
		_ = zlm.DelStreamProxy(camera.ZlmStream.ProxyKey)
	}

	streamKey := fmt.Sprintf("cam%d", id)
	proxyKey, zlmErr := zlm.AddStreamProxy(streamKey, camera.RtspURL)
	status := 1
	if zlmErr != nil {
		status = 2
		proxyKey = ""
		_ = h.store.UpsertZlmStream(id, streamKey, proxyKey, status)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "ZLM addStreamProxy failed: " + zlmErr.Error()})
		return
	}
	_ = h.store.UpsertZlmStream(id, streamKey, proxyKey, status)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "stream started",
		"data": gin.H{
			"stream_key": streamKey,
			"proxy_key":  proxyKey,
			"flv_url":    zlm.BuildFlvURL(streamKey),
			"hls_url":    zlm.BuildHlsURL(streamKey),
		},
	})
}

func (h *CameraHandler) StreamStop(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	camera, err := h.store.GetCamera(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "camera not found"})
		return
	}
	if camera.ZlmStream == nil || camera.ZlmStream.ProxyKey == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "stream not running"})
		return
	}
	if err := zlm.DelStreamProxy(camera.ZlmStream.ProxyKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	_ = h.store.UpdateZlmStreamStatus(id, 0)
	_ = h.store.UpsertZlmStream(id, camera.ZlmStream.StreamKey, "", 0)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "stream stopped"})
}

const inferServerURL = "http://127.0.0.1:8080"

// Snapshot returns a JPEG frame for the camera.
//
// Strategy:
//  1. If the camera has a running task, use its cached frame immediately.
//  2. Otherwise, register a temporary decode-only stream (no models) in the
//     infer server, poll until a frame is cached, then clean up.
func (h *CameraHandler) Snapshot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// ── 1. Try running tasks first (fast path) ──────────────────────────────
	taskIDs, _ := h.store.GetRunningTaskIDsByCamera(id)
	for _, tid := range taskIDs {
		if jpegBytes := fetchInferFrame(fmt.Sprintf("task_%d", tid)); jpegBytes != nil {
			c.Header("Content-Type", "image/jpeg")
			c.Header("Cache-Control", "no-cache, no-store")
			c.Writer.Write(jpegBytes)
			return
		}
	}

	// ── 2. No running task: register a temporary decode-only stream ─────────
	camera, err := h.store.GetCamera(id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	tempID := fmt.Sprintf("snap_cam%d", id)

	// Register stream (models omitted → decode only, no inference)
	regBody := fmt.Sprintf(`{"cam_id":%q,"rtsp_url":%q,"frame_skip":1}`, tempID, camera.RtspURL)
	regResp, err := snapClient.Post(
		inferServerURL+"/api/streams",
		"application/json",
		strings.NewReader(regBody),
	)
	if err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	regResp.Body.Close()

	// Ensure cleanup regardless of outcome
	defer func() {
		req, _ := http.NewRequest(http.MethodDelete, inferServerURL+"/api/streams/"+tempID, nil)
		if req != nil {
			resp, _ := snapClient.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
		}
	}()

	// Poll for the first cached frame (up to 6 s)
	for i := 0; i < 12; i++ {
		time.Sleep(500 * time.Millisecond)
		if jpegBytes := fetchInferFrame(tempID); jpegBytes != nil {
			c.Header("Content-Type", "image/jpeg")
			c.Header("Cache-Control", "no-cache, no-store")
			c.Writer.Write(jpegBytes)
			return
		}
	}

	c.Status(http.StatusServiceUnavailable)
}

// fetchInferFrame fetches a cached JPEG frame from the infer server for the
// given stream ID. Returns nil when no frame is available yet.
func fetchInferFrame(streamID string) []byte {
	url := inferServerURL + "/api/cache/image?stream_id=" + streamID + "&latest=true"
	resp, err := snapClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}
