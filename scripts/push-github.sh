#!/usr/bin/env bash
set -euo pipefail

message="${1:-update music bot}"

git status --short
git add -A

if git diff --cached --quiet; then
  echo "No changes to commit."
  exit 0
fi

git commit -m "$message"
git push
