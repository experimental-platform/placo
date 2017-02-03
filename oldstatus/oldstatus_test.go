package oldstatus

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStatus(t *testing.T) {
	// prepare
	var status StatusData

	mux := getStatusReadMux(&status)
	srv := httptest.NewServer(mux)

	u, err := url.Parse(srv.URL)
	assert.Nil(t, err)
	u.Path = path.Join(u.Path, "json")

	// test1
	status.Lock()
	var p1 float32 = 1.2
	status.Progress = &p1
	w1 := "foobar_img"
	status.What = &w1
	status.Status = "whatever"
	status.Unlock()

	resp, err := http.Get(u.String())
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var responseStatus1 StatusData
	decoder1 := json.NewDecoder(resp.Body)
	assert.Nil(t, decoder1.Decode(&responseStatus1))

	status.RLock()
	assert.Equal(t, p1, *responseStatus1.Progress)
	assert.Equal(t, w1, *responseStatus1.What)
	assert.Equal(t, "whatever", responseStatus1.Status)
	status.RUnlock()

	// test2
	status.Lock()
	status.Progress = nil
	status.What = nil
	status.Status = "other_status"
	status.Unlock()

	resp, err = http.Get(u.String())
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var responseStatus2 StatusData
	decoder2 := json.NewDecoder(resp.Body)
	assert.Nil(t, decoder2.Decode(&responseStatus2))

	status.RLock()
	assert.Nil(t, responseStatus2.Progress)
	assert.Nil(t, responseStatus2.What)
	assert.Equal(t, "other_status", responseStatus2.Status)
	status.RUnlock()
}

func TestUpdateByFile2(t *testing.T) {
	preservedStatusFilePath := statusFilePath
	defer func() { statusFilePath = preservedStatusFilePath }()

	f, err := ioutil.TempFile("", "platconf-unittest-")
	assert.Nil(t, err)
	statusFilePath = f.Name()
	f.Close()
}
