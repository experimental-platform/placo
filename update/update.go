package update

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/experimental-platform/platconf/platconf"
)

type Opts struct {
	Channel string `short:"c" long:"channel" description:"Channel to be installed"`
	//Force bool `short:"f" long:"force" description:"Force installing the current latest release"`
}

func (o *Opts) Execute(args []string) error {
	os.Setenv("DOCKER_API_VERSION", "1.22")

	channel, channelSource := getChannel(o.Channel)
	switch channelSource {
	case csChannelFile:
		fmt.Printf("Using channel '%s' from the channel file.\n", channel)
		break
	case csCommandLine:
		fmt.Printf("Using channel '%s' from the command line.\n", channel)
		break
	case csDefault:
		fmt.Printf("Using channel '%s'(default).\n", channel)
		break
	}

	err := runUpdate()
	if err != nil {
		button(buttonError)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}

func runUpdate() error {
	platconf.RequireRoot()
	button(buttonRainbow)

	fmt.Println("Creating folders in '/etc/systemd' in case they don't exist yet.")
	err := setupPaths("/")
	if err != nil {
		return err
	}

	return nil
}

func setupPaths(rootPrefix string) error {
	requiredPaths := []string{
		"/etc/systemd/journal.conf.d",
		"/etc/systemd/system",
		"/etc/systemd/system/docker.service.d",
		"/etc/systemd/system/scripts",
		"/etc/udev/rules.d",
		"/opt/bin",
	}

	for _, p := range requiredPaths {
		err := os.MkdirAll(path.Join(rootPrefix, p), 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func fetchReleaseData(channel string) (*platconf.ReleaseManifestV2, error) {
	data, err := fetchReleaseJSON(channel)
	if err != nil {
		return nil, err
	}

	var manifest platconf.ReleaseManifestV2
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func fetchReleaseJSON(channel string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/protonet/builds/master/%s.json", channel)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Response status code was %d.", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
