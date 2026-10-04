package main

import (
	"fmt"
	"os"

	"solo-0014-zone-rollout-control/internal/workflowprobe"
)

func main() {
	if err := workflowprobe.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
