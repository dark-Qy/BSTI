#!/usr/bin/env bash

set -euo pipefail

APP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export HTTP_ADDR="${HTTP_ADDR:-0.0.0.0:8787}"
export AGENT_DATA_DIR="${AGENT_DATA_DIR:-/tmp/byte-agent-data}"
export LARK_CLI_BIN="${LARK_CLI_BIN:-${APP_ROOT}/bin/lark-cli}"
export LLM_PROVIDER="${LLM_PROVIDER:-${CUSTOM_LLM_PROVIDER:-}}"
export LLM_API_URL="${LLM_API_URL:-${CUSTOM_LLM_API_URL:-}}"
export LLM_API_KEY="${LLM_API_KEY:-${CUSTOM_LLM_API_KEY:-}}"
export LLM_MODEL="${LLM_MODEL:-${CUSTOM_LLM_MODEL:-}}"
export LLM_MAX_TOKENS="${LLM_MAX_TOKENS:-${CUSTOM_LLM_MAX_TOKENS:-}}"

mkdir -p "${AGENT_DATA_DIR}"
cd "${APP_ROOT}"

exec "${APP_ROOT}/agent"
