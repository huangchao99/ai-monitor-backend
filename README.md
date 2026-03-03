# AI Monitor Backend

> AI 智能视频监控系统 — Go 后端管理服务

本服务是系统的控制中枢，对前端提供完整的 REST API，并负责协调 ZLMediaKit 视频流管理与 Python 算法调度服务。

---

## 系统定位

```
Vue3 前端 (:5173)
     │ REST API
     ▼
Go 后端 (:8090)   ◄─── 本项目
     │
     ├──▶ ZLMediaKit (:80)          # 视频流代理管理
     ├──▶ Python 算法服务 (:9500)   # 任务启停控制
     └──▶ aimonitor.db (SQLite)     # 共享数据库
```

---

## 如何编译 Go 程序

**环境要求**：Go 1.22+，本机 Go 安装在 `/home/hzhy/go/`，编译前需将 `go` 加入 PATH。

```bash
# 设置环境变量（可选，若已全局配置可省略）
export PATH=$PATH:/home/hzhy/go/bin
export GOPATH=/home/hzhy/gopath
export GOPROXY=https://goproxy.cn,direct

# 进入项目目录并编译
cd /home/hzhy/ai-monitor-backend
go build -o ai-monitor-backend .
```

编译成功后，当前目录下会生成可执行文件 `ai-monitor-backend`。

---

## 如何启动后端

### 方式一：使用启动脚本（推荐）

```bash
/home/hzhy/ai-monitor-backend/start.sh
```

脚本会自动完成：
- 设置 `PATH`、`GIN_MODE=release`、数据库路径、ZLM/Python 服务地址等环境变量
- 若未找到已编译的 `ai-monitor-backend`，会先执行 `go build` 再启动
- 以后台进程方式运行可执行文件

### 方式二：手动启动

先按上文完成编译，再设置环境变量并运行：

```bash
cd /home/hzhy/ai-monitor-backend

export PATH=$PATH:/home/hzhy/go/bin
export GIN_MODE=release
export DB_PATH=/home/hzhy/aimonitor.db
export ZLM_BASE_URL=http://127.0.0.1:80
export ZLM_SECRET=vEq3Z2BobQevk5dRs1zZ6DahIt5U9urT
export PYTHON_URL=http://127.0.0.1:9500
export PORT=:8090

./ai-monitor-backend
```

启动后访问：`http://localhost:8090/api/health` 可验证服务是否正常。

---

## 目录结构

```
ai-monitor-backend/
├── main.go               # Gin 路由注册、CORS 配置、DB 初始化、服务入口
├── start.sh              # 一键启动脚本（含自动编译）
├── ai-monitor-backend    # 编译产物（binary）
├── config/
│   └── config.go         # 配置项（支持环境变量覆盖）
├── model/
│   └── model.go          # 所有数据结构定义（Camera, Task, Alarm, ZlmStream 等）
├── store/
│   └── store.go          # SQLite 全量 CRUD（含 zlm_streams 表自动建表）
├── db/
│   ├── aimonitor.sql     # 建表 SQL（参考）
│   └── aimonitor_insert_value.sql  # 测试数据 SQL（参考）
├── zlm/
│   └── client.go         # ZLMediaKit HTTP API 封装
├── pyservice/
│   └── client.go         # Python 算法服务 HTTP 客户端
└── api/
    ├── camera.go         # 摄像头 CRUD + ZLM 流控制接口
    ├── task.go           # 任务 CRUD + 启停（转发至 Python）+ 算法列表
    └── alarm.go          # 告警分页查询 + 状态更新
```

---

## REST API 汇总

### 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查（含 ZLM / Python 连通性探测） |

### 摄像头管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/cameras` | 获取摄像头列表 |
| POST | `/api/cameras` | 创建摄像头（自动调用 ZLM addStreamProxy 启动推流） |
| PUT | `/api/cameras/:id` | 更新摄像头信息 |
| DELETE | `/api/cameras/:id` | 删除摄像头 |
| POST | `/api/cameras/:id/stream/start` | 手动启动 ZLM 代理流 |
| POST | `/api/cameras/:id/stream/stop` | 停止 ZLM 代理流 |

