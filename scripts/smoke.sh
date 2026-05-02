#!/usr/bin/env bash
# orbit smoke test — exercises clone + new end-to-end against a real public repo.
#
# Uses an isolated XDG_STATE_HOME so it never touches your real ~/.local/state/orbit.
# Run from the project root:
#   ./scripts/smoke.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ORBIT="${ORBIT:-$ROOT/orbit}"

if [[ ! -x "$ORBIT" ]]; then
  echo "building orbit..."
  (cd "$ROOT" && go build -o orbit .)
fi

TMP="$(mktemp -d -t orbit-smoke.XXXXXX)"
export XDG_STATE_HOME="$TMP/state"
WORK="$TMP/work"
mkdir -p "$WORK"

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

REPO_URL="${REPO_URL:-https://github.com/octocat/Hello-World}"
PROJECT="$(basename "${REPO_URL%.git}")"

pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ $*"; exit 1; }

echo "=== clone ==="
cd "$WORK"
"$ORBIT" clone "$REPO_URL" >/dev/null 2>&1 || fail "clone failed"
[[ -d "$XDG_STATE_HOME/orbit/repos/$PROJECT" ]] || fail "bare repo not created"
[[ -f "$WORK/$PROJECT/.orbit.yaml" ]]           || fail ".orbit.yaml not created"
[[ ! -e "$WORK/$PROJECT/.git" ]]                || fail "hub should NOT contain .git"
pass "bare + hub created, no .git in hub"

echo "=== new (tracking remote branch) ==="
cd "$WORK/$PROJECT"
"$ORBIT" new master >/dev/null 2>&1 || fail "orbit new master failed"
[[ -e "$WORK/$PROJECT/master/.git" ]] || fail "worktree master/.git missing"
pass "tracking branch worktree created"

echo "=== new (brand new branch) ==="
"$ORBIT" new feature/orbit-smoke >/dev/null 2>&1 || fail "orbit new feature/orbit-smoke failed"
[[ -e "$WORK/$PROJECT/feature-orbit-smoke/.git" ]] || fail "slug worktree missing"
pass "new branch + slug applied"

echo "=== new from inside a worktree subdir ==="
cd "$WORK/$PROJECT/master"
"$ORBIT" new another/branch >/dev/null 2>&1 || fail "orbit new from subdir failed"
[[ -e "$WORK/$PROJECT/another-branch/.git" ]] || fail "subdir-detected hub failed"
pass "hub detection works from worktree subdir"

echo "=== negative: orbit new outside a hub ==="
cd /tmp
if "$ORBIT" new foo >/dev/null 2>&1; then
  fail "expected failure outside hub"
fi
pass "fails clearly outside a hub"

echo "=== negative: orbit clone with existing project ==="
cd "$WORK"
if "$ORBIT" clone "$REPO_URL" >/dev/null 2>&1; then
  fail "expected failure on existing project"
fi
pass "fails clearly on existing project"

echo "=== negative: orbit new with branch already checked out ==="
cd "$WORK/$PROJECT"
if "$ORBIT" new master >/dev/null 2>&1; then
  fail "expected failure when branch already checked out"
fi
pass "fails clearly when branch is checked out elsewhere"

echo "=== negative: orbit new with escaping path ==="
if "$ORBIT" new escape ../escape >/dev/null 2>&1; then
  fail "expected failure on escaping path"
fi
pass "rejects path outside hub"

echo
echo "ALL SMOKE TESTS PASSED"
