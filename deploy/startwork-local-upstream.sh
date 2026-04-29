#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.startwork-upstream.yml"
PROJECT_NAME="${STARTWORK_UPSTREAM_SUB2API_PROJECT:-startwork-upstream-sub2api}"
IMAGE_NAME="${STARTWORK_SUB2API_IMAGE:-startwork/sub2api:latest}"
PORT="${STARTWORK_UPSTREAM_SUB2API_PORT:-18081}"

usage() {
  cat <<EOF
Usage: $(basename "$0") [up|build|down|logs|ps]

Startwork local upstream Sub2API instance.

Defaults:
  project: ${PROJECT_NAME}
  image:   ${IMAGE_NAME}
  url:     http://127.0.0.1:${PORT}

This instance is intentionally separate from the Startwork cluster Sub2API
service, which usually runs as project=startwork service=sub2api on port 18080.
EOF
}

command="${1:-up}"

case "${command}" in
  build)
    docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" build upstream-sub2api
    ;;
  up)
    docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" up -d --build
    echo "Startwork upstream Sub2API: http://127.0.0.1:${PORT}"
    ;;
  down)
    docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" down
    ;;
  logs)
    docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" logs -f upstream-sub2api
    ;;
  ps)
    docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" ps
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
