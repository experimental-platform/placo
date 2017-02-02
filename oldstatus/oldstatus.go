package oldstatus

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Opts contains command line parameters for the 'oldstatus' command
type Opts struct {
	Port int `short:"p" long:"port" description:"Port on which to listen" default:"7887"`
}

var statusFilePath = "/etc/protonet/system/configure-script-status"

// Execute is the function ran when the 'oldstatus' command is used
func (o *Opts) Execute(args []string) error {
	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile(statusFilePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found.", http.StatusNotFound)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlBody))
		log.Printf("Serving the status HTML page to '%s'", r.RemoteAddr)
	})

	log.Println("Starting platform-install-status")
	err := http.ListenAndServe(fmt.Sprintf(":%d", o.Port), nil)
	return err
}
