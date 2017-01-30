package update

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/fsouza/go-dockerclient"
)

type jsonstreamMessage struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Error  string `json:"error"`
}

// jsonstreamErrorDetector is a io.Writer wrapper whose purpose is to err out
// if the stream written to it will contain a Docker image
type jsonstreamErrorDetector struct {
	buffer *bytes.Buffer
}

func (jsed *jsonstreamErrorDetector) Write(p []byte) (n int, err error) {
	if jsed.buffer == nil {
		jsed.buffer = bytes.NewBuffer([]byte{})
	}

	n, err = jsed.buffer.Write(p)
	if err != nil {
		return 0, err
	}

	var msg jsonstreamMessage
	dec := json.NewDecoder(jsed.buffer)

	for {
		err = dec.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
	}

	if msg.Error != "" {
		return 0, fmt.Errorf("Docker error: %s", msg.Error)
	}

	return n, nil
}

func pullImage(repository, tag string) error {
	var jsed jsonstreamErrorDetector

	opts := docker.PullImageOptions{
		Repository:    repository,
		Tag:           tag,
		OutputStream:  &jsed,
		RawJSONStream: true,
	}

	client, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	auth := docker.AuthConfiguration{}

	err = client.PullImage(opts, auth)
	if err != nil {
		return err
	}

	return nil
}

// exportDockerImage writes a TAR archive of a given Docker image's rootfs
// into the output writer.
//
// If returned error is not nil then the final state of the output writer
// is not defined, i.e. the data might have been written partially.
func exportDockerImage(repository, tag string, output io.Writer) error {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	createContainerOpts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: fmt.Sprintf("%s:%s", repository, tag),
		},
	}

	containter, err := c.CreateContainer(createContainerOpts)
	if err != nil {
		return err
	}
	defer c.RemoveContainer(docker.RemoveContainerOptions{ID: containter.ID, RemoveVolumes: true, Force: true})

	exportContainerOptions := docker.ExportContainerOptions{
		ID:           containter.ID,
		OutputStream: output,
	}

	return c.ExportContainer(exportContainerOptions)
}

// extractDockerImage writes a given image's rootfs to the target folder,
// ignoring everything that isn't a regular file or directory
func extractDockerImage(repository, tag, extractDir string) error {
	// this pipe connects exportDockerImage to the tar reader
	pipeReader, pipeWriter := io.Pipe()

	// dump the TAR-ed image to one end of the pipe
	var wg sync.WaitGroup
	wg.Add(1)
	var extractErr error
	go func() {
		extractErr = exportDockerImage(repository, tag, pipeWriter)
		// need to close this end or the other end will block
		pipeWriter.Close()
		wg.Done()
	}()

	// wrap a TAR reader around the other and extract
	tarReader := tar.NewReader(pipeReader)
	for {
		header, nextErr := tarReader.Next()
		if nextErr != nil {
			if nextErr == io.EOF {
				break
			}
			return nextErr
		}

		switch header.Typeflag {
		case tar.TypeDir:
			mkdirErr := os.MkdirAll(path.Join(extractDir, header.Name), 0755)
			if mkdirErr != nil {
				return mkdirErr
			}
			io.CopyN(ioutil.Discard, tarReader, header.Size)
		case tar.TypeReg:
			f, err := os.OpenFile(path.Join(extractDir, header.Name), os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			_, err = io.CopyN(f, tarReader, header.Size)
			if err != nil {
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			linkErr := os.Symlink(header.Linkname, path.Join(extractDir, header.Name))
			if linkErr != nil {
				return linkErr
			}
		default:
			// ignore
			io.CopyN(ioutil.Discard, tarReader, header.Size)
		}
	}

	wg.Wait()

	if extractErr != nil {
		return extractErr
	}

	return nil
}
