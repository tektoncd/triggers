package main

import (
	"fmt"
	"os"

	"github.com/tektoncd/triggers/cmd/cel-eval/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