### 算法字典

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/algorithms` | 获取算法字典（只读，数据来自 DB） |

### 任务管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tasks` | 获取任务列表 |
| POST | `/api/tasks` | 创建任务（含算法配置） |
| DELETE | `/api/tasks/:id` | 删除任务 |
| POST | `/api/tasks/:id/start` | 启动任务（转发至 Python 服务） |
| POST | `/api/tasks/:id/stop` | 停止任务（转发至 Python 服务） |

### 告警管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alarms` | 告警分页列表（支持 `?task_id=&status=&page=&size=` 筛选） |
| PUT | `/api/alarms/:id` | 更新告警状态（标记已处理） |

---

## 数据库

共享 SQLite 数据库：`/home/hzhy/aimonitor.db`

使用 `modernc.org/sqlite`（纯 Go 驱动，无需 CGO/gcc），设置 `SetMaxOpenConns(1)` 避免 SQLite 写冲突，数据库以 WAL 模式运行支持多读单写。

**核心表：**

| 表名 | 说明 |
|------|------|
| `cameras` | 摄像头信息（名称、RTSP 地址、位置） |
| `zlm_streams` | ZLM 代理流状态（stream_key、proxy_key） |
| `algorithms` | 算法字典（algo_key 对应 Python 插件文件名） |
| `tasks` | 监控任务（status: 0=停止, 1=运行, 2=异常） |
| `task_algo_details` | 任务-算法配置（algo_params / alarm_config / roi_config JSON） |
| `alarms` | 报警记录（抓图路径、处理状态） |

---

## ZLMediaKit 集成

| 配置项 | 值 |
|--------|-----|
| 服务地址 | `http://localhost:80` |
| API 密钥 | `vEq3Z2BobQevk5dRs1zZ6DahIt5U9urT` |
| 流 ID 格式 | `cam{camera_id}` |
| HTTP-FLV 播放 | `http://localhost:80/live/cam{id}.live.flv` |

创建摄像头时，后端自动调用 ZLM `addStreamProxy` 拉取 RTSP 流并对外提供 HTTP-FLV 直播地址，供前端 mpegts.js 播放。

### 直播数据流路径

从摄像头到浏览器画面，数据经过以下四个环节，其中 Go 后端负责在关键操作节点写入数据库：

```
用户操作（前端）
    │  POST /api/cameras  或  POST /api/cameras/:id/stream/start
    ▼
Go 后端
    ├─ [DB写入] INSERT cameras 表（仅创建时）
    │   记录摄像头名称、RTSP 地址、位置
    │
    ├─ 调用 ZLM addStreamProxy API
    │   告知 ZLM 去拉取 RTSP 流
    │
    └─ [DB写入] UPSERT zlm_streams 表
        记录 stream_key=cam{id}、proxy_key（ZLM返回）
        status: 1=推流中 / 2=异常
    ▼
ZLMediaKit (:80)
    │  主动拉取 rtsp://摄像头IP/...
    │  RTSP → 转封装为 FLV（不重新编码，仅换容器格式）
    │  对外暴露 HTTP-FLV 长连接
    │  URL: http://工控机IP/live/cam{camera_id}.live.flv
    ▼
浏览器 / mpegts.js
    │  通过 HTTP 长连接持续拉取 FLV 数据
    │  用 MSE (Media Source Extensions) 喂给 <video> 标签
    ▼
页面上的 <video> 元素（用户看到的画面）
```

**`zlm_streams` 表写入时机汇总：**

