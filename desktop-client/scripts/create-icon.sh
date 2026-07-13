#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
python_bin="${PYTHON:-python3}"
"$python_bin" "$root/scripts/create-iconset.py"
