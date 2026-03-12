package store

import (
	"database/sql"
	"fmt"
	"time"

	"ai-monitor-backend/model"

	_ "modernc.org/sqlite"
)

// Store holds the DB handle.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite DB and runs migrations.
func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite is not thread-safe with multiple writers
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`CREATE TABLE IF NOT EXISTS zlm_streams (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			camera_id  INTEGER NOT NULL UNIQUE,
			app        TEXT    NOT NULL DEFAULT 'live',
			stream_key TEXT    NOT NULL,
			proxy_key  TEXT    DEFAULT '',
			status     INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (camera_id) REFERENCES cameras(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS models (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			model_name      TEXT NOT NULL,
			model_path      TEXT NOT NULL,
			labels_path     TEXT DEFAULT '',
			model_type      TEXT DEFAULT 'yolov11',
			input_width     INTEGER DEFAULT 640,
			input_height    INTEGER DEFAULT 640,
			conf_threshold  REAL DEFAULT 0.35,
			nms_threshold   REAL DEFAULT 0.45
		)`,
		`CREATE TABLE IF NOT EXISTS algo_model_map (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			algo_id  INTEGER NOT NULL,
			model_id INTEGER NOT NULL,
			UNIQUE(algo_id, model_id),
			FOREIGN KEY (algo_id)  REFERENCES algorithms(id) ON DELETE CASCADE,
			FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS system_settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`INSERT OR IGNORE INTO system_settings (key, value) VALUES
			('voice_alarm_enabled', '0'),
			('voice_device_ip',     ''),
			('voice_device_user',   ''),
			('voice_device_pass',   '')`,
		`CREATE TABLE IF NOT EXISTS voice_alarm_algo_map (
			algo_id    INTEGER PRIMARY KEY,
			audio_file TEXT NOT NULL,
			FOREIGN KEY (algo_id) REFERENCES algorithms(id) ON DELETE CASCADE
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w (stmt: %s)", err, stmt[:min(40, len(stmt))])
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Cameras ──────────────────────────────────────────────────

func (s *Store) ListCameras() ([]model.Camera, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.rtsp_url, c.location, c.status,
		       z.id, z.camera_id, z.app, z.stream_key, z.proxy_key, z.status, z.created_at
		FROM cameras c
		LEFT JOIN zlm_streams z ON z.camera_id = c.id
		ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []model.Camera
	for rows.Next() {
		var c model.Camera
		var z model.ZlmStream
		var zID sql.NullInt64
		var zCamID sql.NullInt64
		var zApp, zStreamKey, zProxyKey sql.NullString
		var zStatus sql.NullInt64
		var zCreatedAt sql.NullTime
		if err := rows.Scan(
			&c.ID, &c.Name, &c.RtspURL, &c.Location, &c.Status,
			&zID, &zCamID, &zApp, &zStreamKey, &zProxyKey, &zStatus, &zCreatedAt,
		); err != nil {
			return nil, err
		}
		if zID.Valid {
			z.ID = zID.Int64
			z.CameraID = zCamID.Int64
			z.App = zApp.String
			z.StreamKey = zStreamKey.String
			z.ProxyKey = zProxyKey.String
			z.Status = int(zStatus.Int64)
			z.CreatedAt = zCreatedAt.Time
			c.ZlmStream = &z
		}
		cameras = append(cameras, c)
	}
	return cameras, nil
}

func (s *Store) GetCamera(id int64) (*model.Camera, error) {
	cameras, err := s.ListCameras()
	if err != nil {
		return nil, err
	}
	for i := range cameras {
		if cameras[i].ID == id {
			return &cameras[i], nil
		}
	}
	return nil, fmt.Errorf("camera %d not found", id)
}

func (s *Store) CreateCamera(name, rtspURL, location string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO cameras (name, rtsp_url, location, status) VALUES (?, ?, ?, 1)",
		name, rtspURL, location,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateCamera(id int64, name, rtspURL, location string, status *int) error {
	q := "UPDATE cameras SET name=?, rtsp_url=?, location=?"
	args := []any{name, rtspURL, location}
	if status != nil {
		q += ", status=?"
		args = append(args, *status)
	}
	q += " WHERE id=?"
	args = append(args, id)
	_, err := s.db.Exec(q, args...)
	return err
}

func (s *Store) DeleteCamera(id int64) error {
	_, err := s.db.Exec("DELETE FROM cameras WHERE id=?", id)
	return err
}

// ─── ZlmStreams ────────────────────────────────────────────────

func (s *Store) UpsertZlmStream(cameraID int64, streamKey, proxyKey string, status int) error {
	_, err := s.db.Exec(`
		INSERT INTO zlm_streams (camera_id, app, stream_key, proxy_key, status, created_at)
		VALUES (?, 'live', ?, ?, ?, ?)
		ON CONFLICT(camera_id) DO UPDATE SET
			stream_key=excluded.stream_key,
			proxy_key=excluded.proxy_key,
			status=excluded.status`,
		cameraID, streamKey, proxyKey, status, time.Now(),
	)
	return err
}

func (s *Store) UpdateZlmStreamStatus(cameraID int64, status int) error {
	_, err := s.db.Exec(
		"UPDATE zlm_streams SET status=? WHERE camera_id=?", status, cameraID,
	)
	return err
}

func (s *Store) GetZlmStream(cameraID int64) (*model.ZlmStream, error) {
	row := s.db.QueryRow(
		"SELECT id, camera_id, app, stream_key, proxy_key, status, created_at FROM zlm_streams WHERE camera_id=?",
		cameraID,
	)
	var z model.ZlmStream
	if err := row.Scan(&z.ID, &z.CameraID, &z.App, &z.StreamKey, &z.ProxyKey, &z.Status, &z.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &z, nil
}

func (s *Store) DeleteZlmStream(cameraID int64) error {
	_, err := s.db.Exec("DELETE FROM zlm_streams WHERE camera_id=?", cameraID)
	return err
}

// ─── Algorithms ───────────────────────────────────────────────

func (s *Store) ListAlgorithms() ([]model.Algorithm, error) {
	rows, err := s.db.Query("SELECT id, algo_key, algo_name, COALESCE(category,''), COALESCE(param_definition,'') FROM algorithms ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var algos []model.Algorithm
	for rows.Next() {
		var a model.Algorithm
		if err := rows.Scan(&a.ID, &a.AlgoKey, &a.AlgoName, &a.Category, &a.ParamDefinition); err != nil {
			return nil, err
		}
		algos = append(algos, a)
	}
	return algos, nil
}

// ─── Tasks ────────────────────────────────────────────────────

func (s *Store) ListTasks() ([]model.Task, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.task_name, t.camera_id, COALESCE(t.alarm_device_id,''),
		       t.status, COALESCE(t.error_msg,''), COALESCE(t.remark,''), t.created_at,
		       c.name,
		       (SELECT COUNT(*) FROM alarms WHERE task_id=t.id) as alarm_count
		FROM tasks t
		JOIN cameras c ON c.id = t.camera_id
		ORDER BY t.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(
			&t.ID, &t.TaskName, &t.CameraID, &t.AlarmDeviceID,
			&t.Status, &t.ErrorMsg, &t.Remark, &t.CreatedAt,
			&t.CameraName, &t.AlarmCount,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	// Load algo details for each task
	for i := range tasks {
		details, err := s.ListAlgoDetails(tasks[i].ID)
		if err != nil {
			return nil, err
		}
		tasks[i].AlgoDetails = details
	}
	return tasks, nil
}

func (s *Store) GetTask(id int64) (*model.Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %d not found", id)
}

func (s *Store) CreateTask(req model.CreateTaskReq) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO tasks (task_name, camera_id, remark, status) VALUES (?, ?, ?, 0)",
		req.TaskName, req.CameraID, req.Remark,
	)
	if err != nil {
		return 0, err
	}
	taskID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, d := range req.AlgoDetails {
		roiConfig := d.RoiConfig
		if roiConfig == "" {
			roiConfig = "[]"
		}
		algoParams := d.AlgoParams
		if algoParams == "" {
			algoParams = "{}"
		}
		alarmConfig := d.AlarmConfig
		if alarmConfig == "" {
			alarmConfig = "{}"
		}
		_, err = tx.Exec(
			"INSERT INTO task_algo_details (task_id, algo_id, roi_config, algo_params, alarm_config) VALUES (?, ?, ?, ?, ?)",
			taskID, d.AlgoID, roiConfig, algoParams, alarmConfig,
		)
		if err != nil {
			return 0, err
		}
	}

	return taskID, tx.Commit()
}

func (s *Store) UpdateTask(id int64, req model.UpdateTaskReq) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE tasks SET task_name=?, camera_id=?, remark=? WHERE id=?",
		req.TaskName, req.CameraID, req.Remark, id,
	); err != nil {
		return err
	}

	// 替换算法配置：先删旧的，再插新的
	if _, err := tx.Exec("DELETE FROM task_algo_details WHERE task_id=?", id); err != nil {
		return err
	}
	for _, d := range req.AlgoDetails {
		roiConfig := d.RoiConfig
		if roiConfig == "" {
			roiConfig = "[]"
		}
		algoParams := d.AlgoParams
		if algoParams == "" {
			algoParams = "{}"
		}
		alarmConfig := d.AlarmConfig
		if alarmConfig == "" {
			alarmConfig = "{}"
		}
		if _, err := tx.Exec(
			"INSERT INTO task_algo_details (task_id, algo_id, roi_config, algo_params, alarm_config) VALUES (?, ?, ?, ?, ?)",
			id, d.AlgoID, roiConfig, algoParams, alarmConfig,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteTask(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 解除报警与任务的关联，保留历史报警记录（task_name/camera_name 已快照在报警行里）
	if _, err := tx.Exec("UPDATE alarms SET task_id=NULL WHERE task_id=?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM tasks WHERE id=?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpdateTaskStatus(id int64, status int, errMsg string) error {
	_, err := s.db.Exec(
		"UPDATE tasks SET status=?, error_msg=? WHERE id=?",
		status, errMsg, id,
	)
	return err
}

// ─── AlgoDetails ──────────────────────────────────────────────

func (s *Store) ListAlgoDetails(taskID int64) ([]model.AlgoDetail, error) {
	rows, err := s.db.Query(`
		SELECT d.id, d.task_id, d.algo_id,
		       COALESCE(d.roi_config,'[]'), COALESCE(d.algo_params,'{}'), COALESCE(d.alarm_config,'{}'),
		       a.algo_key, a.algo_name
		FROM task_algo_details d
		JOIN algorithms a ON a.id = d.algo_id
		WHERE d.task_id=?`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var details []model.AlgoDetail
	for rows.Next() {
		var d model.AlgoDetail
		if err := rows.Scan(&d.ID, &d.TaskID, &d.AlgoID,
			&d.RoiConfig, &d.AlgoParams, &d.AlarmConfig,
			&d.AlgoKey, &d.AlgoName,
		); err != nil {
			return nil, err
		}
		details = append(details, d)
	}
	return details, nil
}

// ─── Alarms ───────────────────────────────────────────────────

func (s *Store) ListAlarms(taskID int64, algoName, startDate, endDate string, status int, page, size int) ([]model.Alarm, int, error) {
	if size <= 0 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * size

	where := "WHERE 1=1"
	args := []any{}
	if taskID > 0 {
		where += " AND a.task_id=?"
		args = append(args, taskID)
	}
	if algoName != "" {
		where += " AND a.algo_name=?"
		args = append(args, algoName)
	}
	if startDate != "" {
		where += " AND date(a.alarm_time) >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		where += " AND date(a.alarm_time) <= ?"
		args = append(args, endDate)
	}
	if status >= 0 {
		where += " AND a.status=?"
		args = append(args, status)
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	row := s.db.QueryRow("SELECT COUNT(*) FROM alarms a "+where, countArgs...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, size, offset)
	rows, err := s.db.Query(`
		SELECT a.id, a.task_id, a.algo_name,
		       strftime('%Y-%m-%d %H:%M:%S', a.alarm_time) AS alarm_time,
		       COALESCE(a.alarm_location,''), COALESCE(a.image_url,''),
		       a.status, COALESCE(a.alarm_details,'{}'),
		       COALESCE(a.task_name,''), COALESCE(a.camera_name,'')
		FROM alarms a
		`+where+`
		ORDER BY a.alarm_time DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var alarms []model.Alarm
	for rows.Next() {
		var a model.Alarm
		var taskID sql.NullInt64
		if err := rows.Scan(
			&a.ID, &taskID, &a.AlgoName, &a.AlarmTime,
			&a.AlarmLocation, &a.ImageURL,
			&a.Status, &a.AlarmDetails,
			&a.TaskName, &a.CameraName,
		); err != nil {
			return nil, 0, err
		}
		a.TaskID = taskID.Int64 // 0 when task has been deleted
		alarms = append(alarms, a)
	}
	return alarms, total, nil
}

func (s *Store) UpdateAlarmStatus(id int64, status int) error {
	_, err := s.db.Exec("UPDATE alarms SET status=? WHERE id=?", status, id)
	return err
}

// BatchDeleteAlarms removes multiple alarm records in one transaction and returns
// the image URLs (raw file paths) so the caller can clean up snapshot files.
func (s *Store) BatchDeleteAlarms(ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var imageURLs []string
	for _, id := range ids {
		var url string
		row := tx.QueryRow("SELECT COALESCE(image_url,'') FROM alarms WHERE id=?", id)
		if err := row.Scan(&url); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		if _, err := tx.Exec("DELETE FROM alarms WHERE id=?", id); err != nil {
			return nil, err
		}
		if url != "" {
			imageURLs = append(imageURLs, url)
		}
	}
	return imageURLs, tx.Commit()
}

// DeleteAlarm removes an alarm record and returns its image_url (raw path) so the caller can delete the file.
func (s *Store) DeleteAlarm(id int64) (imageURL string, err error) {
	row := s.db.QueryRow("SELECT COALESCE(image_url,'') FROM alarms WHERE id=?", id)
	if err = row.Scan(&imageURL); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("alarm %d not found", id)
		}
		return "", err
	}
	_, err = s.db.Exec("DELETE FROM alarms WHERE id=?", id)
	return imageURL, err
}

// ─── Models ───────────────────────────────────────────────────

func (s *Store) ListModels() ([]model.Model, error) {
	rows, err := s.db.Query(`
		SELECT id, model_name, model_path, COALESCE(labels_path,''),
		       COALESCE(model_type,'yolov11'), COALESCE(input_width,640),
		       COALESCE(input_height,640), COALESCE(conf_threshold,0.35),
		       COALESCE(nms_threshold,0.45)
		FROM models ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []model.Model
	for rows.Next() {
		var m model.Model
		if err := rows.Scan(&m.ID, &m.ModelName, &m.ModelPath, &m.LabelsPath,
			&m.ModelType, &m.InputWidth, &m.InputHeight,
			&m.ConfThreshold, &m.NmsThreshold); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (s *Store) GetModel(id int64) (*model.Model, error) {
	row := s.db.QueryRow(`
		SELECT id, model_name, model_path, COALESCE(labels_path,''),
		       COALESCE(model_type,'yolov11'), COALESCE(input_width,640),
		       COALESCE(input_height,640), COALESCE(conf_threshold,0.35),
		       COALESCE(nms_threshold,0.45)
		FROM models WHERE id=?`, id)
	var m model.Model
	if err := row.Scan(&m.ID, &m.ModelName, &m.ModelPath, &m.LabelsPath,
		&m.ModelType, &m.InputWidth, &m.InputHeight,
		&m.ConfThreshold, &m.NmsThreshold); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model %d not found", id)
		}
		return nil, err
	}
	return &m, nil
}

func (s *Store) CreateModel(req model.CreateModelReq) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO models (model_name, model_path, labels_path, model_type,
		                    input_width, input_height, conf_threshold, nms_threshold)
		VALUES (?,?,?,?,?,?,?,?)`,
		req.ModelName, req.ModelPath, req.LabelsPath, req.ModelType,
		req.InputWidth, req.InputHeight, req.ConfThreshold, req.NmsThreshold,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateModel(id int64, req model.UpdateModelReq) error {
	_, err := s.db.Exec(`
		UPDATE models SET model_name=?, model_path=?, labels_path=?, model_type=?,
		                  input_width=?, input_height=?, conf_threshold=?, nms_threshold=?
		WHERE id=?`,
		req.ModelName, req.ModelPath, req.LabelsPath, req.ModelType,
		req.InputWidth, req.InputHeight, req.ConfThreshold, req.NmsThreshold, id,
	)
	return err
}

func (s *Store) DeleteModel(id int64) error {
	// 检查是否被算法引用
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM algo_model_map WHERE model_id=?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该模型已被 %d 个算法引用，无法删除", count)
	}
	_, err := s.db.Exec("DELETE FROM models WHERE id=?", id)
	return err
}

// ─── Algorithms CRUD ──────────────────────────────────────────

func (s *Store) CreateAlgorithm(req model.CreateAlgorithmReq) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO algorithms (algo_key, algo_name, category, param_definition) VALUES (?,?,?,?)",
		req.AlgoKey, req.AlgoName, req.Category, req.ParamDefinition,
	)
	if err != nil {
		return 0, err
	}
	algoID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, mid := range req.ModelIDs {
		if _, err := tx.Exec("INSERT OR IGNORE INTO algo_model_map (algo_id, model_id) VALUES (?,?)", algoID, mid); err != nil {
			return 0, err
		}
	}
	return algoID, tx.Commit()
}

func (s *Store) UpdateAlgorithm(id int64, req model.UpdateAlgorithmReq) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE algorithms SET algo_key=?, algo_name=?, category=?, param_definition=? WHERE id=?",
		req.AlgoKey, req.AlgoName, req.Category, req.ParamDefinition, id,
	); err != nil {
		return err
	}
	// 替换关联模型
	if req.ModelIDs != nil {
		if _, err := tx.Exec("DELETE FROM algo_model_map WHERE algo_id=?", id); err != nil {
			return err
		}
		for _, mid := range req.ModelIDs {
			if _, err := tx.Exec("INSERT OR IGNORE INTO algo_model_map (algo_id, model_id) VALUES (?,?)", id, mid); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteAlgorithm(id int64) error {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM task_algo_details WHERE algo_id=?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该算法已被 %d 个任务使用，无法删除", count)
	}
	_, err := s.db.Exec("DELETE FROM algorithms WHERE id=?", id)
	return err
}

// ─── AlgoModelMap ─────────────────────────────────────────────

func (s *Store) ListAlgoModels(algoID int64) ([]model.Model, error) {
	rows, err := s.db.Query(`
		SELECT m.id, m.model_name, m.model_path, COALESCE(m.labels_path,''),
		       COALESCE(m.model_type,'yolov11'), COALESCE(m.input_width,640),
		       COALESCE(m.input_height,640), COALESCE(m.conf_threshold,0.35),
		       COALESCE(m.nms_threshold,0.45)
		FROM models m
		JOIN algo_model_map amm ON amm.model_id = m.id
		WHERE amm.algo_id=?`, algoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []model.Model
	for rows.Next() {
		var m model.Model
		if err := rows.Scan(&m.ID, &m.ModelName, &m.ModelPath, &m.LabelsPath,
			&m.ModelType, &m.InputWidth, &m.InputHeight,
			&m.ConfThreshold, &m.NmsThreshold); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

// ─── System Settings ──────────────────────────────────────────

func (s *Store) GetSystemSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM system_settings WHERE key=?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *Store) SetSystemSetting(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO system_settings (key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
		key, value,
	)
	return err
}

func (s *Store) GetVoiceAlarmSettings() (model.VoiceAlarmSettings, error) {
	rows, err := s.db.Query(
		"SELECT key, value FROM system_settings WHERE key IN ('voice_alarm_enabled','voice_device_ip','voice_device_user','voice_device_pass')",
	)
	if err != nil {
		return model.VoiceAlarmSettings{}, err
	}
	defer rows.Close()
	kv := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return model.VoiceAlarmSettings{}, err
		}
		kv[k] = v
	}
	return model.VoiceAlarmSettings{
		Enabled:    kv["voice_alarm_enabled"] == "1",
		DeviceIP:   kv["voice_device_ip"],
		DeviceUser: kv["voice_device_user"],
		DevicePass: kv["voice_device_pass"],
	}, nil
}

func (s *Store) SaveVoiceAlarmSettings(req model.UpdateVoiceAlarmSettingsReq) error {
	enabled := "0"
	if req.Enabled {
		enabled = "1"
	}
	pairs := [][2]string{
		{"voice_alarm_enabled", enabled},
		{"voice_device_ip", req.DeviceIP},
		{"voice_device_user", req.DeviceUser},
		{"voice_device_pass", req.DevicePass},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, p := range pairs {
		if _, err := tx.Exec(
			"INSERT INTO system_settings (key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
			p[0], p[1],
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ─── Voice Alarm Algo Map ─────────────────────────────────────

// ListVoiceAlarmAlgoMaps returns all algorithms joined with their voice mapping (if any).
func (s *Store) ListVoiceAlarmAlgoMaps() ([]model.VoiceAlarmAlgoMap, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.algo_key, a.algo_name, COALESCE(v.audio_file, '') AS audio_file
		FROM algorithms a
		LEFT JOIN voice_alarm_algo_map v ON v.algo_id = a.id
		ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.VoiceAlarmAlgoMap
	for rows.Next() {
		var m model.VoiceAlarmAlgoMap
		if err := rows.Scan(&m.AlgoID, &m.AlgoKey, &m.AlgoName, &m.AudioFile); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func (s *Store) SetVoiceAlarmAlgoMap(algoID int64, audioFile string) error {
	_, err := s.db.Exec(
		"INSERT INTO voice_alarm_algo_map (algo_id, audio_file) VALUES (?,?) ON CONFLICT(algo_id) DO UPDATE SET audio_file=excluded.audio_file",
		algoID, audioFile,
	)
	return err
}

func (s *Store) DeleteVoiceAlarmAlgoMap(algoID int64) error {
	_, err := s.db.Exec("DELETE FROM voice_alarm_algo_map WHERE algo_id=?", algoID)
	return err
}

func (s *Store) ListAlgorithmsWithModels() ([]model.Algorithm, error) {
	algos, err := s.ListAlgorithms()
	if err != nil {
		return nil, err
	}
	for i := range algos {
		models, err := s.ListAlgoModels(algos[i].ID)
		if err != nil {
			return nil, err
		}
		if models == nil {
			models = []model.Model{}
		}
		algos[i].Models = models
	}
	return algos, nil
}
