package update

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullImage(t *testing.T) {
	// pull is OK
	msgStream1 := `{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}`
	// pull errored
	msgStream2 := `{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"error": "something bad happened"}`
	// msg stream is cut off
	msgStream3 := `{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "ba`

	testImageRepo := "quay.io/protonet/dummy"

	var testHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/create" {
			http.Error(w, "Not found.", http.StatusNotFound)
		}

		// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.22/#/create-an-image
		query := r.URL.Query()
		fromImage := query.Get("fromImage")
		tag := query.Get("tag")

		if fromImage == "" || tag == "" {
			http.Error(w, "Missing parameter 'fromImage' or 'tag'", http.StatusBadRequest)
		}

		if fromImage != testImageRepo {
			http.Error(w, "Not found.", http.StatusNotFound)
		}

		switch tag {
		case "v1":
			_, err := w.Write([]byte(msgStream1))
			assert.Nil(t, err)
		case "v2":
			_, err := w.Write([]byte(msgStream2))
			assert.Nil(t, err)
		case "v3":
			_, err := w.Write([]byte(msgStream3))
			assert.Nil(t, err)
		default:
			http.Error(w, "Not found.", http.StatusNotFound)
		}
	}

	srv := httptest.NewServer(testHandler)

	os.Setenv("DOCKER_HOST", srv.URL)
	defer os.Setenv("DOCKER_HOST", "")

	// pull is OK
	err := pullImage(testImageRepo, "v1")
	assert.Nil(t, err)

	// pull errored
	err = pullImage(testImageRepo, "v2")
	assert.EqualError(t, err, "Docker error: something bad happened")

	// msg stream is cut off
	err = pullImage(testImageRepo, "v3")
	assert.Equal(t, err, io.ErrUnexpectedEOF)
}

// TestExportDockerImage tests the functionality of streaming the data
// from a image's rootfs by exporting a pristine container
func TestExportDockerImage(t *testing.T) {
	testContainerID := "b99a1defb349"
	testTARBase64 := "H4sICJJWblgAA2FyY2hpdmUudGFyAO2VUQ7CIAxA+fYUvcFaCuw86Fhi5tS4mez4wrbol0uWCGrG++lPQ0sfpLYQ0UHEUmsYo5kiSjXFGSBGZqVlyQRIUkkpQMdvTYh719ubb6WpLq09d83xTZ5Pq+uFc+Z7POOfYIt99Bewwj+jUcG/UZz9pyD4ryLX8PMwSi34J/36/0Z6/4xEAjByXyMb9+8G215PDno39LtvN5NJzuHn9n857n/O+z8JLkGNNfufiIN/jVqAXJr3p9i4/0wms10eO0nUugAQAAA="
	testTARBase64Buffer := bytes.NewBufferString(testTARBase64)
	testTARBuffer := bytes.NewBuffer([]byte{})

	dec := base64.NewDecoder(base64.StdEncoding, testTARBase64Buffer)
	gunzip, err := gzip.NewReader(dec)
	assert.Nil(t, err)
	_, err = io.Copy(testTARBuffer, gunzip)
	assert.Nil(t, err)

	var testHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if r.URL.Path == "/containers/create" {
				response := struct {
					ID       string `json:"Id"`
					Warnings []string
				}{ID: testContainerID}
				json.NewEncoder(w).Encode(&response)
				return
			}
			break
		case "GET":
			if r.URL.Path == fmt.Sprintf("/containers/%s/export", testContainerID) {
				w.Header().Add("Content-Type", "application/octet-stream")
				w.Write(testTARBuffer.Bytes())
				return
			}
			break
		case "DELETE":
			if r.URL.Path == fmt.Sprintf("/containers/%s", testContainerID) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			break
		}

		http.Error(w, "Not found.", http.StatusNotFound)
		assert.Fail(t, "test tried to access a wrong path", "%s %+v", r.Method, r.URL)
	}

	srv := httptest.NewServer(testHandler)

	os.Setenv("DOCKER_HOST", srv.URL)
	os.Setenv("DOCKER_API_VERSION", "1.22")
	defer os.Setenv("DOCKER_HOST", "")
	defer os.Setenv("DOCKER_API_VERSION", "")

	// direct export to buffer
	imageBuf := bytes.NewBuffer([]byte{})
	err = exportDockerImage("repository", "tag", imageBuf)
	assert.Nil(t, err)
	assert.EqualValues(t, testTARBuffer.Bytes(), imageBuf.Bytes())

	// export through pipe
	imageBuf2 := bytes.NewBuffer([]byte{})
	pipeReader, pipeWriter := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	var extractErr error
	go func() {
		extractErr = exportDockerImage("repository", "tag", bufio.NewWriter(pipeWriter))
		// needs to be closed, or io.Copy from the other end will be stuck forever
		pipeWriter.Close()
		wg.Done()
	}()

	io.Copy(bufio.NewWriter(imageBuf2), bufio.NewReader(pipeReader))
	wg.Wait()
}

