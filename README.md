# orbit

> Git worktree hub manager — one bare repo, one hub, many worktrees, no chaos.

`orbit` keeps each project as a single bare repo cached globally and a local **hub** directory where every branch you work on lives as a sibling worktree. No more `git clone` per branch, no more "which checkout was that again?".

## How it works

- **Bare repo** in `~/.local/state/orbit/repos/<project>` is the single source of truth for refs and objects. Every worktree shares it.
- **Hub** is a directory with `.orbit.yaml` at the root.
- **Hub config** (`.orbit.yaml`) records the project name, the original remote URL, and creation timestamp.
- Worktrees are created with `git worktree add` against the bare repo, always inside the hub.

```text
~/.local/state/orbit/repos/<project>     ← bare repo (cache)

<hub>/
  .orbit.yaml                            ← hub marker + config
  main/                                  ← worktree
  feature-login/                         ← worktree
  fix-bug/                               ← worktree
```


## Install

Requires Go 1.24+ and `git` on `PATH`.

Install the latest published version directly from GitHub:

```bash
go install github.com/leonnardo/orbit@latest
```

Make sure Go's install directory is on your `PATH`. If `GOBIN` is set, Go
installs `orbit` there. Otherwise it installs to `$(go env GOPATH)/bin`.

You can check where Go will install binaries with:

```bash
go env GOBIN GOPATH
```

To install from a local checkout instead:

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
#   BRANCH         STATUS  PATH           COMMIT   MESSAGE
# * main           ✓       main           a1b2c3d  Initial commit
#   feature/login  ●↑      feature-login  d4e5f6a  Add login flow
```

The `*` marks the worktree containing your current directory. It only appears
when `orbit list` is run from inside a worktree; running it from the hub root
prints the same table without a current-worktree marker.

Status signals:

- `✓` clean
- `●` local changes
- `?` untracked files
- `!` conflicts
- `↑` ahead of upstream
- `↓` behind upstream
- `◆` detached HEAD
- `🔒` locked worktree

### Migrate an existing standalone clone into a hub

```bash
cd ~/workspace/some-existing-clone        # a regular `git clone`-d repo
orbit migrate                             # adopts it into the orbit layout
```

The original directory is renamed to `<dir>.orbit-backup-<timestamp>` (kept for safety — you can `rm -rf` it after verifying), a new bare is created in `~/.local/state/orbit/repos/<project>`, and every local branch is recreated as a worktree in the parent. Requires a clean tree, no stash, and all local branches pushed to `origin`.

## Development

```bash
make help    # list targets
make test    # run unit tests
make smoke   # end-to-end test against a public repo (uses an isolated XDG_STATE_HOME)
make fmt
make vet
```

## Roadmap

Next features beyond the MVP, in rough priority order.

- [ ] **`migrate --adopt`** — handle clones that already have sibling worktrees (the manual hub-of-worktrees pattern).
- [ ] **`orbit rm --force`** — remove a dirty worktree (today fails if there are uncommitted changes).
- [ ] **`orbit doctor`** — sanity-check bare/hub/worktree consistency, report drift.
- [ ] **`orbit prune`** — clean up dead worktrees and stale merged branches.
- [ ] **`orbit rename`** — rename a project and/or its hub.

## License

No license declared yet. Treat as personal-use only until one is added.

## Feedback

Issues and PRs are welcome.

