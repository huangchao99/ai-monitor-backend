package config

import (
	"os"
	"path/filepath"
	"strings"
)

func getEnv(keys []string, fallback string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return fallback
}

func normalizeListenAddr(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if strings.HasPrefix(value, ":") {
		return value
	}
	if strings.Contains(value, ":") {
		return value
	}
	return ":" + value
}

var (
	DBPath = getEnv([]string{
		"AI_MONITOR_DB_PATH",
		"DB_PATH",
		"AIMONITOR_DB",
	}, "/var/lib/ai-monitor/aimonitor.db")
	Port = normalizeListenAddr(getEnv([]string{
		"AI_MONITOR_BACKEND_PORT",
		"PORT",
	}, ":8090"), ":8090")
	ZLMBaseURL = getEnv([]string{
		"AI_MONITOR_ZLM_BASE_URL",
		"ZLM_BASE_URL",
	}, "http://127.0.0.1:80")
	ZLMSecret = getEnv([]string{
		"AI_MONITOR_ZLM_SECRET",
		"ZLM_SECRET",
	}, "vEq3Z2BobQevk5dRs1zZ6DahIt5U9urT")
	PythonURL = getEnv([]string{
		"AI_MONITOR_PYTHON_URL",
		"PYTHON_URL",
	}, "http://127.0.0.1:9500")
	InferURL = getEnv([]string{
		"AI_MONITOR_INFER_URL",
		"INFER_SERVER_URL",
	}, "http://127.0.0.1:8080")
	ModelsUploadDir = getEnv([]string{
		"AI_MONITOR_MODELS_DIR",
		"MODELS_UPLOAD_DIR",
	}, "/opt/ai-monitor/current/models")
	SnapshotDir = getEnv([]string{
		"AI_MONITOR_SNAPSHOT_DIR",
		"SNAPSHOT_DIR",
	}, "/var/lib/ai-monitor/snapshots")
	AudioDir = getEnv([]string{
		"AI_MONITOR_AUDIO_DIR",
		"AUDIO_DIR",
	}, "/opt/ai-monitor/current/audio")
	AudioFilesDir = getEnv([]string{
		"AI_MONITOR_AUDIO_FILES_DIR",
	}, filepath.Join(AudioDir, "AudioFile"))
	AudioConfigPath = getEnv([]string{
		"AI_MONITOR_AUDIO_CONFIG_PATH",
	}, filepath.Join(AudioDir, "config.properties"))
	ReleaseRoot = getEnv([]string{
		"AI_MONITOR_RELEASE_ROOT",
	}, "/opt/ai-monitor/current")
	// ZLM stream app name
	ZLMApp = getEnv([]string{
		"AI_MONITOR_ZLM_APP",
	}, "live")
)
