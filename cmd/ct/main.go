package main

import (
	"fmt"
	"os"

	"github.com/GIRIBUIN/ct/internal/command"
)

func main() {
	if err := command.Run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ct:", err)
		os.Exit(1)
	}
}
