#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${ROOT_DIR}/output"
BINARY_NAME="${BINARY_NAME:-agent}"
TARGET_OS="${GOOS:-linux}"
TARGET_ARCH="${GOARCH:-amd64}"
CLI_VENDOR_DIR="${OUTPUT_DIR}/vendor/lark-cli"
CLI_BIN_DIR="${OUTPUT_DIR}/bin"
CLI_PACKAGE="${LARK_CLI_NPM_PACKAGE:-@larksuite/cli}"

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}/data" "${CLI_VENDOR_DIR}" "${CLI_BIN_DIR}"

NPM_CONFIG_CACHE="${ROOT_DIR}/.npm-cache" npm --prefix "${ROOT_DIR}/web" ci
NPM_CONFIG_CACHE="${ROOT_DIR}/.npm-cache" npm --prefix "${ROOT_DIR}/web" run build
NPM_CONFIG_CACHE="${ROOT_DIR}/.npm-cache" npm install -g --prefix "${CLI_VENDOR_DIR}" "${CLI_PACKAGE}"

GOCACHE="${ROOT_DIR}/.gocache" \
GOOS="${TARGET_OS}" \
GOARCH="${TARGET_ARCH}" \
CGO_ENABLED="${CGO_ENABLED:-0}" \
go build -o "${OUTPUT_DIR}/${BINARY_NAME}" ./cmd/agent

cp "${ROOT_DIR}/bootstrap.sh" "${OUTPUT_DIR}/bootstrap.sh"
cp "${ROOT_DIR}/run.sh" "${OUTPUT_DIR}/run.sh"
cat > "${CLI_BIN_DIR}/lark-cli" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_ROOT="${SCRIPT_DIR}/../vendor/lark-cli"

exec "${CLI_ROOT}/bin/lark-cli" "$@"
EOF
chmod +x "${OUTPUT_DIR}/${BINARY_NAME}" "${OUTPUT_DIR}/bootstrap.sh" "${OUTPUT_DIR}/run.sh" "${CLI_BIN_DIR}/lark-cli"
