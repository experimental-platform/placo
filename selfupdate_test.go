package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalRelease(t *testing.T) {
	githubReleaseTestJSON := `{
  "url": "https://api.github.com/repos/experimental-platform/release-tagger/releases/4842813",
  "assets_url": "https://api.github.com/repos/experimental-platform/release-tagger/releases/4842813/assets",
  "upload_url": "https://uploads.github.com/repos/experimental-platform/release-tagger/releases/4842813/assets{?name,label}",
  "html_url": "https://github.com/experimental-platform/release-tagger/releases/tag/v5",
  "id": 4842813,
  "tag_name": "v5",
  "target_commitish": "master",
  "name": null,
  "draft": false,
  "author": {
    "login": "kdomanski",
    "id": 993296,
    "avatar_url": "https://avatars.githubusercontent.com/u/993296?v=3",
    "gravatar_id": "",
    "url": "https://api.github.com/users/kdomanski",
    "html_url": "https://github.com/kdomanski",
    "followers_url": "https://api.github.com/users/kdomanski/followers",
    "following_url": "https://api.github.com/users/kdomanski/following{/other_user}",
    "gists_url": "https://api.github.com/users/kdomanski/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/kdomanski/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/kdomanski/subscriptions",
    "organizations_url": "https://api.github.com/users/kdomanski/orgs",
    "repos_url": "https://api.github.com/users/kdomanski/repos",
    "events_url": "https://api.github.com/users/kdomanski/events{/privacy}",
    "received_events_url": "https://api.github.com/users/kdomanski/received_events",
    "type": "User",
    "site_admin": false
  },
  "prerelease": false,
  "created_at": "2016-12-06T10:51:04Z",
  "published_at": "2016-12-06T10:53:38Z",
  "assets": [
    {
      "url": "https://api.github.com/repos/experimental-platform/release-tagger/releases/assets/2773798",
      "id": 2773798,
      "name": "tagger-v5-linux",
      "label": "",
      "uploader": {
        "login": "kdomanski",
        "id": 993296,
        "avatar_url": "https://avatars.githubusercontent.com/u/993296?v=3",
        "gravatar_id": "",
        "url": "https://api.github.com/users/kdomanski",
        "html_url": "https://github.com/kdomanski",
        "followers_url": "https://api.github.com/users/kdomanski/followers",
        "following_url": "https://api.github.com/users/kdomanski/following{/other_user}",
        "gists_url": "https://api.github.com/users/kdomanski/gists{/gist_id}",
        "starred_url": "https://api.github.com/users/kdomanski/starred{/owner}{/repo}",
        "subscriptions_url": "https://api.github.com/users/kdomanski/subscriptions",
        "organizations_url": "https://api.github.com/users/kdomanski/orgs",
        "repos_url": "https://api.github.com/users/kdomanski/repos",
        "events_url": "https://api.github.com/users/kdomanski/events{/privacy}",
        "received_events_url": "https://api.github.com/users/kdomanski/received_events",
        "type": "User",
        "site_admin": false
      },
      "content_type": "application/octet-stream",
      "state": "uploaded",
      "size": 9029046,
      "download_count": 1,
      "created_at": "2016-12-06T10:53:35Z",
      "updated_at": "2016-12-06T10:53:38Z",
      "browser_download_url": "https://github.com/experimental-platform/release-tagger/releases/download/v5/tagger-v5-linux"
    }
	],
  "tarball_url": "https://api.github.com/repos/experimental-platform/release-tagger/tarball/v5",
  "zipball_url": "https://api.github.com/repos/experimental-platform/release-tagger/zipball/v5",
  "body": null
}`

	githubReleaseTestExpectedData := githubRelease{
		URL:             "https://api.github.com/repos/experimental-platform/release-tagger/releases/4842813",
		AssetsURL:       "https://api.github.com/repos/experimental-platform/release-tagger/releases/4842813/assets",
		UploadURL:       "https://uploads.github.com/repos/experimental-platform/release-tagger/releases/4842813/assets{?name,label}",
		HTMLURL:         "https://github.com/experimental-platform/release-tagger/releases/tag/v5",
		ID:              4842813,
		TagName:         "v5",
		TargetCommitish: "master",
		Name:            nil,
		Draft:           false,
		Prerelease:      false,
		CreatedAt:       "2016-12-06T10:51:04Z",
		PublishedAt:     "2016-12-06T10:53:38Z",
		Assets: []githubReleaseAsset{
			{
				URL:                "https://api.github.com/repos/experimental-platform/release-tagger/releases/assets/2773798",
				ID:                 2773798,
				Name:               "tagger-v5-linux",
				Label:              "",
				ContentType:        "application/octet-stream",
				State:              "uploaded",
				Size:               9029046,
				DownloadCount:      1,
				CreatedAt:          "2016-12-06T10:53:35Z",
				UpdatedAt:          "2016-12-06T10:53:38Z",
				BrowserDownloadURL: "https://github.com/experimental-platform/release-tagger/releases/download/v5/tagger-v5-linux",
			},
		},
		TarballURL: "https://api.github.com/repos/experimental-platform/release-tagger/tarball/v5",
		ZipballURL: "https://api.github.com/repos/experimental-platform/release-tagger/zipball/v5",
		Body:       nil,
	}

	var resultData githubRelease
	err := json.Unmarshal([]byte(githubReleaseTestJSON), &resultData)
	assert.Nil(t, err)
	assert.Equal(t, githubReleaseTestExpectedData, resultData)
}
