package pyservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-monitor-backend/config"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

type taskReq struct {
	TaskID int64 `json:"task_id"`
}

type pyResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func post(path string, body any) (*pyResp, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Post(config.PythonURL+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r pyResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("pyservice parse error: %w, body=%s", err, raw)
	}
	return &r, nil
}

// StartTask asks the Python service to start a task.
func StartTask(taskID int64) error {
	r, err := post("/api/task/start", taskReq{TaskID: taskID})
	if err != nil {
		return err
	}
	if r.Code != 0 {
		return fmt.Errorf("python service error: %s", r.Message)
	}
	return nil
}

// StopTask asks the Python service to stop a task.
func StopTask(taskID int64) error {
	r, err := post("/api/task/stop", taskReq{TaskID: taskID})
	if err != nil {
		return err
	}
	if r.Code != 0 {
		return fmt.Errorf("python service error: %s", r.Message)
	}
	return nil
}

// IsHealthy checks if the Python service is reachable.
func IsHealthy() bool {
	resp, err := httpClient.Get(config.PythonURL + "/api/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}
