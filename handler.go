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

// govanityurls serves Go vanity URLs.
package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"google.golang.org/appengine"
	yaml "gopkg.in/yaml.v2"
)

// Handler contains all the running data for our web server.
type Handler struct {
	*Config
	PathConfigs
}

// vcsPrefixMap provides defaults for VCS type if it's not provided.
// The list of strings is used in strings.HasPrefix().
var vcsPrefixMap = map[string][]string{
	"git": []string{"https://git", "https://bitbucket"},
	"bzr": []string{"https://bazaar"},
	"hg":  []string{"https://hg.", "https://mercurial"},
	"svn": []string{"https://svn."},
}

// Config contains the config file data.
type Config struct {
	Title string `yaml:"title,omitempty"`
	Host  string `yaml:"host,omitempty"`
	Links []struct {
		Title string `yaml:"title,omitempty"`
		URL   string `yaml:"url,omitempty"`
	} `yaml:"links,omitempty"`
	CacheAge   *uint64                `yaml:"cache_max_age,omitempty"`
	Paths      map[string]*PathConfig `yaml:"paths,omitempty"`
	RedirPaths []string               `yaml:"redir_paths,omitempty"`
	Src        string                 `yaml:"src,omitempty"`
}

// PathConfigs contains our list of configured routing-paths.
type PathConfigs []*PathConfig

// PathConfig is the configuration for a single routing path.
type PathConfig struct {
	Path         string   `yaml:"-"`
	CacheAge     *uint64  `yaml:"cache_max_age,omitempty"`
	RedirPaths   []string `yaml:"redir_paths,omitempty"`
	Repo         string   `yaml:"repo,omitempty"`
	Redir        string   `yaml:"redir,omitempty"`
	Display      string   `yaml:"display,omitempty"`
	VCS          string   `yaml:"vcs,omitempty"`
	cacheControl string
}

// PathReq is returned by find() with a non-nil PathConfig
// when a request has been matched to a path. Host comes unset.
// This struct is passed into the vanity template.
type PathReq struct {
	Host    string
	Subpath string
	*PathConfig
}

func newHandler(configData []byte) (*Handler, error) {
	h := &Handler{Config: &Config{Paths: make(map[string]*PathConfig)}}
	if err := yaml.Unmarshal(configData, h.Config); err != nil {
		return nil, err
	}
	if h.Title == "" {
		h.Title = h.Host
	}
	cacheControl := fmt.Sprintf("public, max-age=86400") // 24 hours (in seconds)
	if h.CacheAge != nil {
		cacheControl = fmt.Sprintf("public, max-age=%d", *h.CacheAge)
	}
	for p := range h.Paths {
		h.Paths[p].Path = p
		if len(h.Paths[p].RedirPaths) < 1 {
			// was not provided, pass in global value.
			h.Paths[p].RedirPaths = h.RedirPaths
		}
		h.Paths[p].setRepoCacheControl(cacheControl)
		h.Paths[p].setRepoDisplay()
		if err := h.Paths[p].setRepoVCS(); err != nil {
			return nil, err
		}
		h.PathConfigs = append(h.PathConfigs, h.Paths[p])
	}
	sort.Sort(h.PathConfigs)
	return h, nil
}

func (p *PathConfig) setRepoCacheControl(cc string) {
	p.cacheControl = cc
	if p.CacheAge != nil {
		// provided, override global value.
		p.cacheControl = fmt.Sprintf("public, max-age=%d", *p.CacheAge)
	}
}

// Set display path.
func (p *PathConfig) setRepoDisplay() {
	if p.Display != "" {
		return
	}
	// github, gitlab, git, svn, hg, bzr - may need more tweaking for some of these.
	p.Display = fmt.Sprintf("%v %v/tree/master{/dir} %v/blob/master{/dir}/{file}#L{line}", p.Repo, p.Repo, p.Repo)
	if strings.HasPrefix(p.Repo, "https://bitbucket.org") {
		// bitbucket is weird.
		p.Display = fmt.Sprintf("%v %v/src/default{/dir} %v/src/default{/dir}/{file}#{file}-{line}", p.Repo, p.Repo, p.Repo)
	}
}

