# orbit

> Git worktree hub manager — one bare repo, one hub, many worktrees, no chaos.

`orbit` keeps each project as a single bare repo cached globally and a local **hub** directory where every branch you work on lives as a sibling worktree. No more `git clone` per branch, no more "which checkout was that again?".

```text
~/.local/state/orbit/repos/<project>     ← bare repo (cache)

<hub>/
  .orbit.yaml                            ← hub marker + config
  main/                                  ← worktree
  feature-login/                         ← worktree
  fix-bug/                               ← worktree
```

## Status

MVP complete. Five commands shipped: `clone`, `new`, `rm`, `list`, `migrate`. See [Roadmap](#roadmap) for what's next.

## Install

Requires Go 1.24+ and `git` on `PATH`.

```bash
git clone <this-repo> orbit
cd orbit/main           # if you cloned with orbit; otherwise just cd into the repo
make install            # installs to ~/.local/bin/orbit
```

Make sure `~/.local/bin` is on your `PATH`. To install elsewhere:

```bash
make install BIN_DIR=/usr/local/bin
```

## Usage

### Clone a project (creates a hub)

```bash
cd ~/workspace
orbit clone https://github.com/octocat/Hello-World
# bare cached at ~/.local/state/orbit/repos/Hello-World
# hub created at ~/workspace/Hello-World/.orbit.yaml
```

Pick a different local name:

```bash
orbit clone https://github.com/octocat/Hello-World hello
```

### Open or create a branch in a worktree

```bash
cd ~/workspace/Hello-World

orbit new master                  # tracks origin/master, worktree at ./master
orbit new feature/login           # creates new branch from origin/HEAD, worktree at ./feature-login
orbit new fix/bug worktrees/bug   # custom path inside the hub
```

`orbit new` is "create or open":

- branch exists locally → open it
- branch exists only on `origin` → create local tracking branch and open it
- branch doesn't exist anywhere → create new from `origin/HEAD`

You can run `orbit new` from anywhere inside the hub, including from inside an existing worktree — the hub is detected by walking up the tree looking for `.orbit.yaml`.

### Remove a worktree

```bash
cd ~/workspace/Hello-World
orbit rm feature-login                    # removes the worktree, keeps the branch
orbit rm feature-login --delete-branch    # removes the worktree AND deletes the branch (safe delete only)
```

Accepts an absolute path, a relative path, or just the worktree's basename.

### List worktrees

```bash
cd ~/workspace/Hello-World
orbit list
# Hello-World  (https://github.com/octocat/Hello-World)
#   WORKTREE        BRANCH          FOLDER
# * main            main            ~/workspace/Hello-World/main
#   feature-login   feature/login   ~/workspace/Hello-World/feature-login
```

The `*` marks the worktree containing your current directory.

### Migrate an existing standalone clone into a hub

```bash
cd ~/workspace/some-existing-clone        # a regular `git clone`-d repo
orbit migrate                             # adopts it into the orbit layout
```

The original directory is renamed to `<dir>.orbit-backup-<timestamp>` (kept for safety — you can `rm -rf` it after verifying), a new bare is created in `~/.local/state/orbit/repos/<project>`, and every local branch is recreated as a worktree in the parent. Requires a clean tree, no stash, and all local branches pushed to `origin`.

## Roadmap

Next features beyond the MVP, in rough priority order.

- **`migrate --adopt`** — handle clones that already have sibling worktrees (the manual hub-of-worktrees pattern).
- **`orbit rm --force`** — remove a dirty worktree (today fails if there are uncommitted changes).
- **`orbit doctor`** — sanity-check bare/hub/worktree consistency, report drift.
- **`orbit prune`** — clean up dead worktrees and stale merged branches.
- **`orbit rename`** — rename a project and/or its hub.

## How it works

- **Bare repo** in `~/.local/state/orbit/repos/<project>` is the single source of truth for refs and objects. Every worktree shares it.
- **Hub** is just a directory with `.orbit.yaml` at the root. It is not a Git repository — only the worktrees inside it are.
- **Hub config** (`.orbit.yaml`) records the project name, the original remote URL, and a creation timestamp. That's it.
- Worktrees are created with `git worktree add` against the bare repo, always inside the hub.

The bare repo is **not** a `--mirror` clone: that would overwrite local branches on every fetch. We do `git init --bare` plus a manual `fetch '+refs/heads/*:refs/remotes/origin/*'`, which keeps `refs/heads/*` (your worktree branches) separate from `refs/remotes/origin/*`.

## Development

```bash
make help    # list targets
make test    # run unit tests
make smoke   # end-to-end test against a public repo (uses an isolated XDG_STATE_HOME)
make fmt
make vet
```
