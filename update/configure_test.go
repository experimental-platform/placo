package update

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/experimental-platform/platconf/platconf"
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

	// create script in script dir that SHOULD get deleted
	err = ioutil.WriteFile(path.Join(tempRootScriptsDir, "whatever.sh"), []byte("0xB33F"), 0755)
	assert.Nil(t, err)

	// create link in bin dir that SHOULD get deleted
	err = os.Symlink(path.Join(tempRootScriptsDir, "whatever.sh"), path.Join(tempRootBinDir, "whatever-link"))
	assert.Nil(t, err)

	// create script that should get added
	err = os.MkdirAll(path.Join(fakeConfigureDir, "scripts"), 0755)
	assert.Nil(t, err)
	err = ioutil.WriteFile(path.Join(fakeConfigureDir, "scripts", "newscript.sh"), []byte("lol"), 0755)
	assert.Nil(t, err)

	err = setupUtilityScripts(tempRootDir, fakeConfigureDir)
	assert.Nil(t, err)

	// test whether the protected files remain
	_, err = os.Lstat(path.Join(tempRootBinDir, "platconf"))
	assert.Nil(t, err)
	_, err = os.Lstat(path.Join(tempRootBinDir, "protonet_zpool.sh"))
	assert.Nil(t, err)

	// test whether the dropped script is gone
	_, err = os.Lstat(path.Join(tempRootScriptsDir, "whatever.sh"))
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))

	// test whether the dropped symlink is gone
	_, err = os.Lstat(path.Join(tempRootBinDir, "whatever-link"))
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))

	// test whether the new script got installed
	_, err = os.Lstat(path.Join(tempRootScriptsDir, "newscript.sh"))
	assert.Nil(t, err)

	// test whether the new symlink got installed
	_, err = os.Lstat(path.Join(tempRootBinDir, "newscript"))
	assert.Nil(t, err)
}

func TestParseTemplate(t *testing.T) {
	pristineUnit := `# ExperimentalPlatform
[Unit]
Description=CollectD
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=0
TimeoutStopSec=15
Restart=always
RestartSec=5s
ExecStartPre=/usr/bin/mkdir -p /data/collectd/rrd
ExecStartPre=-/usr/bin/docker rm -f collectd
ExecStartPre=/usr/bin/docker run -d \
    --name collectd \
    --net host \
    --volume /data/collectd/rrd:/rrd:rw \
    --volume /dev:/dev:ro \
    --volume /var/run/docker.sock:/var/run/docker.sock \
    quay.io/experimentalplatform/collectd:{{tag}}
ExecStart=/usr/bin/docker logs -f collectd
ExecStop=/usr/bin/docker stop collectd
ExecStopPost=/usr/bin/docker stop collectd

[Install]
WantedBy=multi-user.target`

	parsedUnit := `# ExperimentalPlatform
[Unit]
Description=CollectD
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=0
TimeoutStopSec=15
Restart=always
RestartSec=5s
ExecStartPre=/usr/bin/mkdir -p /data/collectd/rrd
ExecStartPre=-/usr/bin/docker rm -f collectd
ExecStartPre=/usr/bin/docker run -d \
    --name collectd \
    --net host \
    --volume /data/collectd/rrd:/rrd:rw \
    --volume /dev:/dev:ro \
    --volume /var/run/docker.sock:/var/run/docker.sock \
    quay.io/experimentalplatform/collectd:release-tag-1234
ExecStart=/usr/bin/docker logs -f collectd
ExecStop=/usr/bin/docker stop collectd
ExecStopPost=/usr/bin/docker stop collectd

[Install]
WantedBy=multi-user.target`
	manifest := platconf.ReleaseManifestV2{
		Images: []platconf.ReleaseManifestV2Image{
			{
				Name: "quay.io/experimentalplatform/collectd",
				Tag:  "release-tag-1234",
			},
		},
	}

	tempFile, err := ioutil.TempFile("", "platconf-unittest-")
	assert.Nil(t, err)

	unitFile := tempFile.Name()
	defer os.Remove(unitFile)

	_, err = tempFile.WriteString(pristineUnit)
	assert.Nil(t, err)
	tempFile.Close()

	err = parseTemplate(tempFile.Name(), &manifest)
	assert.Nil(t, err)

	readData, err := ioutil.ReadFile(unitFile)
	assert.Nil(t, err)

	assert.Equal(t, parsedUnit, string(readData))
}

