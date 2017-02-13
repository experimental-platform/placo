package update

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

	// TODO remove this later
	log.Fatal("The update functionality is not yet available.")

	err := runUpdate(o.Channel, "/")
	if err != nil {
		button(buttonError)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}

func runUpdate(specifiedChannel string, rootDir string) error {
	// prepare
	platconf.RequireRoot()
	button(buttonRainbow)
	setStatus("preparing", nil, nil)

	// get channel
	channel, channelSource := getChannel(specifiedChannel)
	logChannelDetection(channel, channelSource)

	// get release data
	releaseData, err := fetchReleaseData(channel)
	if err != nil {
		return err
	}

	// get & extract 'configure'
	configureImgData := releaseData.GetImageByName("quay.io/experimentalplatform/configure")
	if configureImgData == nil {
		return fmt.Errorf("configure image data not found in the manifest")
	}

	configureExtractDir, err := extractConfigure(configureImgData.Tag)
	if err != nil {
		return err
	}
	defer os.RemoveAll(configureExtractDir)

	// setup paths
	fmt.Println("Creating folders in '/etc/systemd' in case they don't exist yet.")
	err = setupPaths(rootDir)
	if err != nil {
		return err
	}

	return nil
}

func setupPaths(rootPrefix string) error {
	requiredPaths := []string{
		"/etc/protonet",
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
	url := fmt.Sprintf("https://raw.githubusercontent.com/protonet/builds/master/manifest-v2/%s.json", channel)
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

func extractConfigure(tag string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "platconf_")
	if err != nil {
		return "", err
	}

	log.Println("Pulling configure image")
	err = pullImage("quay.io/experimentalplatform/configure", tag)
	if err != nil {
		return "", err
	}

	log.Println("Extracting configure image")
	err = extractDockerImage("quay.io/experimentalplatform/configure", tag, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}
