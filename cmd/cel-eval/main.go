package main

import (
	"log"
	"os"

	"github.com/tektoncd/triggers/cmd/cel-eval/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
