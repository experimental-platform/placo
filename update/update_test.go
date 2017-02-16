package update

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/experimental-platform/platconf/platconf"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateSetupPaths(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	err = setupPaths(tempDir)
	assert.Nil(t, err)

	requiredPaths := []string{
		"/etc/systemd/journal.conf.d",
		"/etc/systemd/system",
		"/etc/systemd/system/docker.service.d",
		"/etc/systemd/system/scripts",
		"/etc/udev/rules.d",
		"/opt/bin",
	}

	for _, p := range requiredPaths {
		fileinfo, err := os.Stat(path.Join(tempDir, p))
		assert.Nil(t, err)
		assert.True(t, fileinfo.IsDir())
	}
}

func TestFetchReleaseJSON(t *testing.T) {
	testBody := "foobarteststring"
	testChannel := "WhateverTheF"
	testChannelNoAccess := "GoAwayChannel"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	mockURL1 := fmt.Sprintf("https://raw.githubusercontent.com/protonet/builds/master/manifest-v2/%s.json", testChannel)
	httpmock.RegisterResponder("GET", mockURL1, httpmock.NewStringResponder(200, testBody))

	mockURL2 := fmt.Sprintf("https://raw.githubusercontent.com/protonet/builds/master/manifest-v2/%s.json", testChannelNoAccess)
	httpmock.RegisterResponder("GET", mockURL2, httpmock.NewStringResponder(403, "Access denied."))

	data, err := fetchReleaseJSONv2(testChannel)
	assert.Nil(t, err)
	assert.Equal(t, len(testBody), len(data))

	_, err = fetchReleaseJSONv2(testChannelNoAccess)
	assert.NotNil(t, err)

	_, err = fetchReleaseJSONv2("noSuchChannel")
	assert.NotNil(t, err)
}

func TestFetchReleaseData(t *testing.T) {
	testBody := `{
  "build": 12345,
  "codename": "Kaufman",
  "url": "https://www.example.com/",
  "published_at": "1990-12-31T23:59:60Z",
  "images": [
  	{
  		"name": "quay.io/experiementalplatform/geilerserver",
  		"tag": "v1.2.3.4",
  		"pre_download": true
  	},
  	{
  		"name": "quay.io/protonet/rickroll",
  		"tag": "latest",
  		"pre_download": false
  	}
  ]
}`
	testBrokenJSON := "213ewqsd"

	expectedJSON := platconf.ReleaseManifestV2{
		Build:           12345,
		Codename:        "Kaufman",
		ReleaseNotesURL: "https://www.example.com/",
		PublishedAt:     "1990-12-31T23:59:60Z",
		Images: []platconf.ReleaseManifestV2Image{
			platconf.ReleaseManifestV2Image{
				Name:        "quay.io/experiementalplatform/geilerserver",
				Tag:         "v1.2.3.4",
				PreDownload: true,
			},
			platconf.ReleaseManifestV2Image{
				Name:        "quay.io/protonet/rickroll",
				Tag:         "latest",
				PreDownload: false,
			},
		},
	}
	testChannel := "WhateverTheF"
	testChannelBrokenJSON := "SomeOtherChan"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	mockURL1 := fmt.Sprintf("https://raw.githubusercontent.com/protonet/builds/master/manifest-v2/%s.json", testChannel)
	httpmock.RegisterResponder("GET", mockURL1, httpmock.NewStringResponder(200, testBody))
	mockURL2 := fmt.Sprintf("https://raw.githubusercontent.com/protonet/builds/master/manifest-v2/%s.json", testChannelBrokenJSON)
	httpmock.RegisterResponder("GET", mockURL2, httpmock.NewStringResponder(200, testBrokenJSON))

	manifest, err := fetchReleaseDataV2(testChannel)
	assert.Nil(t, err)
	assert.NotNil(t, manifest)
	assert.EqualValues(t, expectedJSON, *manifest)

	_, err = fetchReleaseDataV2(testChannelBrokenJSON)
	assert.NotNil(t, err)
}
