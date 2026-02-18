#!/bin/bash

set -euo pipefail

python3 ./scripts/coverage_check/main.py "$@"
