package main

import (
	"github.com/shhac/agent-stripe/internal/cli"
)

var version = "dev"

func main() {
	cli.Execute(version)
}