func TestParseAllTemplates(t *testing.T) {
	pristineUnit := `# ExperimentalPlatform
[Unit]
Description=CollectD
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=0
TimeoutStopSec=15
Restart=always
RestartSec=5s
ExecStartPre=/usr/bin/mkdir -p /data/collectd/rrd
ExecStartPre=-/usr/bin/docker rm -f collectd
ExecStartPre=/usr/bin/docker run -d \
    --name collectd \
    --net host \
    --volume /data/collectd/rrd:/rrd:rw \
    --volume /dev:/dev:ro \
    --volume /var/run/docker.sock:/var/run/docker.sock \
    quay.io/experimentalplatform/collectd:{{tag}}
ExecStart=/usr/bin/docker logs -f collectd
ExecStop=/usr/bin/docker stop collectd
ExecStopPost=/usr/bin/docker stop collectd

[Install]
WantedBy=multi-user.target`

	parsedUnit := `# ExperimentalPlatform
[Unit]
Description=CollectD
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=0
TimeoutStopSec=15
Restart=always
RestartSec=5s
ExecStartPre=/usr/bin/mkdir -p /data/collectd/rrd
ExecStartPre=-/usr/bin/docker rm -f collectd
ExecStartPre=/usr/bin/docker run -d \
    --name collectd \
    --net host \
    --volume /data/collectd/rrd:/rrd:rw \
    --volume /dev:/dev:ro \
    --volume /var/run/docker.sock:/var/run/docker.sock \
    quay.io/experimentalplatform/collectd:release-tag-1234
ExecStart=/usr/bin/docker logs -f collectd
ExecStop=/usr/bin/docker stop collectd
ExecStopPost=/usr/bin/docker stop collectd

[Install]
WantedBy=multi-user.target`
	manifest := platconf.ReleaseManifestV2{
		Images: []platconf.ReleaseManifestV2Image{
			{
				Name: "quay.io/experimentalplatform/collectd",
				Tag:  "release-tag-1234",
			},
		},
	}

	tempConfigureDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempConfigureDir)

	tempRootDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempRootDir)

	err = os.MkdirAll(path.Join(tempConfigureDir, "services"), 0755)
	assert.Nil(t, err)
	unitFile := path.Join(tempConfigureDir, "services", "sample.service")

	err = ioutil.WriteFile(unitFile, []byte(pristineUnit), 0644)
	assert.Nil(t, err)

	err = parseAllTemplates(tempRootDir, tempConfigureDir, &manifest)
	assert.Nil(t, err)

	readData, err := ioutil.ReadFile(unitFile)
	assert.Nil(t, err)

	assert.Equal(t, parsedUnit, string(readData))
}

func TestIsBrokenLink(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	// fail on relativePath
	_, err = isBrokenLink("relative/path")
	assert.Equal(t, ErrIsRelative, err)

	// a regular file
	fullPath := path.Join(tempDir, "regular")
	err = ioutil.WriteFile(fullPath, []byte{}, 0644)
	assert.Nil(t, err)
	isBroken, err := isBrokenLink(fullPath)
	assert.Nil(t, err)
	assert.False(t, isBroken)

	// a directory
	fullPath = path.Join(tempDir, "some_dir")
	err = os.Mkdir(fullPath, 0755)
	assert.Nil(t, err)
	isBroken, err = isBrokenLink(fullPath)
	assert.Nil(t, err)
	assert.False(t, isBroken)

	// a correct link (relative)
	// create existing target
	err = ioutil.WriteFile(path.Join(tempDir, "existing-target-relative"), []byte{}, 0644)
	assert.Nil(t, err)
	// create symlink
	fullPath = path.Join(tempDir, "correct-symlink-relative")
	err = os.Symlink("existing-target-relative", fullPath)
	assert.Nil(t, err)
	// test
	isBroken, err = isBrokenLink(fullPath)
	assert.Nil(t, err)
	assert.False(t, isBroken)

	// a broken link (relative)
	// create symlink
	fullPath = path.Join(tempDir, "broken-symlink-relative")
	err = os.Symlink("absent-target-relative", fullPath)
	assert.Nil(t, err)
	// test
	isBroken, err = isBrokenLink(fullPath)
	assert.Nil(t, err)
	assert.True(t, isBroken)

	// a correct link (absolute)
	// create symlink
	fullPath = path.Join(tempDir, "correct-symlink-absolute")
	err = os.Symlink("/dev/null", fullPath)
	assert.Nil(t, err)
	// test
	isBroken, err = isBrokenLink(fullPath)
	assert.Nil(t, err)
	assert.False(t, isBroken)

	// a broken link (absolute)
	// create symlink
	fullPath = path.Join(tempDir, "broken-symlink-absolute")
	err = os.Symlink("/dev/absent-target-relative", fullPath)
	assert.Nil(t, err)
	// test
	isBroken, err = isBrokenLink(fullPath)
	assert.Nil(t, err)
	assert.True(t, isBroken)
}

