package update

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChannelCommandLine(t *testing.T) {
	commandLineChannel := "foobar"
	result, method := getChannel(commandLineChannel)
	assert.Equal(t, commandLineChannel, result)
	assert.Equal(t, csCommandLine, method)
}

func TestGetChannelFromFile(t *testing.T) {
	channelFileChannel := "testchannel123"

	tempFile, err := ioutil.TempFile("", "")
	assert.Nil(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	tempFile.WriteString(channelFileChannel)
	tempFile.Sync()

	channelFilePath = tempFile.Name()

	result, method := getChannel("") // channel from command line is an empty string
	assert.Equal(t, channelFileChannel, result)
	assert.Equal(t, csChannelFile, method)
}

func TestGetChannelFromFileIsEmpty(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "")
	assert.Nil(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	tempFile.Sync()

	channelFilePath = tempFile.Name()

	result, method := getChannel("") // channel from command line is an empty string
	assert.Equal(t, defaultChannel, result)
	assert.Equal(t, csDefault, method)
}

func TestGetChannelFromFileDoesntExist(t *testing.T) {
	channelFilePath = "/this/file/should/not/exist.txt"

	result, method := getChannel("") // channel from command line is an empty string
	assert.Equal(t, defaultChannel, result)
	assert.Equal(t, csDefault, method)
}
