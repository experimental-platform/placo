package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

var statusSocketPath = "/var/run/platconf-status.sock"

type statusData struct {
	Status   string   `json:"status"`
	Progress *float32 `json:"progress"`
	What     *string  `json:"what"`
}

func setStatus(status string, progress *float32, what *string) error {
	fakeDial := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", statusSocketPath)
	}

	client := http.Client{
		Transport: &http.Transport{
			Dial: fakeDial,
		},
	}

	sd := statusData{
		Status:   status,
		Progress: progress,
		What:     what,
	}

	data, err := json.Marshal(&sd)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "http://oldstatus/status", bytes.NewReader(data))
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("setStatus: return code is %d", response.StatusCode)
	}

	return nil
}
