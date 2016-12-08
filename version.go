package main

import "fmt"

var (
	VersionTag string
)

type versionOpts struct {
}

func (o *versionOpts) Execute(args []string) error {
	if VersionTag == "" {
		fmt.Println("No version information available. This is a development binary.")
	} else {
		fmt.Printf("Platform Configurator release %s\n", VersionTag)
	}

	return nil
}
