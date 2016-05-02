package server

import (
	"html/template"
	"net/http"
	"os"

	"github.com/Shyp/rickover/config"
)

var data homepageData

func init() {
	data = homepageData{
		Version: config.Version,
		URL:     os.Getenv("HOMEPAGE_IFRAME_URL"),
	}
	title := os.Getenv("HOMEPAGE_TITLE")
	if title == "" {
		title = "Rickover Dashboard"
	}
	data.Title = title
}

type homepageData struct {
	Version string
	URL     string
	Title   string
}

var homepagetemplate = `<!doctype html>
<html>
<head>
	<title>{{ .Title }}</title>
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
