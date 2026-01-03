package main

import (
	"os"

	"github.com/tharsanan1/ai-helper/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
