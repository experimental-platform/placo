package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
)

type githubReleaseAsset struct {
	URL                string `json:"url"`
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Label              string `json:"label"`
	ContentType        string `json:"content_type"`
	State              string `json:"state"`
	Size               int    `json:"size"`
	DownloadCount      int    `json:"download_count"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	BrowserDownloadURL string `json:"browser_download_url"`
	// skipped "uploader"
}

type githubRelease struct {
	URL             string               `json:"url"`
	AssetsURL       string               `json:"assets_url"`
	UploadURL       string               `json:"upload_url"`
	HTMLURL         string               `json:"html_url"`
	ID              int                  `json:"id"`
	TagName         string               `json:"tag_name"`
	TargetCommitish string               `json:"target_commitish"`
	Name            *interface{}         `json:"name"`
	Draft           bool                 `json:"draft"`
	Prerelease      bool                 `json:"prerelease"`
	CreatedAt       string               `json:"created_at"`
	PublishedAt     string               `json:"published_at"`
	Assets          []githubReleaseAsset `json:"assets"`
	TarballURL      string               `json:"tarball_url"`
	ZipballURL      string               `json:"zipball_url"`
	Body            *interface{}         `json:"body"`
	// skipped "author"
}

type selfupdateOpts struct {
	Force bool `short:"f" long:"force" description:"Force installing the current latest release"`
}

func (o *selfupdateOpts) Execute(args []string) error {
	err := runSelfUpdate(o.Force)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}

func runSelfUpdate(force bool) error {
	if force {
		fmt.Println("Forcing a self-update.")
	}

	if VersionTag == "" && !force {
		fmt.Println("Running a development binary, skipping update.")
		return nil
	}

	latestRelease, err := getLatestPlatconfRelease()
	if err != nil {
		return err
	}

	fmt.Printf("Latest release is %s\n", latestRelease.TagName)

	if latestRelease.TagName == VersionTag && !force {
		fmt.Println("Already up-to-date.")
		return nil
	}

	if len(latestRelease.Assets) != 1 {
		return fmt.Errorf("Latest release has %d assets. Cancelling.", len(latestRelease.Assets))
	}

	requireRoot()

	err = installPlatconfFromURL(latestRelease.Assets[0].BrowserDownloadURL)
	if err != nil {
		return err
	}

	fmt.Println("Self-update completed successfully.")
	return nil
}

func installPlatconfFromURL(url string) error {
	targetBinaryDir := "/opt/bin"
	targetBinaryFullPath := path.Join(targetBinaryDir, "platconf")
	tempFileFullPath := path.Join(targetBinaryDir, "platconf-download.tmp")

	tempFile, err := os.OpenFile(tempFileFullPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: OpenFile: %s", err.Error())
	}
	defer os.Remove(tempFileFullPath)
	defer tempFile.Close()

	fmt.Printf("Downloading new binary to '%s'\n", tempFileFullPath)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: http.Get: %s", err.Error())
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: Copy: %s", err.Error())
	}

	err = tempFile.Sync()
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: Sync: %s", err.Error())
	}

	tempFile.Close()

	fmt.Printf("Installing the new binary to '%s'\n", targetBinaryFullPath)
	err = os.MkdirAll(targetBinaryDir, 0755)
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: MkdirAll: %s", err.Error())
	}

	err = os.Rename(tempFileFullPath, targetBinaryFullPath)
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: Rename: %s", err.Error())
	}

	err = os.Chmod(targetBinaryFullPath, 0755)
	if err != nil {
		return fmt.Errorf("installPlatconfFromURL: Chmod: %s", err.Error())
	}

	return nil
}

func getLatestPlatconfRelease() (*githubRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/experimental-platform/platconf/releases/latest")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("No releases found.")
	}

	decoder := json.NewDecoder(resp.Body)
	var result githubRelease
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
