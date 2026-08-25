//nolint:gochecknoglobals,lll
package templates

import (
	"strconv"
	"strings"
	"text/template"
	"time"
)

// Funcs are template helpers shared by the HTML pages.
var Funcs = map[string]any{
	"TrimPrefix":  strings.TrimPrefix,
	"currentYear": func() string { return strconv.Itoa(time.Now().Year()) },
	// Add more if you need them.
}

// Index represents the index page.
var Index = template.Must(template.New("index").Funcs(Funcs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <title>{{.Title}} - {{.Host}}</title>
  <link rel='icon' href='/favicon.ico' type='image/x-icon'/ >
  <meta name="author" content="Copyright 2019-{{currentYear}} - {{.Title}}">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link href="https://fonts.googleapis.com/css?family=Raleway:400,300,600" rel="stylesheet" type="text/css">

  <!-- these are in static/css -->
  <link rel="stylesheet" href="https://docs.golift.io/css/normalize.css">
  <link rel="stylesheet" href="https://docs.golift.io/css/custom.css">
  <link rel="stylesheet" href="https://docs.golift.io/css/skeleton.css">
</head>
<body>
  <div class="container">
{{if .LogoURL }}
    <!-- header image -->
    <div class="row" style="margin-top: 10%">
      <img height="200px" src="{{.LogoURL}}">
    </div>
{{end}}
    <!-- header content -->
    <div class="row" style="margin-top: 5%">
      <div class="two-thirds column">
        <h1>{{.Host}} - {{.Title}}</h1>
        <p>{{.Description}}</p>
      </div>
      <div class="one-third column">
        <h4>Resources</h1>
{{- range .Links}}
        <li><a href="{{.URL}}">{{.Title}}</a></li>{{end}}
      </div>
    </div>

    <!-- package content -->
    <div class="value-props row">

      <div class="one-third column value-prop">
        <h5>Go Modules</h5>
        <ul>
{{- range .Paths}} {{if and .Repo (not .Wildcard) (not .Name)}}
          <li><a href="{{.Path}}">{{TrimPrefix .Path "/"}}</a></li>{{end}}{{- end}}
        </ul>
      </div>

      <div class="one-third column value-prop">
        <h5>Applications</h5>
        <ul>
	{{- range .Paths}} {{if .Name}}
          <li><a href="{{.Redir}}">{{.Name}}</a></li>{{end}}{{- end}}
        </ul>
      </div>

      <div class="one-third column value-prop">
        &copy; 2019-{{currentYear}} {{.Title}}<br>
{{if .Src}}        (<a href="{{.Src}}">source</a>){{end}}
      </div>
    </div>
  </div><!-- container class -->
</body>
</html>
`))

// GoGet is used for requests where go-get=1 is present.
var GoGet = template.Must(template.New("goget").Funcs(Funcs).Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
    <meta name="go-import" content="{{.Host}}{{.ImportPath}} {{.VCS}} {{.RepoPath}}"/>
    <meta name="go-source" content="{{.SourcePath}}"/>
    <meta http-equiv="refresh" content="0; url=https://{{.Host}}{{.ImportPath}}"/>
  </head>
</html>
`))

// Vanity is a nicely formatted css-using page for an import path.
var Vanity = template.Must(template.New("vanity").Funcs(Funcs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta http-equiv="content-type" content="text/html; charset=UTF-8">
  <title>Package {{.Title}} - {{.IndexTitle}}</title>
  <link rel="icon" href="/favicon.ico" type="image/x-icon"/>

  <meta name="go-import" content="{{.Host}}{{.ImportPath}} {{.VCS}} {{.RepoPath}}"/>
  <meta name="go-source" content="{{.SourcePath}}"/>
  <meta name="description" content="{{.RepoPath}}">
  <meta name="author" content="Copyright 2019-{{currentYear}} - {{.IndexTitle}}">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link href="https://fonts.googleapis.com/css?family=Raleway:400,300,600" rel="stylesheet" type="text/css">

  <!-- these are in static/css -->
  <link rel="stylesheet" href="https://docs.golift.io/css/normalize.css">
  <link rel="stylesheet" href="https://docs.golift.io/css/custom.css">
  <link rel="stylesheet" href="https://docs.golift.io/css/skeleton.css">
</head>
<body>
  <div class="container">
{{if .ImageURL}}
    <!-- header image -->
    <div class="row" style="margin-top: 10%">
      <img height="150px" src="{{.ImageURL}}">
    </div>
{{end}}
    <!-- main content -->
    <div class="row" style="margin-top: 5%">
      <div class="two-thirds column">
        <h1>{{.Host}}{{.ImportPath}}</h1>
        <p>{{.Description}}</p>
      </div>
{{- if .Links}}
      <!-- custom links -->
      <div class="one-third column">
        <h4>Resources</h1>
{{- range .Links}}
        <li><a href="{{.URL}}">{{.Title}}</a></li>{{end}}
      </div>{{end}}
    </div>

    <!-- built-in links -->
    <div class="value-props row">
      <div class="one-third column value-prop">
        <form action="https://godoc.org/{{.Host}}{{.ImportPath}}" method="get">
          <input type="button" class="button button-primary" onClick="window.location.href = 'https://godoc.org/{{.Host}}{{.ImportPath}}';" value="Documentation"/>
        </form>
      </div>
      <div class="one-third column value-prop">
        <form action="{{.RepoPath}}" method="get">
          <input type="button" class="button button-primary" onClick="window.location.href = '{{.RepoPath}}';" value="Code Repository"/>
        </form>
      </div>
    </div>

    <!-- usage & logo -->
    <div class="value-props row">
      <div class="two-thirds column value-prop">
        <p>Download this package.</p>
        <pre><code>go get {{.Host}}{{.ImportPath}}</code></pre>
        <p>Use this package.</p>
        <pre><code>import (
  "{{.Host}}{{.ImportPath}}"
)</code></pre>
        <p>Refer to the package as <code>{{.Title}}</code></p>
      </div>
      <div class="one-third column value-prop">
{{- if .LogoURL}}
        <a href="https://{{.Host}}"><img class="value-img" src="{{.LogoURL}}"></a>
{{- end}}
        <p>&copy; 2019-{{currentYear}} {{.IndexTitle}}<p>
      </div>
    </div>

  </div>
</body>
</html>`))
