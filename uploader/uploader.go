package uploader

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ai-monitor-backend/model"
	"ai-monitor-backend/store"
)

const snapshotDir = "/home/hzhy/ai-monitor-service/snapshots"

// Uploader 后台报警上传 Worker
type Uploader struct {
	store *store.Store
}

func New(s *store.Store) *Uploader {
	return &Uploader{store: s}
}

// Start 启动后台定时 Worker，每 60 秒执行一次
func (u *Uploader) Start() {
	go func() {
		u.runOnce() // 启动时立即执行一次
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			u.runOnce()
		}
	}()
}

// RunNow 立即触发一次上传（用于手动重试）
func (u *Uploader) RunNow() {
	go u.runOnce()
}

func (u *Uploader) runOnce() {
	settings, err := u.store.GetAlarmUploadSettings()
	if err != nil {
		log.Printf("[uploader] GetSettings error: %v", err)
		return
	}
	if !settings.Enabled || settings.UploadURL == "" {
		return
	}

	// 将所有新产生的 alarm 自动入队
	if err := u.store.EnqueueNewAlarms(); err != nil {
		log.Printf("[uploader] EnqueueNewAlarms error: %v", err)
	}

	items, err := u.store.GetPendingUploads()
	if err != nil {
		log.Printf("[uploader] GetPendingUploads error: %v", err)
		return
	}

	for _, item := range items {
		if err := u.uploadOne(settings, item); err != nil {
			log.Printf("[uploader] alarm %d failed: %v", item.AlarmID, err)
			u.store.MarkUploadFailed(item.QueueID, err.Error())
		} else {
			u.store.MarkUploadSuccess(item.QueueID)
			log.Printf("[uploader] alarm %d uploaded OK", item.AlarmID)
		}
	}
}

func (u *Uploader) uploadOne(s model.AlarmUploadSettings, item model.PendingUploadItem) error {
	imageBase64 := ""
	if item.ImageURL != "" {
		// image_url 格式为 "/snapshots/xxx.jpg"，实际文件在 snapshotDir 下
		imgPath := filepath.Join(snapshotDir, filepath.Base(item.ImageURL))
		if data, err := os.ReadFile(imgPath); err == nil {
			imageBase64 = base64.StdEncoding.EncodeToString(data)
		}
		// 图片读取失败不中断，继续上传其他字段
	}

	payload := map[string]string{
		"DeviceID":        s.DeviceID,
		"TaskCode":        item.TaskName,
		"CameraNo":        item.CameraName,
		"ResultID":        fmt.Sprintf("%d", item.AlarmID),
		"RecogType":       item.AlgoName,
		"ResultTime":      item.AlarmTime,
		"ResultContent":   extractContent(item.AlarmDetails),
		"ResultImageData": imageBase64,
	}

	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(s.UploadURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}
	return nil
}

// extractContent 尝试从 alarm_details JSON 中提取 description 字段，失败则原样返回
func extractContent(alarmDetails string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(alarmDetails), &m); err == nil {
		if desc, ok := m["description"].(string); ok && desc != "" {
			return desc
		}
	}
	return alarmDetails
}
