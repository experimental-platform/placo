package main

import (
	"github.com/experimental-platform/platconf/update"
)

var opts struct {
	Update     update.Opts    `command:"update"`
	SelfUpdate selfupdateOpts `command:"selfupdate"`
	Version    versionOpts    `command:"version"`
}