// setRepoVCS makes sure the provides VCS type is supported,
// or sets it automatically based on the repo's prefix.
func (p *PathConfig) setRepoVCS() error {
	// Check and set VCS type.
	switch {
	case p.Repo == "" && p.Redir != "":
		// Redirect-only can go anywhere.
	case p.VCS == "github" || p.VCS == "gitlab" || p.VCS == "bitbucket":
		p.VCS = "git"
	case p.VCS == "":
		// Try to figure it out.
		var err error
		p.VCS, err = findRepoVCS(p.Repo)
		return err
	default:
		// Already filled in, make sure it's supported.
		if _, ok := vcsPrefixMap[p.VCS]; !ok {
			return fmt.Errorf("configuration for %v: unknown VCS %s", p, p.VCS)
		}
	}
	return nil
}

// findRepoVCS checks the vcsMapList for a supported vcs type based on a repo's prefix.
func findRepoVCS(repo string) (string, error) {
	for vcs, prefixList := range vcsPrefixMap {
		for _, prefix := range prefixList {
			if strings.HasPrefix(repo, prefix) {
				return vcs, nil
			}
		}
	}
	return "", fmt.Errorf("cannot infer VCS from %s", repo)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pc := h.PathConfigs.find(r.URL.Path)
	if pc.PathConfig == nil {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		} else if err := indexTmpl.Execute(w, &h.Config); err != nil {
			http.Error(w, "cannot render the page", http.StatusInternalServerError)
		}
		return
	}

	// Redirect for file downloads.
	if pc.Redir != "" && StringInSlices(pc.Subpath, pc.RedirPaths) {
		redirTo := pc.Redir + strings.TrimPrefix(r.URL.Path, pc.Path)
		http.Redirect(w, r, redirTo, 302)
		return
	}
	if pc.Repo == "" {
		// Repo is not set and no paths to redirect, so we're done.
		http.NotFound(w, r)
		return
	}

	// Create a vanity redirect page.
	w.Header().Set("Cache-Control", pc.cacheControl)
	pc.Host = h.Hostname(r)
	pc.Path = strings.TrimSuffix(pc.Path, "/")
	if err := vanityTmpl.Execute(w, pc); err != nil {
		http.Error(w, "cannot render the page", http.StatusInternalServerError)
	}
}

// Hostname returns the appropriate Host header for this request.
func (h *Handler) Hostname(r *http.Request) string {
	if h.Host != "" {
		return h.Host
	}
	appHost := appengine.DefaultVersionHostname(appengine.NewContext(r))
	if appHost == "" {
		return r.Host
	}
	return appHost
}

// StringInSlices checks if a string exists in a list of strings.
// Used to determine if a sub path should be redirected or not.
// Not used for normal vanity URLs, only used for `redir`.
func StringInSlices(str string, slice []string) bool {
	for _, s := range slice {
		if strings.Contains(str, s) {
			return true
		}
	}
	return false
}

// Len is a sort.Search interface method.
func (pset PathConfigs) Len() int {
	return len(pset)
}

// Less is a sort.Search interface method.
func (pset PathConfigs) Less(i, j int) bool {
	return pset[i].Path < pset[j].Path
}

// Swap is a sort.Search interface method.
func (pset PathConfigs) Swap(i, j int) {
	pset[i], pset[j] = pset[j], pset[i]
}

func (pset PathConfigs) find(path string) PathReq {
	// Fast path with binary search to retrieve exact matches
	// e.g. given pset ["/", "/abc", "/xyz"], path "/def" won't match.
	i := sort.Search(len(pset), func(i int) bool {
		return pset[i].Path >= path
	})
	if i < len(pset) && pset[i].Path == path {
		return PathReq{PathConfig: pset[i]}
	}
	if i > 0 && strings.HasPrefix(path, pset[i-1].Path+"/") {
		return PathReq{
			PathConfig: pset[i-1],
			Subpath:    path[len(pset[i-1].Path)+1:],
		}
	}

	// Slow path, now looking for the longest prefix/shortest subpath i.e.
	// e.g. given pset ["/", "/abc/", "/abc/def/", "/xyz"/]
	//  * query "/abc/foo" returns "/abc/" with a subpath of "foo"
	//  * query "/x" returns "/" with a subpath of "x"
	lenShortestSubpath := len(path)
	var p PathReq

	// After binary search with the >= lexicographic comparison,
	// nothing greater than i will be a prefix of path.
	max := i
	for i := 0; i < max; i++ {
		ps := pset[i]
		if len(ps.Path) >= len(path) {
			// We previously didn't find the path by search, so any
			// route with equal or greater length is NOT a match.
			continue
		}
		sSubpath := strings.TrimPrefix(path, ps.Path)
		if len(sSubpath) < lenShortestSubpath {
			p.Subpath = sSubpath
			lenShortestSubpath = len(sSubpath)
			p.PathConfig = pset[i]
		}
	}
	return p
}
