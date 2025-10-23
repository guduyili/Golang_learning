#!/bin/bash
set -euo pipefail

# 核心配置（适配你的目录结构）
APP_BIN="/root/Golang_learning/IMtest/imtest"
APP_NAME="IMtest"
LOG_FILE="/opt/apps/logs/${APP_NAME}.log"
PID_FILE="/opt/apps/run/${APP_NAME}.pid"
# 应用根目录（用于切换工作目录，确保 ./static 可被定位）
APP_HOME="/root/Golang_learning/IMtest"

# 检查可执行文件是否存在
check_app_binary() {
    if [ ! -f "$APP_BIN" ]; then  # 修正：使用兼容的[ ]语法，避免[[ ]]的潜在问题
        echo "❌  应用可执行文件不存在：$APP_BIN"
        echo "   提示：请确认编译命令是否正确（如 go build -o imtest）"
        exit 1
    fi
    if [ ! -x "$APP_BIN" ]; then  # 修正：使用[ ]替代[[ ]]，确保兼容性
        echo "⚠  可执行文件无执行权限，正在自动添加..."
        chmod +x "$APP_BIN"
    fi
}

# 确保日志/PID目录存在
ensure_dirs() {
    mkdir -p "$(dirname "$LOG_FILE")" "$(dirname "$PID_FILE")"
    chmod 755 "$(dirname "$LOG_FILE")" "$(dirname "$PID_FILE")"
}

# 检查应用是否正在运行（修正条件判断语法）
is_running() {
    if [ -f "$PID_FILE" ]; then  # 修正：使用[ ]替代[[ ]]
        local pid
        pid=$(cat "$PID_FILE" 2>/dev/null || true)
        # 分步骤判断，避免复杂表达式导致的语法错误
        if [ -n "$pid" ] && [ -d "/proc/$pid" ]; then  # 修正：拆分条件为两个[ ]，用&&连接
            if grep -q "$APP_BIN" "/proc/$pid/cmdline" 2>/dev/null; then
                return 0
            fi
        fi
    fi
    if pgrep -f "$APP_BIN" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

# 获取当前运行的PID
get_pid() {
    if [ -f "$PID_FILE" ]; then  # 修正：使用[ ]替代[[ ]]
        local pid
        pid=$(cat "$PID_FILE" 2>/dev/null || true)
        if [ -n "$pid" ] && [ -d "/proc/$pid" ] && grep -q "$APP_BIN" "/proc/$pid/cmdline" 2>/dev/null; then
            echo "$pid"
            return 0
        fi
    fi
    pgrep -f "$APP_BIN" | head -n 1
}

# 启动应用
start() {
    check_app_binary
    ensure_dirs

    if is_running; then
        echo "⚠  $APP_NAME 已在运行（PID: $(get_pid)）"
        status
        return 0
    fi

    echo "$(date +'%F %T') ▶ 正在启动 $APP_NAME..."
    echo "   可执行文件：$APP_BIN"
    echo "   日志文件：$LOG_FILE"

        # 切换到应用根目录再启动，确保代码中的 http.FileServer(http.Dir("./static")) 能找到静态资源
        (
            cd "$APP_HOME"
            nohup "$APP_BIN" >> "$LOG_FILE" 2>&1 &
        )
    local pid=$!
    echo "$pid" > "$PID_FILE"

    sleep 2
    if is_running; then
        echo "✅  $APP_NAME 启动成功（PID: $pid）"
    else
        echo "❌  $APP_NAME 启动失败！请查看日志：tail -n 20 $LOG_FILE"
        rm -f "$PID_FILE"
        exit 1
    fi
}

# 停止应用
stop() {
    if ! is_running; then
        echo "⚠  $APP_NAME 未在运行"
        return 0
    fi

    local pid=$(get_pid)
    echo "$(date +'%F %T') ▶ 正在停止 $APP_NAME（PID: $pid）..."

    kill "$pid" 2>/dev/null

    local timeout=30
    local count=0
    while is_running && [ $count -lt $timeout ]; do  # 修正：使用[ ]替代[[ ]]
        sleep 1
        count=$((count + 1))
    done

    if is_running; then
        echo "⚠  优雅停止超时（${timeout}秒），正在强制终止..."
        kill -9 "$pid" 2>/dev/null
        sleep 2
    fi

    if is_running; then
        echo "❌  无法停止 $APP_NAME，请手动执行：kill -9 $pid"
        exit 1
    else
        echo "✅  $APP_NAME 已停止"
        rm -f "$PID_FILE"
    fi
}

# 重启应用
restart() {
    echo "$(date +'%F %T') ▶ 正在重启 $APP_NAME..."
    stop
    sleep 1
    start
}

# 查看应用状态
status() {
    echo "$(date +'%F %T') ▶ $APP_NAME 状态"
    if is_running; then
        local pid=$(get_pid)
        local start_time=$(ps -p "$pid" -o lstart= | sed 's/^ *//')
        local cpu_usage=$(ps -p "$pid" -o %cpu= | sed 's/^ *//')
        local mem_usage=$(ps -p "$pid" -o %mem= | sed 's/^ *//')
        echo "   🟢 运行中"
        echo "   PID:       $pid"
        echo "   启动时间:  $start_time"
        echo "   CPU使用率: $cpu_usage%"
        echo "   内存使用率: $mem_usage%"
        echo "   日志路径:  $LOG_FILE"
    else
        echo "   🔴 未运行"
        echo "   日志路径:  $LOG_FILE"
    fi
}

# 查看日志
logs() {
    if [ ! -f "$LOG_FILE" ]; then  # 修正：使用[ ]替代[[ ]]
        echo "❌  日志文件不存在（应用可能未启动过）：$LOG_FILE"
        exit 1
    fi

    echo "📄 $APP_NAME 日志（按 Ctrl+C 退出）"
    echo "   日志路径：$LOG_FILE"
    echo "----------------------------------------"
    if [ "$#" -eq 1 ] && [ "$1" = "follow" ]; then  # 修正：使用[ ]和=替代[[ ]]和==
        tail -f "$LOG_FILE"
    else
        tail -n 50 "$LOG_FILE"
        echo "----------------------------------------"
        echo "提示：执行 ./$0 logs follow 实时查看日志"
    fi
}

# 帮助信息
usage() {
    echo "用法：$0 [命令]"
    echo "命令说明（针对 $APP_NAME 应用）："
    echo "  start    启动应用"
    echo "  stop     停止应用"
    echo "  restart  重启应用"
    echo "  status   查看状态"
    echo "  logs     查看日志（加 follow 实时跟踪）"
    echo "  help     显示帮助"
}

# 解析命令
case "${1:-}" in
    start)    start    ;;
    stop)     stop     ;;
    restart)  restart  ;;
    status)   status   ;;
    logs)     logs "${2:-}" ;;
    help)     usage    ;;
    *)
        echo "❌  未知命令：${1:-}"
        usage
        exit 1
        ;;
esac
    