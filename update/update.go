package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/experimental-platform/platconf/platconf"
)

// Opts contains command line parameters for the 'update' command
type Opts struct {
	Channel string `short:"c" long:"channel" description:"Channel to be installed"`
	Pullers int    `short:"p" long:"pullers" description:"Maximum images being pulled at once" default:"4"`
	//Force bool `short:"f" long:"force" description:"Force installing the current latest release"`
}

// Execute is the function ran when the 'update' command is used
func (o *Opts) Execute(args []string) error {
	os.Setenv("DOCKER_API_VERSION", "1.22")

	if o.Pullers < 1 {
		return errors.New("The maximum number of pullers must be > 0")
	}

	err := runUpdate(o.Channel, "/", o.Pullers)
	if err != nil {
		button(buttonError)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}

func runUpdate(specifiedChannel string, rootDir string, maxPullers int) error {
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

	// setup default hostname
	hostameFilePath := path.Join(rootDir, "/etc/protonet/hostname")
	if _, err = os.Stat(hostameFilePath); os.IsNotExist(err) {
		ioutil.WriteFile(hostameFilePath, []byte("protonet"), 0644)
	}

	err = performOSUpdate()
	if err != nil {
		// we also get an error on a "no update" result, so this is fine
		log.Println("update-engine returned error:", err.Error())
	}

	// TODO run configure
	err = setupUtilityScripts(rootDir, configureExtractDir)
	if err != nil {
		return err
	}

	// END TODO run configure
	err = pullAllImages(releaseData, maxPullers)
	if err != nil {
		return err
	}

	err = parseAllTemplates(rootDir, configureExtractDir, releaseData)
	if err != nil {
		return err
	}

	err = cleanupSystemd(rootDir)
	if err != nil {
		return err
	}

	err = setupUdev(rootDir, configureExtractDir)
	if err != nil {
		return err
	}

	err = setupSystemD(rootDir, configureExtractDir)
	if err != nil {
		return err
	}

	err = setupChannelFile(path.Join(rootDir, "etc/protonet/system/channel"), channel)
	if err != nil {
		return err
	}

	setStatus("finalizing", nil, nil)

	err = finalize(releaseData, rootDir)
	if err != nil {
		return err
	}

	setStatus("done", nil, nil)

	// TODO allow to skip the reboot
	log.Println("Triggering a reboot")
	rebootCmd := exec.Command("/usr/sbin/shutdown", "--reboot", "1")
	rebootCmd.Run()

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

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return nil, fmt.Errorf("no such channel: '%s'", channel)
	default:
		return nil, fmt.Errorf("response status code was %d", resp.StatusCode)
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
	err = pullImage("quay.io/experimentalplatform/configure", tag, nil)
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
