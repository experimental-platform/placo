package main

var opts struct {
	Update struct {
	} `command:"update"`
	SelfUpdate selfupdateOpts `command:"selfupdate"`
	Version    versionOpts    `command:"version"`
}
