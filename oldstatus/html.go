package oldstatus

const htmlBody = `<html>
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
