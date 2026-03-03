-- ============================================================
-- AI 视频监控平台数据库结构 (aimonitor.db)
-- 创建时间: 2026年2月27日
-- 数据库: SQLite3
-- ============================================================

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

-- 1. 摄像头表
CREATE TABLE cameras (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,                  -- 摄像头名称 (如: cam01)
    rtsp_url TEXT NOT NULL,              -- RTSP流地址
    location TEXT,                       -- 安装地点 (如: 临港办公室)
    status INTEGER DEFAULT 1 CHECK(status IN (0,1))  -- 1: 在线, 0: 离线
);

-- 2. 算法能力字典表 (系统预设的所有算法)
CREATE TABLE algorithms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    algo_key TEXT UNIQUE,                -- 算法标识 (如: phone_detect, sleep_detect)
    algo_name TEXT NOT NULL,             -- 算法显示名 (如: 玩手机, 闭眼)
    category TEXT                        -- 算法分类 (如: 行为分析, 消防检测)
);

-- 3. 任务主表
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_name TEXT NOT NULL,             -- 任务名称
    camera_id INTEGER NOT NULL,          -- 关联摄像头
    alarm_device_id TEXT,                -- 告警音柱配置 (存储ID或配置)
    status INTEGER DEFAULT 0 CHECK(status IN (0,1,2)), -- 0:停止, 1:运行中, 2:异常
    error_msg TEXT,                      -- 错误信息 (如: 拉流失败)
    remark TEXT,                         -- 备注
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (camera_id) REFERENCES cameras(id)
);

-- 4. 任务-算法配置详情表 (核心：存储每个算法不一样的参数)
CREATE TABLE task_algo_details (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER,                     -- 关联任务
    algo_id INTEGER,                     -- 关联算法
    roi_config TEXT,                     -- JSON: 识别区域坐标 [[x,y],[x,y]...]
    algo_params TEXT,                    -- JSON: 闭眼时长、置信度等特定参数
    alarm_config TEXT,                   -- JSON: 告警间隔、时间段等
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (algo_id) REFERENCES algorithms(id),
    UNIQUE(task_id, algo_id)
);

-- 5. 报警记录表
CREATE TABLE alarms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER,                     -- 关联任务
    algo_name TEXT,                      -- 报警算法名 (冗余存储，方便查询)
    alarm_time DATETIME DEFAULT CURRENT_TIMESTAMP, -- 报警时间
    alarm_location TEXT,                 -- 报警地点 (通常取摄像头的location)
    image_url TEXT,                      -- 抓拍图路径 (如: /uploads/2024/02/27/abc.jpg)
    status INTEGER DEFAULT 0 CHECK(status IN (0,1)),  -- 处理状态: 0:未处理, 1:已处理
    alarm_details TEXT,                  -- JSON: 存储识别时的具体信息 (如置信度、目标框坐标)
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- 高频查询索引（按时间倒序查最近告警）
CREATE INDEX idx_alarms_time ON alarms(alarm_time DESC);

-- 按任务查询
CREATE INDEX idx_alarms_task_id ON alarms(task_id);

-- 按状态查询
CREATE INDEX idx_alarms_status ON alarms(status);

-- 6. zlmediakit流
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
