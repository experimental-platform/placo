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
