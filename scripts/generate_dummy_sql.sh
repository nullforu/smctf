#!/bin/bash

set -euo pipefail

python3 ./scripts/generate_dummy_sql/main.py "$@"