| 触发操作 | 写入行为 |
|----------|----------|
| `POST /api/cameras`（创建摄像头） | INSERT `cameras` 表；UPSERT `zlm_streams`（status=1 或 2） |
| `POST /api/cameras/:id/stream/start`（手动启流） | UPSERT `zlm_streams`（更新 proxy_key、status=1 或 2） |
| `PUT /api/cameras/:id`（修改 RTSP 地址） | 先停旧流，再 UPSERT `zlm_streams`（新 proxy_key、status=1 或 2） |
| `POST /api/cameras/:id/stream/stop`（停流） | UPSERT `zlm_streams`（proxy_key 清空、status=0） |
| `DELETE /api/cameras/:id`（删除摄像头） | 停止 ZLM 流；DELETE `cameras`（`zlm_streams` 因外键 CASCADE 自动删除） |

**关键细节：**

- ZLM 做的是**转封装**（remux），而非转码，CPU 占用极低
- 前端直播流 URL 直接指向 ZLM `:80` 端口，**不经过 Go 后端**，避免后端成为视频带宽瓶颈
- 前端使用 `mpegts.js` 的追帧模式（`liveBufferLatencyChasing`），将端到端延迟控制在 0.5~3 秒
- `zlm_streams` 使用 `ON CONFLICT(camera_id) DO UPDATE`（Upsert），同一摄像头只保留一条流状态记录

**直播流方案对比：**

| 方案 | 延迟 | 说明 |
|------|------|------|
| **HTTP-FLV**（当前） | ~1-3 秒 | 延迟低，兼容性好，ZLM 原生支持 |
| HLS | ~5-30 秒 | 延迟高，不适合实时监控 |
| WebRTC | <1 秒 | 延迟最低，但信令复杂 |

> **注意：PCMA 音频兼容性**
> IP 摄像头通常使用 PCMA（G.711 A-law）音频，ZLM 会将其原样封装进 HTTP-FLV（音频 codec ID = 7）。浏览器 MSE 不支持该音频编码，会导致前端播放失败（`CodecUnsupported`）。前端已通过设置 `hasAudio: false` 绕过此问题（详见前端 README）。若后续需要音频预览，可将 ZLM 的 `[ffmpeg] bin` 配置指向 `/opt/ffmpeg-rk/bin/ffmpeg`（支持 `hevc_rkmpp` 解码 + `h264_rkmpp` 编码），并改用 `addFFmpegSource` API 在推流时将音频转码为 AAC。

---

## Python 服务集成

任务启停请求通过 HTTP 转发至 Python 算法调度服务（`localhost:9500`）：

- 启动任务 → `POST http://localhost:9500/api/task/start`
- 停止任务 → `POST http://localhost:9500/api/task/stop`

---

## 生产环境部署（Go 托管前端）

无需 Nginx，Go 后端可直接托管编译后的前端静态文件：

```bash
# 第一步：构建前端
cd /home/hzhy/ai-monitor-frontend
npm run build
```

然后在 `main.go` 中添加静态文件路由：

```go
r.Static("/assets", "/home/hzhy/ai-monitor-frontend/dist/assets")
r.StaticFile("/favicon.ico", "/home/hzhy/ai-monitor-frontend/dist/favicon.ico")
r.NoRoute(func(c *gin.Context) {
    c.File("/home/hzhy/ai-monitor-frontend/dist/index.html")
})
```

部署后通过 `http://工控机IP:8090` 直接访问完整 Web 界面。

---

## 技术栈

- **语言**：Go 1.22.5（`/home/hzhy/go/`，ARM64）
- **Web 框架**：Gin
- **数据库驱动**：`modernc.org/sqlite`（纯 Go，无 CGO 依赖）
- **模块缓存**：`/home/hzhy/gopath/`
- **下载代理**：`https://goproxy.cn,direct`

---

## 相关文档

- 系统总体介绍：`/home/hzhy/ai-monitor-intro.md`
- 数据库建表 SQL：`/home/hzhy/aimonitor.sql`
- Python 算法服务：`/home/hzhy/ai-monitor-service/README.md`
- C++ 推理服务 API：`/home/hzhy/infer-server/infer-server/docs/api_reference.md`
