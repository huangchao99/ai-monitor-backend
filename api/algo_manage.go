package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-monitor-backend/config"
	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
)

// AlgoManageHandler handles algorithm, model and plugin management.
type AlgoManageHandler struct {
	store *store.Store
}

func NewAlgoManageHandler(s *store.Store) *AlgoManageHandler {
	return &AlgoManageHandler{store: s}
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": -1, "message": msg, "data": nil})
}

// ─── Algorithms ──────────────────────────────────────────────

// ListAlgorithms returns all algorithms with their associated models.
func (h *AlgoManageHandler) ListAlgorithms(c *gin.Context) {
	algos, err := h.store.ListAlgorithmsWithModels()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, algos)
}

func (h *AlgoManageHandler) CreateAlgorithm(c *gin.Context) {
	var req model.CreateAlgorithmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.store.CreateAlgorithm(req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

func (h *AlgoManageHandler) UpdateAlgorithm(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req model.UpdateAlgorithmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.UpdateAlgorithm(id, req); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

func (h *AlgoManageHandler) DeleteAlgorithm(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteAlgorithm(id); err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	ok(c, nil)
}

// ─── Models ───────────────────────────────────────────────────

func (h *AlgoManageHandler) ListModels(c *gin.Context) {
	models, err := h.store.ListModels()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, models)
}

func (h *AlgoManageHandler) CreateModel(c *gin.Context) {
	var req model.CreateModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.store.CreateModel(req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

func (h *AlgoManageHandler) UpdateModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req model.UpdateModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.UpdateModel(id, req); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

func (h *AlgoManageHandler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteModel(id); err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	ok(c, nil)
}

// ─── Plugin Proxy ─────────────────────────────────────────────

// proxyToPython forwards plugin management requests to the Python service.
func proxyToPython(c *gin.Context, method, path string, body io.Reader, contentType string) {
	targetURL := fmt.Sprintf("%s%s", config.PythonURL, path)
	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		fail(c, http.StatusInternalServerError, "proxy build request failed: "+err.Error())
		return
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	// Copy query string
	req.URL.RawQuery = c.Request.URL.RawQuery

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(c, http.StatusBadGateway, "Python service unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fail(c, http.StatusInternalServerError, "read python response failed")
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func (h *AlgoManageHandler) ListPlugins(c *gin.Context) {
	proxyToPython(c, http.MethodGet, "/api/plugins", nil, "")
}

func (h *AlgoManageHandler) UploadPlugin(c *gin.Context) {
	// Read multipart body and forward as-is
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fail(c, http.StatusBadRequest, "read body failed")
		return
	}
	proxyToPython(c, http.MethodPost, "/api/plugins",
		bytes.NewReader(body), c.Request.Header.Get("Content-Type"))
}

func (h *AlgoManageHandler) DeletePlugin(c *gin.Context) {
	filename := c.Param("filename")
	proxyToPython(c, http.MethodDelete, "/api/plugins/"+filename, nil, "")
}
