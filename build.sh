#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${ROOT_DIR}/output"
BINARY_NAME="${BINARY_NAME:-agent}"
TARGET_OS="${GOOS:-linux}"
TARGET_ARCH="${GOARCH:-amd64}"

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}/data"

GOCACHE="${ROOT_DIR}/.gocache" \
GOOS="${TARGET_OS}" \
GOARCH="${TARGET_ARCH}" \
CGO_ENABLED="${CGO_ENABLED:-0}" \
go build -o "${OUTPUT_DIR}/${BINARY_NAME}" ./cmd/agent

cp "${ROOT_DIR}/bootstrap.sh" "${OUTPUT_DIR}/bootstrap.sh"
cp -R "${ROOT_DIR}/photos" "${OUTPUT_DIR}/photos"
chmod +x "${OUTPUT_DIR}/${BINARY_NAME}" "${OUTPUT_DIR}/bootstrap.sh"
