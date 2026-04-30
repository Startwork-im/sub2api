#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
LOG_DIR="${SCRIPT_DIR}/logs"
LOG_FILE="${LOG_DIR}/fork_auto_update.log"
STATE_FILE="${LOG_DIR}/fork_auto_update.state"
LOCK_DIR="${LOG_DIR}/.fork_auto_update.lock"
LOCK_PID_FILE="${LOCK_DIR}/pid"

mkdir -p "$LOG_DIR"

[ -f "$HOME/.bashrc" ] && source "$HOME/.bashrc"
[ -f "$HOME/.profile" ] && source "$HOME/.profile"
export PATH=$PATH:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin

FORCE_MODE=false
[ "${1:-}" = "--force" ] && FORCE_MODE=true
LOCK_HELD=false

log() {
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $1"
    echo "$msg"
    echo "$msg" >> "$LOG_FILE"
}

send_lark_msg() {
    local msg="$1"
    if [ -z "${FORK_AUTO_UPDATE_LARK_WEBHOOK:-}" ]; then
        return 0
    fi
    if command -v python3 &>/dev/null; then
        local payload
        payload=$(python3 -c 'import json, sys; text = sys.argv[1].replace("\\n", "\n"); print(json.dumps({"msg_type": "text", "content": {"text": text}}, ensure_ascii=False))' "$msg" 2>&1)
        [ $? -eq 0 ] || return 1
        curl -sS -X POST -H "Content-Type: application/json" -d "$payload" "$FORK_AUTO_UPDATE_LARK_WEBHOOK" >> "$LOG_FILE" 2>&1 || true
    fi
}

fail() {
    log "❌ $1"
    local log_tail=""
    [ -f "$LOG_FILE" ] && log_tail=$(tail -n 20 "$LOG_FILE")
    send_lark_msg "❌ Sub2API fork 自动更新失败: $1\n📋 日志:\n$log_tail"
    exit 1
}

release_lock() {
    if [ "$LOCK_HELD" != "true" ]; then
        return
    fi
    rm -f "$LOCK_PID_FILE"
    rmdir "$LOCK_DIR" 2>/dev/null || true
    LOCK_HELD=false
}

acquire_lock() {
    if mkdir "$LOCK_DIR" 2>/dev/null; then
        printf '%s\n' "$$" > "$LOCK_PID_FILE"
        LOCK_HELD=true
        trap 'release_lock' EXIT
        trap 'release_lock; exit 1' INT TERM
        return 0
    fi

    local existing_pid=""
    if [ -f "$LOCK_PID_FILE" ]; then
        existing_pid=$(cat "$LOCK_PID_FILE" 2>/dev/null || true)
    fi
    if [ -n "$existing_pid" ] && kill -0 "$existing_pid" 2>/dev/null; then
        log "已有 fork 自动更新进程正在运行 (PID: $existing_pid)，跳过本次执行"
        exit 0
    fi

    rm -f "$LOCK_PID_FILE"
    rmdir "$LOCK_DIR" 2>/dev/null || true
    mkdir "$LOCK_DIR" 2>/dev/null || { log "获取更新锁失败"; exit 0; }
    printf '%s\n' "$$" > "$LOCK_PID_FILE"
    LOCK_HELD=true
    trap 'release_lock' EXIT
    trap 'release_lock; exit 1' INT TERM
}

load_env() {
    [ -f "$ENV_FILE" ] || fail "缺少环境文件: $ENV_FILE"
    set -a
    . "$ENV_FILE"
    set +a

    FORK_AUTO_UPDATE_ENABLED="${FORK_AUTO_UPDATE_ENABLED:-false}"
    FORK_AUTO_UPDATE_REMOTE="${FORK_AUTO_UPDATE_REMOTE:-origin}"
    FORK_AUTO_UPDATE_BRANCH_PREFIX="${FORK_AUTO_UPDATE_BRANCH_PREFIX:-upstream-v}"
    FORK_AUTO_UPDATE_COMPOSE_FILE="${FORK_AUTO_UPDATE_COMPOSE_FILE:-docker-compose.local.yml}"
    FORK_AUTO_UPDATE_IMAGE="${FORK_AUTO_UPDATE_IMAGE:-startwork/sub2api:managed}"
    SERVER_PORT="${SERVER_PORT:-8080}"

    if [ "$FORK_AUTO_UPDATE_ENABLED" != "true" ] && [ "$FORCE_MODE" != "true" ]; then
        log "FORK_AUTO_UPDATE_ENABLED 未开启，跳过"
        exit 0
    fi
}

compose() {
    docker compose -f "$FORK_AUTO_UPDATE_COMPOSE_FILE" "$@"
}

load_state() {
    DEPLOYED_BRANCH=""
    DEPLOYED_COMMIT=""
    [ -f "$STATE_FILE" ] || return 0
    while IFS='=' read -r key value; do
        case "$key" in
            BRANCH) DEPLOYED_BRANCH="$value" ;;
            COMMIT) DEPLOYED_COMMIT="$value" ;;
        esac
    done < "$STATE_FILE"
}

save_state() {
    cat > "$STATE_FILE" <<EOF
BRANCH=$1
COMMIT=$2
EOF
}

