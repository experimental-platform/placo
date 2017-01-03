package platconf

// ReleaseManifestV1 describes a the build manifests used
// by Kamil's update system in late 2016.
type ReleaseManifestV1 struct {
	Build       int32             `json:"build"`
	Codename    string            `json:"codename"`
	URL         string            `json:"url"`
	PublishedAt string            `json:"published_at"`
	Images      map[string]string `json:"images"`
}

// ReleaseManifestV2 describes the build manifests introduced for platconf
// by Kamil in early 2017
type ReleaseManifestV2 struct {
	Build           int32                    `json:"build"`
	Codename        string                   `json:"codename"`
	ReleaseNotesURL string                   `json:"url"`
	PublishedAt     string                   `json:"published_at"`
	Images          []ReleaseManifestV2Image `json:"images"`
}

// ReleaseManifestV2Image describes an image entry in ReleaseManifestV2
type ReleaseManifestV2Image struct {
	Name        string `json:"name"`         // full image name w/ registry name minus tag
	Tag         string `json:"tag"`          //
	PreDownload bool   `json:"pre_download"` // Should the image be downloaded pre-emptively by update?
}
