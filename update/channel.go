package update

import "io/ioutil"

type channelSource int

const (
	csCommandLine channelSource = iota
	csChannelFile channelSource = iota
	csDefault     channelSource = iota
)

const defaultChannel string = "soul3"

var channelFilePath = "/etc/protonet/system/channel"

func getChannel(commandLineChannel string) (string, channelSource) {
	// If the channel has been specified on the command line then go with it
	if commandLineChannel != "" {
		return commandLineChannel, csCommandLine
	}

	// Now try the channel file
	channelFileData, err := ioutil.ReadFile(channelFilePath)
	if err != nil {
		// It doesn't exist
		return defaultChannel, csDefault
	}

	// File exists, but is empty
	if len(channelFileData) == 0 {
		return defaultChannel, csDefault
	}

	return string(channelFileData), csChannelFile
}