func TestExtractDockerImage(t *testing.T) {
	testContainerID := "b99a1defb349"
	testTARBase64 := "H4sICJJWblgAA2FyY2hpdmUudGFyAO2VUQ7CIAxA+fYUvcFaCuw86Fhi5tS4mez4wrbol0uWCGrG++lPQ0sfpLYQ0UHEUmsYo5kiSjXFGSBGZqVlyQRIUkkpQMdvTYh719ubb6WpLq09d83xTZ5Pq+uFc+Z7POOfYIt99Bewwj+jUcG/UZz9pyD4ryLX8PMwSi34J/36/0Z6/4xEAjByXyMb9+8G215PDno39LtvN5NJzuHn9n857n/O+z8JLkGNNfufiIN/jVqAXJr3p9i4/0wms10eO0nUugAQAAA="
	testTARBase64Buffer := bytes.NewBufferString(testTARBase64)
	testTARBuffer := bytes.NewBuffer([]byte{})

	dec := base64.NewDecoder(base64.StdEncoding, testTARBase64Buffer)
	gunzip, err := gzip.NewReader(dec)
	assert.Nil(t, err)
	_, err = io.Copy(testTARBuffer, gunzip)
	assert.Nil(t, err)

	var testHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if r.URL.Path == "/containers/create" {
				response := struct {
					ID       string `json:"Id"`
					Warnings []string
				}{ID: testContainerID}
				json.NewEncoder(w).Encode(&response)
				return
			}
			break
		case "GET":
			if r.URL.Path == fmt.Sprintf("/containers/%s/export", testContainerID) {
				w.Header().Add("Content-Type", "application/octet-stream")
				w.Write(testTARBuffer.Bytes())
				return
			}
			break
		case "DELETE":
			if r.URL.Path == fmt.Sprintf("/containers/%s", testContainerID) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			break
		}

		http.Error(w, "Not found.", http.StatusNotFound)
		assert.Fail(t, "test tried to access a wrong path", "%s %+v", r.Method, r.URL)
	}

	srv := httptest.NewServer(testHandler)

	os.Setenv("DOCKER_HOST", srv.URL)
	os.Setenv("DOCKER_API_VERSION", "1.22")
	defer os.Setenv("DOCKER_HOST", "")
	defer os.Setenv("DOCKER_API_VERSION", "")

	tempDir, err := ioutil.TempDir("", "platconf-unittest-")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	err = extractDockerImage("repository", "tag", tempDir)
	assert.Nil(t, err)

	rootInfo, err := ioutil.ReadDir(tempDir)
	assert.Nil(t, err)
	assert.Len(t, rootInfo, 3)

	// 'a'
	assert.Equal(t, "a", rootInfo[0].Name())
	assert.True(t, rootInfo[0].IsDir())
	aInfo, err := ioutil.ReadDir(path.Join(tempDir, "a"))
	assert.Nil(t, err)
	assert.Len(t, aInfo, 1)

	// 'a/b'
	assert.Equal(t, "b", aInfo[0].Name())
	assert.True(t, aInfo[0].IsDir())
	abInfo, err := ioutil.ReadDir(path.Join(tempDir, "a", "b"))
	assert.Nil(t, err)
	assert.Len(t, abInfo, 1)

	// 'a/b/d'
	assert.Equal(t, "d", abInfo[0].Name())
	assert.False(t, abInfo[0].IsDir())
	dContent, err := ioutil.ReadFile(path.Join(tempDir, "a", "b", "d"))
	assert.Nil(t, err)
	assert.Equal(t, "example text\n", string(dContent))

	// 'c'
	assert.Equal(t, "c", rootInfo[1].Name())
	assert.True(t, rootInfo[1].IsDir())
	cInfo, err := ioutil.ReadDir(path.Join(tempDir, "c"))
	assert.Nil(t, err)
	assert.Len(t, cInfo, 0)

	// 'e'
	assert.Equal(t, "e", rootInfo[2].Name())
	assert.NotZero(t, rootInfo[2].Mode()|os.ModeSymlink, "the file 'e' is not a symlink")
}
