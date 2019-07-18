// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"code.golift.io/badgedata"
	_ "code.golift.io/badgedata/grafana"
	"google.golang.org/appengine"
)

func main() {
	var configPath string
	switch len(os.Args) {
	case 1:
		configPath = "config.yaml"
	case 2:
		configPath = os.Args[1]
	default:
		log.Fatal("usage: govanityurls [CONFIG]")
	}
	vanity, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	h, err := newHandler(vanity)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/bd/", badgedata.Handler())
	http.Handle("/", h)
	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":8080"
	}
	log.Println("Listening at http://127.0.0.1" + port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func defaultHost(r *http.Request) string {
	host := appengine.DefaultVersionHostname(appengine.NewContext(r))
	if host == "" {
		return r.Host
	}
	return host
}
