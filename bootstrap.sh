#!/usr/bin/env bash

set -euo pipefail

APP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export HTTP_ADDR="${HTTP_ADDR:-0.0.0.0:8787}"
export AGENT_DATA_DIR="${AGENT_DATA_DIR:-${APP_ROOT}/data}"

mkdir -p "${AGENT_DATA_DIR}"
cd "${APP_ROOT}"

exec "${APP_ROOT}/agent"
