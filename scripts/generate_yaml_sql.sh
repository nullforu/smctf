#!/usr/bin/env bash
set -euo pipefail

python3 ./scripts/generate_yaml_sql/main.py "$@"
