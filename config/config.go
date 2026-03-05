package config

import "os"

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	DBPath     = getEnv("DB_PATH", "/home/hzhy/aimonitor.db")
	Port       = getEnv("PORT", ":8090")
	ZLMBaseURL = getEnv("ZLM_BASE_URL", "http://127.0.0.1:80")
	ZLMSecret  = getEnv("ZLM_SECRET", "vEq3Z2BobQevk5dRs1zZ6DahIt5U9urT")
	PythonURL  = getEnv("PYTHON_URL", "http://127.0.0.1:9500")
	// Directory where uploaded model/label files are stored
	ModelsUploadDir = getEnv("MODELS_UPLOAD_DIR", "/home/hzhy/models")
	// ZLM stream app name
	ZLMApp = "live"
)
