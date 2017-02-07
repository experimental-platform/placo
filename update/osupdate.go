package update

import "os/exec"

// TODO perhaps make this a parallel thread in the future?

func performOSUpdate() error {
	cmd := exec.Command("/usr/bin/update_engine_client", "-update")
	return cmd.Run()
}
