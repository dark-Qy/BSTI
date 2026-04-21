#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

cp "${ROOT_DIR}/bootstrap.sh" "${TMP_DIR}/bootstrap.sh"
chmod +x "${TMP_DIR}/bootstrap.sh"

cat > "${TMP_DIR}/agent" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf 'LLM_PROVIDER=%s\n' "${LLM_PROVIDER:-}"
printf 'LLM_API_URL=%s\n' "${LLM_API_URL:-}"
printf 'LLM_API_KEY=%s\n' "${LLM_API_KEY:-}"
printf 'LLM_MODEL=%s\n' "${LLM_MODEL:-}"
printf 'LLM_MAX_TOKENS=%s\n' "${LLM_MAX_TOKENS:-}"
printf 'AGENT_DATA_DIR=%s\n' "${AGENT_DATA_DIR:-}"
EOF
chmod +x "${TMP_DIR}/agent"

OUTPUT="$(
  CUSTOM_LLM_PROVIDER="kimi" \
  CUSTOM_LLM_API_URL="https://api.moonshot.cn/v1/chat/completions" \
  CUSTOM_LLM_API_KEY="secret-key" \
  CUSTOM_LLM_MODEL="kimi-k2.5" \
  CUSTOM_LLM_MAX_TOKENS="6000" \
  "${TMP_DIR}/bootstrap.sh"
)"

printf '%s\n' "${OUTPUT}" | grep -Fx 'LLM_PROVIDER=kimi'
printf '%s\n' "${OUTPUT}" | grep -Fx 'LLM_API_URL=https://api.moonshot.cn/v1/chat/completions'
printf '%s\n' "${OUTPUT}" | grep -Fx 'LLM_API_KEY=secret-key'
printf '%s\n' "${OUTPUT}" | grep -Fx 'LLM_MODEL=kimi-k2.5'
printf '%s\n' "${OUTPUT}" | grep -Fx 'LLM_MAX_TOKENS=6000'
printf '%s\n' "${OUTPUT}" | grep -Fx 'AGENT_DATA_DIR=/tmp/byte-agent-data'
