#!/usr/bin/env bash

set -euo pipefail

APP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -d "${APP_ROOT}/output" ]; then
	exec "${APP_ROOT}/output/bootstrap.sh"
fi

exec "${APP_ROOT}/bootstrap.sh"
