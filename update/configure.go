package update

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/experimental-platform/platconf/platconf"
)

// ErrIsRelative is an error to be returned by functions that require
// an absolute path when a relative path has been passed to them
var ErrIsRelative = errors.New("a relative path was given")

func copyFile(dst, src string, mode os.FileMode) error {
	// only use absolute paths to prevent epic fails
	if !path.IsAbs(dst) || !path.IsAbs(src) {
		return fmt.Errorf("copyFile must use absolute paths")
	}

	// does source exist and is it a regular file?
	srcFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	// open source
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// open destination
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	defer dstFile.Sync()

	// copy
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func setupUtilityScripts(rootDir, configureDir string) error {
	binDir := path.Join(rootDir, "opt", "bin")
	scriptsDir := path.Join(rootDir, "etc", "systemd", "system", "scripts")
	ignoreScripts := []string{
		"protonet_zpool.sh",
		"platconf",
	}

	binDirContents, err := ioutil.ReadDir(binDir)
	if err != nil {
		return err
	}

	// remove contents of /opt/bin/
	// TODO don't remove and then copy
	// instead copy to another dir and then replace the directories
	// so the operation is more atomic
removeBindirContents:
	for _, f := range binDirContents {
		fullpath := path.Join(binDir, f.Name())
		basename := f.Name()

		// should we leave this one behind?
		for _, toSkip := range ignoreScripts {
			if basename == toSkip {
				log.Println("setupUtilityScripts: skipping", toSkip)
				continue removeBindirContents
			}
		}

		if !f.IsDir() {
			log.Println("Removing old", basename)
			err = os.Remove(fullpath)
			if err != nil {
				return fmt.Errorf("Failed to remove '%s': %s", fullpath, err.Error())
			}
		}
	}

	// remove contents of /etc/systemd/system/scripts/
	scriptsDirContents, err := ioutil.ReadDir(scriptsDir)
	if err != nil {
		return err
	}

	for _, f := range scriptsDirContents {
		fullpath := path.Join(scriptsDir, f.Name())
		basename := f.Name()
		if f.Mode().IsRegular() {
			log.Println("Removing old", basename)
			err = os.Remove(fullpath)
			if err != nil {
				return err
			}
		}
	}

	// install new scripts
	log.Println("Installing new scripts")
	newScriptsContents, err := ioutil.ReadDir(path.Join(configureDir, "scripts"))
	if err != nil {
		return err
	}
	for _, f := range newScriptsContents {
		fullpath := path.Join(configureDir, "scripts", f.Name())
		basename := f.Name()
		dst := path.Join(scriptsDir, basename)
		linkLocation := strings.TrimSuffix(path.Join(binDir, basename), ".sh")
		log.Println("\t", "*", basename)
		err = copyFile(dst, fullpath, 0755)
		if err != nil {
			return fmt.Errorf("setupUtilityScripts: failed to copy file: %s", err.Error())
		}
		err = os.Symlink(dst, linkLocation)
		if err != nil {
			return fmt.Errorf("setupUtilityScripts: failed to symlink: %s", err.Error())
		}
	}

	log.Println("Done.")
	return nil
}

func setupBinaries(rootDir, configureDir string) error {
	binDir := path.Join(rootDir, "opt", "bin")

	uniqueBinaries := []string{
		"button",
		"tcpdump",
		"speedtest",
		"masterpassword",
		"ipmitool",
		"self_destruct",
	}

	// unique binaries first
	for _, b := range uniqueBinaries {
		if b == "platconf" {
			log.Printf("WARNING: setupBinaries tried to overwrite platconf with '%s'", path.Join(configureDir, "platconf"))
			continue
		}
		src := path.Join(configureDir, b)
		dst := path.Join(binDir, b)
		err := copyFile(dst, src, 0755)
		if err != nil {
			return err
		}
	}

	// now the Go binaries
	goBinaries, err := ioutil.ReadDir(path.Join(configureDir, "binaries"))
	if err != nil {
		return err
	}

	for _, b := range goBinaries {
		if b.Name() == "platconf" {
			log.Printf("WARNING: setupBinaries tried to overwrite platconf with '%s'", path.Join(configureDir, "binaries/platconf"))
			continue
		}
		dst := path.Join(binDir, b.Name())
		src := path.Join(configureDir, "binaries", b.Name())
		err := copyFile(dst, src, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func pullAllImages(manifest *platconf.ReleaseManifestV2, maxPullers int) error {
	// TODO add retry

	type pullerMsg struct {
		ImgName string
		Error   error
	}

	imagesTotal := len(manifest.Images)
	imagesChan := make(chan platconf.ReleaseManifestV2Image)
	pullerChan := make(chan pullerMsg)

	for i := 0; i < maxPullers; i++ {
		go func() {
			for {
				img, ok := <-imagesChan
				if !ok {
					return
				}

				err := pullImage(img.Name, img.Tag, nil)
				pullerChan <- pullerMsg{
					ImgName: img.Name,
					Error:   err,
				}
			}
		}()
	}

	go func() {
		for _, img := range manifest.Images {
			imagesChan <- img
		}
	}()

	for i := 0; i < imagesTotal; i++ {
		msg := <-pullerChan
		if msg.Error != nil {
			log.Printf("Downloading '%s': FAILED", msg.ImgName)
			log.Printf("Downloading '%s': %s", msg.ImgName, msg.Error.Error())
			return msg.Error
		}

		log.Printf("Downloading '%s': OK", msg.ImgName)
	}

	return nil
}

func parseAllTemplates(rootDir, configureDir string, manifest *platconf.ReleaseManifestV2) error {
	servicesDir := path.Join(configureDir, "services")

	files, err := ioutil.ReadDir(servicesDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.Mode().IsRegular() {
			return fmt.Errorf("parseAllTemplates: file '%s' is not a regular file", f.Name())
		}

		unitPath := path.Join(servicesDir, f.Name())
		err = parseTemplate(unitPath, manifest)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseTemplate(path string, manifest *platconf.ReleaseManifestV2) error {
	imgRegexp := regexp.MustCompile(`quay.io/[a-z]*/[a-z0-9\-]*`)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	match := imgRegexp.FindString(string(data))
	if match == "" {
		return nil
	}

	imageManifest := manifest.GetImageByName(match)
	if imageManifest == nil {
		return fmt.Errorf("parseTemplate: image '%s' is not in the manifest", match)
	}

	tagRegexp := regexp.MustCompile(`{{tag}}`)
	result := tagRegexp.ReplaceAll(data, []byte(imageManifest.Tag))

	err = ioutil.WriteFile(path, result, 0644)
	if err != nil {
		return err
	}

	return nil
}

func isBrokenLink(nodePath string) (bool, error) {
	if !path.IsAbs(nodePath) {
		return false, ErrIsRelative
	}

	fileinfo, err := os.Lstat(nodePath)
	if err != nil {
		return false, err
	}

	// is it a symlink?
	if fileinfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		dest, err := os.Readlink(nodePath)
		if err != nil {
			return false, err
		}

		var destFullPath string
		if path.IsAbs(dest) {
			destFullPath = dest
		} else {
			destFullPath = path.Join(path.Dir(nodePath), dest)
		}

		_, err = os.Lstat(destFullPath)
		return err != nil, nil
	}

	return false, nil
}

func removeBrokenLinks(dir string) error {
	if !path.IsAbs(dir) {
		return ErrIsRelative
	}

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range entries {
		fullPath := path.Join(dir, f.Name())
		broken, err := isBrokenLink(fullPath)
		if err != nil {
			return err
		}

		if broken {
			os.Remove(fullPath)
		}
	}

	return nil
}

func isPlatformUnit(unitPath string) (bool, error) {
	if !path.IsAbs(unitPath) {
		return false, ErrIsRelative
	}

	stat, err := os.Lstat(unitPath)
	if err != nil {
		return false, err
	}
	if !stat.Mode().IsRegular() {
		return false, nil
	}

	f, err := os.Open(unitPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// is empty
	if !scanner.Scan() {
		return false, nil
	}

	// check for prefix instead of full line match in case of trailing spaces, etc.
	return strings.HasPrefix(scanner.Text(), "# ExperimentalPlatform"), nil
}

func removePlatformUnits(dir string) error {
	if !path.IsAbs(dir) {
		return ErrIsRelative
	}

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range entries {
		fullPath := path.Join(dir, f.Name())
		isPlatform, err := isPlatformUnit(fullPath)
		if err != nil {
			return err
		}

		if isPlatform {
			os.Remove(fullPath)
		}
	}

	return nil
}

func cleanupSystemd(rootDir string) error {
	systemDir := path.Join(rootDir, "etc/systemd/system")
	networkDir := path.Join(rootDir, "etc/systemd/network")

	log.Printf("Cleaning up '%s'\n", systemDir)

	// First remove broken links, this should avoid confusing error messages
	err := removeBrokenLinks(systemDir)
	if err != nil {
		return err
	}

	err = removePlatformUnits(systemDir)
	if err != nil {
		return err
	}

	// do it again to remove garbage
	err = removeBrokenLinks(systemDir)
	if err != nil {
		return err
	}

	// remove network config files
	err = removePlatformUnits(networkDir)
	if err != nil {
		return err
	}

	log.Println("DONE.")

	return nil
}

func setupUdev(rootDir, configureDir string) error {
	log.Println("Setting up udev rules")
	src := path.Join(configureDir, "config", "80-protonet.rules")
	dst := path.Join(rootDir, "etc/udev/rules.d", "80-protonet.rules")

	err := copyFile(dst, src, 0644)
	if err != nil {
		return err
	}

	// TODO don't restart udev if file wasn't changed
	cmd := exec.Command("/usr/bin/udevadm", "control", "--reload-rules")
	cmd.Run()

	return nil
}

func setupSystemD(rootDir, configureDir string) error {
	log.Println("Setting up systemD services")

	// copy normal units
	serviceFiles, err := ioutil.ReadDir(path.Join(configureDir, "services"))
	if err != nil {
		return err
	}

	for _, sf := range serviceFiles {
		src := path.Join(configureDir, "services", sf.Name())
		dst := path.Join(rootDir, "etc/systemd/system", sf.Name())
		err = copyFile(dst, src, 0644)
		if err != nil {
			return err
		}
	}

	// copy docker log override
	src := path.Join(configureDir, "config/50-log-warn.conf")
	dst := path.Join(rootDir, "etc/systemd/system/docker.service.d/50-log-warn.conf")
	err = copyFile(dst, src, 0644)
	if err != nil {
		return err
	}

	// copy journalD config
	src = path.Join(configureDir, "config/journald_protonet.conf")
	dst = path.Join(rootDir, "etc/systemd/journald.conf.d/journald_protonet.conf")
	err = copyFile(dst, src, 0644)
	if err != nil {
		return err
	}

	// copy klog config
	src = path.Join(configureDir, "config/sysctl-klog.conf")
	dst = path.Join(rootDir, "etc/sysctl.d/sysctl-klog.conf")
	err = copyFile(dst, src, 0644)
	if err != nil {
		return err
	}

	// copy network config files
	networkFiles, err := ioutil.ReadDir(path.Join(configureDir, "config"))
	if err != nil {
		return err
	}

	for _, sf := range networkFiles {
		if strings.HasSuffix(sf.Name(), ".network") {
			src = path.Join(configureDir, "config", sf.Name())
			dst = path.Join(rootDir, "etc/systemd/network", sf.Name())
			err = copyFile(dst, src, 0644)
			if err != nil {
				return err
			}
		}
	}

	// reload all the things
	log.Println("Reloading the config files.")
	err = systemdDaemonReload()
	if err != nil {
		return err
	}

	// enable the systemd-networkd-wait-online.service
	err = systemdEnableUnits([]string{"systemd-networkd-wait-online.service"})
	if err != nil {
		return err
	}

	// enable everything
	log.Println("Enabling all config files")
	units, err := ioutil.ReadDir(path.Join(rootDir, "etc/systemd/system"))
	if err != nil {
		return err
	}

	// TODO maybe do this in one go?
	for _, u := range units {
		if !strings.HasSuffix(u.Name(), ".sh") && u.Mode().IsRegular() {
			err = systemdEnableUnits([]string{u.Name()})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func setupChannelFile(channelFilePath, channel string) error {
	currentChannel, err := ioutil.ReadFile(channelFilePath)
	if err == nil && string(currentChannel) == channel {
		return nil
	}

	err = systemdStopUnit("trigger-update-protonet.path")
	if err != nil {
		return err
	}
	defer systemdRestartUnit("trigger-update-protonet.path")

	return ioutil.WriteFile(channelFilePath, []byte(channel), 0644)
}

func finalize(manifest *platconf.ReleaseManifestV2, rootDir string) error {
	err := ioutil.WriteFile(path.Join(rootDir, "etc/protonet/system/release_number"), []byte(fmt.Sprintf("%d", manifest.Build)), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(rootDir, "etc/protonet/system/codename"), []byte(manifest.Codename), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(rootDir, "etc/protonet/system/release_notes_url"), []byte(manifest.ReleaseNotesURL), 0644)
	if err != nil {
		return err
	}

	return nil
}

func removeOldImages() error {
	// TODO

	return nil
}
