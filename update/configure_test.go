package update

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupUtilityScripts(t *testing.T) {
	tempRootDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempRootDir)
	fakeConfigureDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(fakeConfigureDir)

	tempRootScriptsDir := path.Join(tempRootDir, "etc/systemd/system/scripts")
	tempRootBinDir := path.Join(tempRootDir, "opt/bin")

	// make some paths
	err = os.MkdirAll(tempRootScriptsDir, 0755)
	assert.Nil(t, err)
	err = os.MkdirAll(tempRootBinDir, 0755)
	assert.Nil(t, err)

	// create scripts that should NOT get deleted
	err = ioutil.WriteFile(path.Join(tempRootBinDir, "platconf"), []byte("foobar"), 0755)
	assert.Nil(t, err)
	err = ioutil.WriteFile(path.Join(tempRootBinDir, "protonet_zpool.sh"), []byte("barfoo"), 0755)
	assert.Nil(t, err)

	// create script that SHOULD get deleted
	err = ioutil.WriteFile(path.Join(tempRootBinDir, "whatever.sh"), []byte("0xB33F"), 0755)
	assert.Nil(t, err)

	// create script that should get added
	err = os.MkdirAll(path.Join(fakeConfigureDir, "scripts"), 0755)
	assert.Nil(t, err)
	err = ioutil.WriteFile(path.Join(fakeConfigureDir, "scripts", "newscript.sh"), []byte("lol"), 0755)
	assert.Nil(t, err)

	err = setupUtilityScripts(tempRootDir, fakeConfigureDir)
	assert.Nil(t, err)

	// test whether the protected files remain
	_, err = os.Stat(path.Join(tempRootBinDir, "platconf"))
	assert.Nil(t, err)
	_, err = os.Stat(path.Join(tempRootBinDir, "protonet_zpool.sh"))
	assert.Nil(t, err)

	// test whether the dropped file is gone
	_, err = os.Stat(path.Join(tempRootBinDir, "whatever.sh"))
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))

	// test whether the new script got installed
	_, err = os.Stat(path.Join(tempRootScriptsDir, "newscript.sh"))
	assert.Nil(t, err)

	// TODO test symlinks
}
