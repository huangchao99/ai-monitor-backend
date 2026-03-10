package model

import "time"

// Camera maps to the cameras table.
type Camera struct {
	ID       int64  `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	RtspURL  string `json:"rtsp_url" db:"rtsp_url"`
	Location string `json:"location" db:"location"`
	Status   int    `json:"status" db:"status"`
	// Joined from zlm_streams
	ZlmStream *ZlmStream `json:"zlm_stream,omitempty"`
}

// ZlmStream tracks ZLMediaKit proxy stream state.
type ZlmStream struct {
	ID        int64     `json:"id" db:"id"`
	CameraID  int64     `json:"camera_id" db:"camera_id"`
	App       string    `json:"app" db:"app"`
	StreamKey string    `json:"stream_key" db:"stream_key"`
	ProxyKey  string    `json:"proxy_key" db:"proxy_key"`
	Status    int       `json:"status" db:"status"` // 0:inactive 1:active 2:error
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// Derived URLs (not in DB)
	FlvURL string `json:"flv_url,omitempty"`
	HlsURL string `json:"hls_url,omitempty"`
}

// Algorithm maps to the algorithms table.
type Algorithm struct {
	ID              int64  `json:"id" db:"id"`
	AlgoKey         string `json:"algo_key" db:"algo_key"`
	AlgoName        string `json:"algo_name" db:"algo_name"`
	Category        string `json:"category" db:"category"`
	ParamDefinition string `json:"param_definition" db:"param_definition"`
	// Joined
	Models []Model `json:"models,omitempty"`
}

// Model maps to the models table.
type Model struct {
	ID             int64   `json:"id" db:"id"`
	ModelName      string  `json:"model_name" db:"model_name"`
	ModelPath      string  `json:"model_path" db:"model_path"`
	LabelsPath     string  `json:"labels_path" db:"labels_path"`
	ModelType      string  `json:"model_type" db:"model_type"`
	InputWidth     int     `json:"input_width" db:"input_width"`
	InputHeight    int     `json:"input_height" db:"input_height"`
	ConfThreshold  float64 `json:"conf_threshold" db:"conf_threshold"`
	NmsThreshold   float64 `json:"nms_threshold" db:"nms_threshold"`
}

// AlgoModelMap maps to the algo_model_map table.
type AlgoModelMap struct {
	ID      int64 `json:"id" db:"id"`
	AlgoID  int64 `json:"algo_id" db:"algo_id"`
	ModelID int64 `json:"model_id" db:"model_id"`
}

// Task maps to the tasks table.
type Task struct {
	ID            int64     `json:"id" db:"id"`
	TaskName      string    `json:"task_name" db:"task_name"`
	CameraID      int64     `json:"camera_id" db:"camera_id"`
	AlarmDeviceID string    `json:"alarm_device_id" db:"alarm_device_id"`
	Status        int       `json:"status" db:"status"` // 0:stopped 1:running 2:error
	ErrorMsg      string    `json:"error_msg" db:"error_msg"`
	Remark        string    `json:"remark" db:"remark"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	// Joined
	CameraName string        `json:"camera_name,omitempty"`
	AlgoDetails []AlgoDetail  `json:"algo_details,omitempty"`
	AlarmCount  int           `json:"alarm_count,omitempty"`
}

// AlgoDetail maps to task_algo_details.
type AlgoDetail struct {
	ID          int64  `json:"id" db:"id"`
	TaskID      int64  `json:"task_id" db:"task_id"`
	AlgoID      int64  `json:"algo_id" db:"algo_id"`
	RoiConfig   string `json:"roi_config" db:"roi_config"`
	AlgoParams  string `json:"algo_params" db:"algo_params"`
	AlarmConfig string `json:"alarm_config" db:"alarm_config"`
	// Joined
	AlgoKey  string `json:"algo_key,omitempty"`
	AlgoName string `json:"algo_name,omitempty"`
}

// Alarm maps to the alarms table.
type Alarm struct {
	ID           int64  `json:"id" db:"id"`
	TaskID       int64  `json:"task_id" db:"task_id"`
	AlgoName     string `json:"algo_name" db:"algo_name"`
	AlarmTime    string `json:"alarm_time" db:"alarm_time"`
	AlarmLocation string `json:"alarm_location" db:"alarm_location"`
	ImageURL     string    `json:"image_url" db:"image_url"`
	Status       int       `json:"status" db:"status"` // 0:unhandled 1:handled
	AlarmDetails string    `json:"alarm_details" db:"alarm_details"`
	// Joined
	TaskName   string `json:"task_name"`
	CameraName string `json:"camera_name"`
}

// ---- Request structs ----

type CreateCameraReq struct {
	Name    string `json:"name" binding:"required"`
	RtspURL string `json:"rtsp_url" binding:"required"`
	Location string `json:"location"`
}

type UpdateCameraReq struct {
	Name     string `json:"name"`
	RtspURL  string `json:"rtsp_url"`
	Location string `json:"location"`
	Status   *int   `json:"status"`
}

type CreateTaskReq struct {
	TaskName  string            `json:"task_name" binding:"required"`
	CameraID  int64             `json:"camera_id" binding:"required"`
	Remark    string            `json:"remark"`
	AlgoDetails []AlgoDetailReq `json:"algo_details" binding:"required,min=1"`
}

type UpdateTaskReq struct {
	TaskName    string          `json:"task_name" binding:"required"`
	CameraID    int64           `json:"camera_id" binding:"required"`
	Remark      string          `json:"remark"`
	AlgoDetails []AlgoDetailReq `json:"algo_details" binding:"required,min=1"`
}

type AlgoDetailReq struct {
	AlgoID      int64  `json:"algo_id" binding:"required"`
	RoiConfig   string `json:"roi_config"`
	AlgoParams  string `json:"algo_params"`
	AlarmConfig string `json:"alarm_config"`
}

type UpdateAlarmReq struct {
	Status int `json:"status"`
}

// ---- Model request structs ----

type CreateModelReq struct {
	ModelName     string  `json:"model_name" binding:"required"`
	ModelPath     string  `json:"model_path" binding:"required"`
	LabelsPath    string  `json:"labels_path"`
	ModelType     string  `json:"model_type"`
	InputWidth    int     `json:"input_width"`
	InputHeight   int     `json:"input_height"`
	ConfThreshold float64 `json:"conf_threshold"`
	NmsThreshold  float64 `json:"nms_threshold"`
}

type UpdateModelReq struct {
	ModelName     string  `json:"model_name"`
	ModelPath     string  `json:"model_path"`
	LabelsPath    string  `json:"labels_path"`
	ModelType     string  `json:"model_type"`
	InputWidth    int     `json:"input_width"`
	InputHeight   int     `json:"input_height"`
	ConfThreshold float64 `json:"conf_threshold"`
	NmsThreshold  float64 `json:"nms_threshold"`
}

// ---- Algorithm request structs ----

type CreateAlgorithmReq struct {
	AlgoKey         string  `json:"algo_key" binding:"required"`
	AlgoName        string  `json:"algo_name" binding:"required"`
	Category        string  `json:"category"`
	ParamDefinition string  `json:"param_definition"`
	ModelIDs        []int64 `json:"model_ids"`
}

type UpdateAlgorithmReq struct {
	AlgoKey         string  `json:"algo_key"`
	AlgoName        string  `json:"algo_name"`
	Category        string  `json:"category"`
	ParamDefinition string  `json:"param_definition"`
	ModelIDs        []int64 `json:"model_ids"`
}
