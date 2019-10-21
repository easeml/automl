package main

import (
	"log"

	"github.com/ds3lab/easeml/engine/command"
	"github.com/spf13/cobra/doc"
)

func main() {
	
	cmd := command.GetRootCommand()

	err := doc.GenMarkdownTree(cmd, ".")
	if err != nil {
		log.Fatal(err)
	}
}
