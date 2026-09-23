#!/bin/sh
# Fork-only helper: sync this fork with the official MagiTrickle repository.
#
# Usage:
#   scripts/sync-upstream.sh            # fetch upstream develop, update local `upstream`, merge into current branch
#   scripts/sync-upstream.sh --no-merge # only fetch and update the `upstream` mirror branch
#   UPSTREAM_REF=main scripts/sync-upstream.sh   # sync from the official release branch instead
#
# Environment:
#   UPSTREAM_URL   git URL of the official repo (default: GitLab)
#   UPSTREAM_REF   upstream branch to sync from (default: develop)
#   MIRROR_BRANCH  local branch that mirrors upstream (default: upstream)
set -eu

UPSTREAM_URL="${UPSTREAM_URL:-https://gitlab.com/magitrickle/magitrickle.git}"
UPSTREAM_REF="${UPSTREAM_REF:-develop}"
MIRROR_BRANCH="${MIRROR_BRANCH:-upstream}"
MERGE=1
[ "${1:-}" = "--no-merge" ] && MERGE=0

cd "$(git rev-parse --show-toplevel)"

if ! git remote get-url upstream >/dev/null 2>&1; then
  echo ">> adding remote 'upstream' -> $UPSTREAM_URL"
  git remote add upstream "$UPSTREAM_URL"
fi

echo ">> fetching upstream/$UPSTREAM_REF and tags"
git fetch --tags upstream "$UPSTREAM_REF"

echo ">> updating mirror branch '$MIRROR_BRANCH' -> upstream/$UPSTREAM_REF"
git branch -f --no-track "$MIRROR_BRANCH" "upstream/$UPSTREAM_REF"

if [ "$MERGE" -eq 0 ]; then
  echo ">> done (no merge). Push the mirror with: git push origin $MIRROR_BRANCH --tags"
  exit 0
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "!! working tree is not clean, commit or stash your changes first" >&2
  exit 1
fi

current="$(git rev-parse --abbrev-ref HEAD)"
if [ "$current" = "$MIRROR_BRANCH" ]; then
  echo "!! you are on the mirror branch '$MIRROR_BRANCH'; switch to the fork branch (e.g. develop) first" >&2
  exit 1
fi

if git merge-base --is-ancestor "upstream/$UPSTREAM_REF" HEAD; then
  echo ">> '$current' already contains upstream/$UPSTREAM_REF, nothing to merge"
  exit 0
fi

echo ">> merging upstream/$UPSTREAM_REF into '$current'"
if git merge --no-ff --no-edit "upstream/$UPSTREAM_REF" \
     -m "Merge upstream $UPSTREAM_REF ($(git rev-parse --short "upstream/$UPSTREAM_REF")) into $current"; then
  echo ">> merged. Review, then: git push origin $current $MIRROR_BRANCH --tags"
else
  echo "!! merge conflicts. Resolve them, then 'git commit' and push." >&2
  echo "   (or 'git merge --abort' to cancel)" >&2
  exit 1
fi
