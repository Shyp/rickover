package server

import (
	"net/http"
	"os"

	"github.com/Shyp/rickover/config"
	"github.com/alecthomas/template"
)

var data = homepageData{
	Version: config.Version,
	URL:     os.Getenv("HOMEPAGE_IFRAME_URL"),
}

type homepageData struct {
	Version string
	URL     string
}

var homepagetemplate = `<!doctype html>
<html>
<head>
	<style>
	html, body, #dashboard {
		height: 100%;
	}
	body {
		font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif;
		margin: 0;
	}
	#dashboard {
		width: 100%;
	}
	#title {
		padding: 10px 5px;
		margin: 0;
	}
	</style>
</head>
<body>
	<h3 id="title">rickover version {{ .Version }}</h3>
	<iframe height="100%" width="100%" id="dashboard" src="{{ .URL }}">
</body>
</html>`

func renderHomepage(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.New("homepage").Parse(homepagetemplate))
	tpl.Execute(w, data)
}
