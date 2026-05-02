package main

import (
	"os"

	"github.com/leonnardo/orbit/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
