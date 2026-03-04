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
	_, err := s.db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;

		CREATE TABLE IF NOT EXISTS zlm_streams (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			camera_id  INTEGER NOT NULL UNIQUE,
			app        TEXT    NOT NULL DEFAULT 'live',
			stream_key TEXT    NOT NULL,
			proxy_key  TEXT    DEFAULT '',
			status     INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (camera_id) REFERENCES cameras(id) ON DELETE CASCADE
		);
	`)
	return err
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

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id=?", id)
	return err
}

func (s *Store) UpdateTaskStatus(id int64, status int, errMsg string) error {
	_, err := s.db.Exec(
		"UPDATE tasks SET status=?, error_msg=? WHERE id=?",
		status, errMsg, id,
	)
	return err
}

// GetRunningTaskIDsByCamera returns IDs of running tasks for the given camera.
func (s *Store) GetRunningTaskIDsByCamera(cameraID int64) ([]int64, error) {
	rows, err := s.db.Query("SELECT id FROM tasks WHERE camera_id=? AND status=1", cameraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
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
		       t.task_name, c.name
		FROM alarms a
		JOIN tasks t ON t.id = a.task_id
		JOIN cameras c ON c.id = t.camera_id
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
		if err := rows.Scan(
			&a.ID, &a.TaskID, &a.AlgoName, &a.AlarmTime,
			&a.AlarmLocation, &a.ImageURL,
			&a.Status, &a.AlarmDetails,
			&a.TaskName, &a.CameraName,
		); err != nil {
			return nil, 0, err
		}
		alarms = append(alarms, a)
	}
	return alarms, total, nil
}

func (s *Store) UpdateAlarmStatus(id int64, status int) error {
	_, err := s.db.Exec("UPDATE alarms SET status=? WHERE id=?", status, id)
	return err
}
