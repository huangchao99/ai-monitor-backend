#!/bin/bash
# Development helper for starting AI Monitor Go Backend.
export PATH="${PATH}:/home/hzhy/go/bin"
export GIN_MODE=release
export AI_MONITOR_DB_PATH="${AI_MONITOR_DB_PATH:-/var/lib/ai-monitor/aimonitor.db}"
export AI_MONITOR_ZLM_BASE_URL="${AI_MONITOR_ZLM_BASE_URL:-http://127.0.0.1:80}"
export AI_MONITOR_ZLM_SECRET="${AI_MONITOR_ZLM_SECRET:-vEq3Z2BobQevk5dRs1zZ6DahIt5U9urT}"
export AI_MONITOR_PYTHON_URL="${AI_MONITOR_PYTHON_URL:-http://127.0.0.1:9500}"
export AI_MONITOR_BACKEND_PORT="${AI_MONITOR_BACKEND_PORT:-8090}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# 每次启动前编译，避免改了代码却只重启旧进程（表现为新接口一直 404）
echo "Building ai-monitor-backend..."
GOPATH=/home/hzhy/gopath GOPROXY=https://goproxy.cn,direct go build -o ai-monitor-backend . || exit 1

exec ./ai-monitor-backend
