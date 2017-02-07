package update

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

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
			err := os.Remove(fullpath)
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
			err := os.Remove(fullpath)
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
