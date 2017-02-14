package update

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
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

func pullAllImages(manifest *platconf.ReleaseManifestV2) error {
	// TODO add retry

	type pullerMsg struct {
		ImgName string
		Error   error
	}

	imagesTotal := len(manifest.Images)
	imagesChan := make(chan platconf.ReleaseManifestV2Image)
	pullersTotal := 4
	pullerChan := make(chan pullerMsg)

	for i := 0; i < pullersTotal; i++ {
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
