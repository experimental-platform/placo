package oldstatus

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Opts contains command line parameters for the 'oldstatus' command
type Opts struct {
	Port int `short:"p" long:"port" description:"Port on which to listen" default:"7887"`
}

// StatusData is the data structure sent to the status page
type StatusData struct {
	Status       string   `json:"status"`
	Progress     *float32 `json:"progress"`
	What         *string  `json:"what"`
	sync.RWMutex `json:"-"`
}

var statusFilePath = "/etc/protonet/system/configure-script-status"
var statusSocketPath = "/var/run/platconf-status.sock"

func updateStatusFromFile(status *StatusData, filePath string) error {
	var tempStatus StatusData
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&tempStatus)
	if err != nil {
		return err
	}

	status.Lock()
	defer status.Unlock()
	status.Status = tempStatus.Status
	status.Progress = tempStatus.Progress
	status.What = tempStatus.What
	return nil
}

func watchStatusFileForChange(status *StatusData, filePath string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to initalize a file watcher: %s", err.Error())
	}
	defer watcher.Close()

	err = watcher.Add(filePath)
	if err != nil {
		log.Fatalf("Failed to add file '%s' to the file watcher: %s", filePath, err.Error())
	}

	log.Println("Status file watcher started on", statusFilePath)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				err := updateStatusFromFile(status, statusFilePath)
				if err != nil {
					log.Println("ERROR: failed to read status from SKVS file:", err.Error())
				}
			}
		case err := <-watcher.Errors:
			log.Println("ERROR: failed while watching SKVS update status file:", err.Error())
		}
	}
}

func listenOnUnixSocket(status *StatusData, path string) error {
	os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}

	putStatusMux := http.NewServeMux()
	putStatusMux.HandleFunc("/status", func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != "PUT" {
			http.Error(rw, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}

		var tempStatus StatusData
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()

		err := decoder.Decode(&tempStatus)
		if err != nil {
			log.Println("ERROR: failed to decode status from UNIX domain socket", err.Error())
			http.Error(rw, "Couldn't decode status.", http.StatusBadRequest)
			return
		}

		status.Lock()
		defer status.Unlock()
		status.Status = tempStatus.Status
		status.Progress = tempStatus.Progress
		status.What = tempStatus.What

		http.Error(rw, "OK", http.StatusAccepted)
	})

	server := &http.Server{
		Handler: putStatusMux,
	}

	go server.Serve(listener)
	return nil
}

func getStatusReadMux(status *StatusData) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		status.RLock()
		defer status.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.Encode(&status)
	})

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found.", http.StatusNotFound)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlBody))
		log.Printf("Serving the status HTML page to '%s'", r.RemoteAddr)
	})

	return mux
}

// Execute is the function ran when the 'oldstatus' command is used
func (o *Opts) Execute(args []string) error {
	var status StatusData

	err := updateStatusFromFile(&status, statusFilePath)
	if err != nil {
		log.Printf("ERROR: failed to read status from SKVS file: %s", err.Error())
	} else {
		go watchStatusFileForChange(&status, statusFilePath)
	}

	server := &http.Server{
		Handler: getStatusReadMux(&status),
	}

	log.Println("Starting platform-install-status")
	// We're explicitly opening a listener first to check if there is already
	// an instance running. If we could bind to the port successfully then
	// we can also safely remove the UNIX domain socket.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", o.Port))
	if err != nil {
		return err
	}

	err = listenOnUnixSocket(&status, statusSocketPath)
	if err != nil {
		return err
	}

	err = server.Serve(listener)
	return err
}
