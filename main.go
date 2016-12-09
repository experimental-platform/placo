package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/jessevdk/go-flags"
)

func requireRoot() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("requireRoot(): %s\n", err.Error())
		os.Exit(1)
	}

	if currentUser.Uid != "0" {
		fmt.Println("ROOT access is required for this operation.")
		os.Exit(1)
	}
}

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