resolve_latest_branch() {
    git -C "$REPO_DIR" for-each-ref --format='%(refname:short)' "refs/remotes/${FORK_AUTO_UPDATE_REMOTE}/${FORK_AUTO_UPDATE_BRANCH_PREFIX}*" \
        | sed "s#^${FORK_AUTO_UPDATE_REMOTE}/##" \
        | sort -V \
        | tail -n 1
}

acquire_lock
load_env
load_state

[ -d "${REPO_DIR}/.git" ] || fail "当前目录不是 git 仓库 checkout: $REPO_DIR"
[ -f "${REPO_DIR}/Dockerfile" ] || fail "仓库根目录缺少 Dockerfile: ${REPO_DIR}/Dockerfile"
[ -f "${SCRIPT_DIR}/${FORK_AUTO_UPDATE_COMPOSE_FILE}" ] || fail "缺少 compose 文件: ${SCRIPT_DIR}/${FORK_AUTO_UPDATE_COMPOSE_FILE}"

log "开始检查 fork 更新..."
timeout 30 git -C "$REPO_DIR" fetch "$FORK_AUTO_UPDATE_REMOTE" --prune >> "$LOG_FILE" 2>&1 || fail "git fetch 失败"

TARGET_BRANCH="$(resolve_latest_branch)"
[ -n "$TARGET_BRANCH" ] || fail "未找到 ${FORK_AUTO_UPDATE_BRANCH_PREFIX}* 分支"
TARGET_COMMIT="$(git -C "$REPO_DIR" rev-parse "${FORK_AUTO_UPDATE_REMOTE}/${TARGET_BRANCH}")"
LOCAL_BRANCH="$(git -C "$REPO_DIR" symbolic-ref --short HEAD 2>/dev/null || echo detached)"
LOCAL_COMMIT="$(git -C "$REPO_DIR" rev-parse HEAD 2>/dev/null || echo '')"

NEED_UPDATE=false
UPDATE_REASON=""
if [ "$FORCE_MODE" = "true" ]; then
    NEED_UPDATE=true
    UPDATE_REASON="手动强制更新"
elif [ ! -f "$STATE_FILE" ]; then
    NEED_UPDATE=true
    UPDATE_REASON="首次启用 fork 自动更新"
elif [ "$DEPLOYED_BRANCH" != "$TARGET_BRANCH" ]; then
    NEED_UPDATE=true
    UPDATE_REASON="发现新 tag 分支 ${TARGET_BRANCH}"
elif [ "$DEPLOYED_COMMIT" != "$TARGET_COMMIT" ]; then
    NEED_UPDATE=true
    UPDATE_REASON="检测到 ${TARGET_BRANCH} 新提交 ${DEPLOYED_COMMIT:0:7} -> ${TARGET_COMMIT:0:7}"
elif [ "$LOCAL_BRANCH" != "$TARGET_BRANCH" ] || [ "$LOCAL_COMMIT" != "$TARGET_COMMIT" ]; then
    NEED_UPDATE=true
    UPDATE_REASON="本地 checkout 与目标分支不同步"
fi

if [ "$NEED_UPDATE" != "true" ]; then
    log "无更新"
    exit 0
fi

log "🚀 ${UPDATE_REASON}"
git -C "$REPO_DIR" checkout -B "$TARGET_BRANCH" "${FORK_AUTO_UPDATE_REMOTE}/${TARGET_BRANCH}" >> "$LOG_FILE" 2>&1 || fail "切换目标分支失败"
git -C "$REPO_DIR" reset --hard "${FORK_AUTO_UPDATE_REMOTE}/${TARGET_BRANCH}" >> "$LOG_FILE" 2>&1 || fail "重置到远端分支失败"

export SUB2API_IMAGE="$FORK_AUTO_UPDATE_IMAGE"
log "构建 Sub2API 镜像 ${SUB2API_IMAGE}..."
docker build -t "$SUB2API_IMAGE" "$REPO_DIR" >> "$LOG_FILE" 2>&1 || fail "Sub2API 镜像构建失败"

log "重建 Sub2API 容器..."
compose up -d --no-deps sub2api >> "$LOG_FILE" 2>&1 || fail "Sub2API 容器更新失败"

READY=false
for i in $(seq 1 30); do
    if curl -fsS "http://127.0.0.1:${SERVER_PORT}/health" >/dev/null 2>&1; then
        READY=true
        break
    fi
    sleep 2
done
[ "$READY" = "true" ] || fail "Sub2API 健康检查失败"

save_state "$TARGET_BRANCH" "$TARGET_COMMIT"
SHORT_HEAD="${TARGET_COMMIT:0:7}"
COMMIT_TIME=$(git -C "$REPO_DIR" log -1 --format=%cd --date=format:'%Y-%m-%d %H:%M')

log "✅ 更新完成: ${TARGET_BRANCH} @ ${SHORT_HEAD}"
send_lark_msg "✅ Sub2API fork 自动更新成功!\n🌿 分支: ${TARGET_BRANCH}\n📦 版本: ${SHORT_HEAD}\n🕒 时间: ${COMMIT_TIME}\n📝 原因: ${UPDATE_REASON}"
