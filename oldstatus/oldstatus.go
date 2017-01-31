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

const statusFilePath = "/etc/protonet/system/configure-script-status"

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
		htmlBody := `<html>
	<head>
	<title>⬢ Protonet SOUL installation/update status</title>
	</head>
	<body>
	<h1>⬢ Protonet SOUL</h1>
	<h4>Installation/update status</h4>
	<div id="status_text"></div>
	<script>
	function getStatusObject() {
	  var request = new XMLHttpRequest();
	  request.open("GET", "/json", false);
	  request.send();

	  if (request.status == 200) {
			return JSON.parse(request.responseText);
		} else {
			return null;
		}
	}

	function loadStatus() {
	  var statusObject = getStatusObject();

	  document.getElementById("status_text").innerHTML = "Update status: ";
		if (statusObject == null) {
			document.getElementById("status_text").innerHTML += '<span style="color: #EE0000;">unknown</span>';
		} else {
			var status = statusObject['status'];
			var progress = statusObject['progress'];
			var what = statusObject['what'];

			document.getElementById("status_text").innerHTML += status + "<br />";
			if (progress != null) {
				document.getElementById("status_text").innerHTML += "Download progress: " + progress.toFixed(1) + "%<br />";
			}
			if (what != null) {
				document.getElementById("status_text").innerHTML += "Currently downloading: '" + what + "'<br />";
			}
		}
	}

	setInterval(loadStatus, 500);
	</script>
	</body>
	</html>
	`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlBody))
		log.Printf("Serving the status HTML page to '%s'", r.RemoteAddr)
	})

	log.Println("Starting platform-install-status")
	err := http.ListenAndServe(fmt.Sprintf(":%d", o.Port), nil)
	return err
}