func TestRemoveBrokenLinks(t *testing.T) {
	//isBrokenLink
	tempDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	err = removeBrokenLinks("relative/path")
	assert.Equal(t, ErrIsRelative, err)

	// create a regular file
	fullPath := path.Join(tempDir, "a_regular")
	err = ioutil.WriteFile(fullPath, []byte{}, 0644)
	assert.Nil(t, err)

	// create a directory
	fullPath = path.Join(tempDir, "b_some_dir")
	err = os.Mkdir(fullPath, 0755)
	assert.Nil(t, err)

	// a correct link (relative)
	fullPath = path.Join(tempDir, "c_correct-symlink-relative")
	err = os.Symlink("a_regular", fullPath)
	assert.Nil(t, err)

	// a broken link (relative)
	fullPath = path.Join(tempDir, "d_broken-symlink-relative")
	err = os.Symlink("z_absent-target-relative", fullPath)
	assert.Nil(t, err)

	// a correct link (absolute)
	fullPath = path.Join(tempDir, "e_correct-symlink-absolute")
	err = os.Symlink("/dev/null", fullPath)
	assert.Nil(t, err)

	// a broken link (absolute)
	fullPath = path.Join(tempDir, "f_broken-symlink-absolute")
	err = os.Symlink("/dev/absent-target-relative", fullPath)
	assert.Nil(t, err)

	err = removeBrokenLinks(tempDir)
	assert.Nil(t, err)

	fileinfo, err := ioutil.ReadDir(tempDir)
	assert.Nil(t, err)
	assert.Len(t, fileinfo, 4)
}

func TestIsPlatformUnit(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	// fail on relativePath
	_, err = isPlatformUnit("relative/path")
	assert.Equal(t, ErrIsRelative, err)

	// a directory
	fullPath := path.Join(tempDir, "some_dir")
	err = os.Mkdir(fullPath, 0755)
	assert.Nil(t, err)
	isPlatform, err := isPlatformUnit(fullPath)
	assert.Nil(t, err)
	assert.False(t, isPlatform)

	// a regular file, non platform
	fullPath = path.Join(tempDir, "regular")
	err = ioutil.WriteFile(fullPath, []byte{}, 0644)
	assert.Nil(t, err)
	isPlatform, err = isPlatformUnit(fullPath)
	assert.Nil(t, err)
	assert.False(t, isPlatform)

	// a regular file, platform unit
	fullPath = path.Join(tempDir, "regular2")
	err = ioutil.WriteFile(fullPath, []byte("# ExperimentalPlatform \nfoobar"), 0644)
	assert.Nil(t, err)
	isPlatform, err = isPlatformUnit(fullPath)
	assert.Nil(t, err)
	assert.True(t, isPlatform)
}

func TestRemovePlatformUnits(t *testing.T) {
	//isBrokenLink
	tempDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	err = removePlatformUnits("relative/path")
	assert.Equal(t, ErrIsRelative, err)

	// create a regular file, non platform
	fullPath := path.Join(tempDir, "a_regular")
	err = ioutil.WriteFile(fullPath, []byte("whatever\nbar\nfoo"), 0644)
	assert.Nil(t, err)

	// create a directory
	fullPath = path.Join(tempDir, "b_some_dir")
	err = os.Mkdir(fullPath, 0755)
	assert.Nil(t, err)

	// create a platform unit
	fullPath = path.Join(tempDir, "c_correct-symlink-relative")
	err = ioutil.WriteFile(fullPath, []byte("# ExperimentalPlatform \nfoobar"), 0644)
	assert.Nil(t, err)

	err = removePlatformUnits(tempDir)
	assert.Nil(t, err)

	fileinfo, err := ioutil.ReadDir(tempDir)
	assert.Nil(t, err)
	assert.Len(t, fileinfo, 2)
}
