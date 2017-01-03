package platconf

import (
	"fmt"
	"os"
	"os/user"
)

// RequireRoot checks whether we are currently runnning as root.
// If not, it will immediately bail.
func RequireRoot() {
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
