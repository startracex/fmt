package main

import (
	"os"

	"github.com/startracex/fmt/pkg/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
