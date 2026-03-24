#!/bin/bash
# Start AI Monitor Go Backend
export PATH=$PATH:/home/hzhy/go/bin
export GIN_MODE=release
export DB_PATH=/home/hzhy/aimonitor.db
export ZLM_BASE_URL=http://127.0.0.1:80
export ZLM_SECRET=vEq3Z2BobQevk5dRs1zZ6DahIt5U9urT
export PYTHON_URL=http://127.0.0.1:9500
export PORT=:8090

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# 每次启动前编译，避免改了代码却只重启旧进程（表现为新接口一直 404）
echo "Building ai-monitor-backend..."
GOPATH=/home/hzhy/gopath GOPROXY=https://goproxy.cn,direct go build -o ai-monitor-backend . || exit 1

exec ./ai-monitor-backend
