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

echo "=== list on empty hub ==="
cd "$WORK/$PROJECT"
LIST_OUT="$("$ORBIT" list 2>&1)" || fail "orbit list on empty hub failed"
echo "$LIST_OUT" | grep -q "^$PROJECT  (" || fail "list header missing project + remote"
echo "$LIST_OUT" | grep -q "no worktrees yet" || fail "empty hint missing"
pass "list shows header + empty hint"

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

echo "=== list with multiple worktrees ==="
cd "$WORK/$PROJECT"
LIST_OUT="$("$ORBIT" list 2>&1)" || fail "orbit list failed"
echo "$LIST_OUT" | grep -q "^  another-branch " || fail "another-branch missing from list"
echo "$LIST_OUT" | grep -q "^  feature-orbit-smoke " || fail "feature-orbit-smoke missing from list"
echo "$LIST_OUT" | grep -q "^  master " || fail "master missing from list"
# Lines must be sorted alphabetically (skip the WORKTREE column header line)
LIST_NAMES="$(echo "$LIST_OUT" | awk '/^  [^(]/{print $1}' | grep -v '^WORKTREE$')"
SORTED_NAMES="$(echo "$LIST_NAMES" | sort)"
[[ "$LIST_NAMES" == "$SORTED_NAMES" ]] || fail "list output is not sorted: $LIST_NAMES"
pass "list shows all worktrees, sorted"

echo "=== list from a worktree subdir ==="
cd "$WORK/$PROJECT/master"
"$ORBIT" list >/dev/null 2>&1 || fail "orbit list from worktree subdir failed"
pass "list works from inside a worktree"

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

echo "=== rm by basename (keeps branch) ==="
cd "$WORK/$PROJECT"
"$ORBIT" rm another-branch >/dev/null 2>&1 || fail "orbit rm by basename failed"
[[ ! -e "$WORK/$PROJECT/another-branch" ]] || fail "rm did not remove dir"
git --git-dir="$XDG_STATE_HOME/orbit/repos/$PROJECT" show-ref --verify --quiet "refs/heads/another/branch" \
  || fail "branch should still exist"
pass "rm removes worktree, keeps branch"

echo "=== rm with --delete-branch ==="
"$ORBIT" rm feature-orbit-smoke --delete-branch >/dev/null 2>&1 || fail "orbit rm --delete-branch failed"
[[ ! -e "$WORK/$PROJECT/feature-orbit-smoke" ]] || fail "rm did not remove dir"
if git --git-dir="$XDG_STATE_HOME/orbit/repos/$PROJECT" show-ref --verify --quiet "refs/heads/feature/orbit-smoke"; then
  fail "branch should be deleted"
fi
pass "rm --delete-branch removes worktree and branch"

echo "=== rm by absolute path ==="
"$ORBIT" rm "$WORK/$PROJECT/master" >/dev/null 2>&1 || fail "orbit rm by abs path failed"
[[ ! -e "$WORK/$PROJECT/master" ]] || fail "rm did not remove dir"
pass "rm by absolute path"

echo "=== negative: rm of non-existent ==="
if "$ORBIT" rm nope >/dev/null 2>&1; then
  fail "expected failure on non-existent worktree"
fi
pass "fails clearly on unknown target"

echo "=== negative: rm outside a hub ==="
cd /tmp
if "$ORBIT" rm something >/dev/null 2>&1; then
  fail "expected failure outside hub"
fi
pass "fails clearly outside hub"

echo "=== negative: list outside a hub ==="
cd /tmp
if "$ORBIT" list >/dev/null 2>&1; then
  fail "expected failure outside hub"
fi
pass "list fails clearly outside a hub"

echo "=== negative: --delete-branch on bare HEAD branch ==="
cd "$WORK/$PROJECT"
"$ORBIT" new master >/dev/null 2>&1 || fail "could not re-create master worktree"
# Force bare HEAD to point to master so the preflight has something to refuse.
git --git-dir="$XDG_STATE_HOME/orbit/repos/$PROJECT" symbolic-ref HEAD refs/heads/master
if "$ORBIT" rm master --delete-branch >/dev/null 2>&1; then
  fail "expected failure deleting bare HEAD branch"
fi
[[ -e "$WORK/$PROJECT/master" ]] || fail "preflight should have aborted before removing worktree"
pass "refuses to delete bare HEAD branch (preflight aborts before removal)"
# cleanup the master worktree we just re-created (no --delete-branch this time)
"$ORBIT" rm master >/dev/null 2>&1 || fail "cleanup of master failed"

echo "=== migrate ==="
# Use a fresh state dir so we don't collide with the existing $PROJECT bare.
MIG_STATE="$TMP/mig-state"
MIG_PARENT="$TMP/mig-work"
mkdir -p "$MIG_STATE" "$MIG_PARENT"
git clone "$REPO_URL" "$MIG_PARENT/standalone" >/dev/null 2>&1 || fail "git clone of standalone failed"
[[ -d "$MIG_PARENT/standalone/.git" ]] || fail "standalone .git should be a directory before migrate"

cd "$MIG_PARENT/standalone"
XDG_STATE_HOME="$MIG_STATE" "$ORBIT" migrate >/dev/null 2>&1 || fail "orbit migrate failed"

[[ ! -e "$MIG_PARENT/standalone" ]]                       || fail "original dir should be renamed away"
ls -d "$MIG_PARENT"/standalone.orbit-backup-* >/dev/null 2>&1 || fail "backup dir not found"
[[ -f "$MIG_PARENT/.orbit.yaml" ]]                        || fail ".orbit.yaml not created in parent"
[[ -d "$MIG_STATE/orbit/repos/$PROJECT" ]]                || fail "bare not created at fresh state dir"
[[ -e "$MIG_PARENT/master/.git" ]]                        || fail "master worktree missing"
[[ -f "$MIG_PARENT/master/.git" ]]                        || fail "master/.git should be a gitlink file, not a dir"
pass "migrate: bare + hub + master worktree created, original moved to backup"

echo "=== negative: orbit migrate outside a git repo ==="
NON_GIT="$TMP/non-git"
mkdir -p "$NON_GIT"
cd "$NON_GIT"
if XDG_STATE_HOME="$MIG_STATE" "$ORBIT" migrate >/dev/null 2>&1; then
  fail "expected failure outside a git repo"
fi
pass "migrate fails outside a git repo"

echo "=== negative: orbit migrate from inside an orbit worktree ==="
cd "$WORK/$PROJECT"
"$ORBIT" new master >/dev/null 2>&1 || fail "could not create master worktree for migrate negative test"
cd "$WORK/$PROJECT/master"
[[ -f "$WORK/$PROJECT/master/.git" ]] || fail "expected .git to be a file (gitlink) inside an orbit worktree"
if "$ORBIT" migrate >/dev/null 2>&1; then
  fail "expected failure from inside an orbit worktree"
fi
pass "migrate fails when .git is a gitlink file (orbit worktree)"
# cleanup the master worktree we just re-created
cd "$WORK/$PROJECT"
"$ORBIT" rm master >/dev/null 2>&1 || fail "cleanup of master after migrate negative failed"

echo
echo "ALL SMOKE TESTS PASSED"
