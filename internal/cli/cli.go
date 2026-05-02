// Package cli dispatches subcommands.
package cli

import (
	"fmt"
	"os"

	"github.com/leonnardo/orbit/internal/commands"
)

const usage = `orbit — git worktree hub manager

usage:
  orbit clone <repo-url-or-path> [project]
  orbit new   <branch> [path]
  orbit rm    <path-or-name> [--delete-branch]
  orbit list
`

func Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	cmd, rest := args[0], args[1:]

	switch cmd {
	case "clone":
		return runCmd(commands.Clone, rest)
	case "new":
		return runCmd(commands.New, rest)
	case "rm":
		return runCmd(commands.Rm, rest)
	case "list":
		fmt.Fprintln(os.Stderr, "orbit list: not implemented yet")
		return 1
	case "-h", "--help", "help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "orbit: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}

func runCmd(fn func([]string) error, args []string) int {
	if err := fn(args); err != nil {
		fmt.Fprintf(os.Stderr, "orbit: %s\n", err)
		return 1
	}
	return 0
}
