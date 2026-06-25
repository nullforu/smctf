#!/usr/bin/env bash
# Sync the invite-bot/ subtree from the wargame repository.
#
# The canonical invite-bot source lives in the wargame repo under invite-bot/.
# This script re-splits that directory into a temporary branch and pulls it into
# this repo's invite-bot/ subtree, so changes made in wargame flow into smctf.
#
# Usage:
#   scripts/sync-invite-bot.sh
#   WARGAME_REPO=/path/to/wargame scripts/sync-invite-bot.sh
set -euo pipefail

WARGAME_REPO="${WARGAME_REPO:-/Users/projects/wargame}"
PREFIX="invite-bot"
SPLIT_BRANCH="invitebot-subtree"

if [ ! -d "${WARGAME_REPO}/.git" ]; then
  echo "wargame repo not found at ${WARGAME_REPO} (set WARGAME_REPO=...)" >&2
  exit 1
fi

# 1) (Re)create a split whose root is the invite-bot tree.
git -C "${WARGAME_REPO}" branch -D "${SPLIT_BRANCH}" >/dev/null 2>&1 || true
git -C "${WARGAME_REPO}" subtree split --prefix="${PREFIX}" -b "${SPLIT_BRANCH}"

# 2) Pull it into this repo's invite-bot/ subtree.
git subtree pull --prefix="${PREFIX}" "${WARGAME_REPO}" "${SPLIT_BRANCH}" \
  -m "chore: sync invite-bot from wargame"

# 3) Clean up the transient split branch.
git -C "${WARGAME_REPO}" branch -D "${SPLIT_BRANCH}" >/dev/null 2>&1 || true

echo "invite-bot synced from ${WARGAME_REPO}"
