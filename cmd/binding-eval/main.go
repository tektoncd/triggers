package main

import (
	"fmt"
	"os"

	"github.com/tektoncd/triggers/cmd/binding-eval/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
