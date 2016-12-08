package main

import (
	"os"

	"github.com/jessevdk/go-flags"
)

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if flagserr, ok := err.(*flags.Error); !ok || flagserr.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
		}
		os.Exit(1)
	}
}
