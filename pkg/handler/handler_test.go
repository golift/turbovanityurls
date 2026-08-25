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

//nolint:funlen,noctx
package handler_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"golift.io/turbovanityurls/pkg/handler"
	yaml "gopkg.in/yaml.v3"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config string
		path   string

		goImport string
		goSource string
	}{
		{
			name: "explicit display",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/portmidi",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi _ _",
		},
		{
			name: "display GitHub inference",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://github.com/rakyll/portmidi\n",
			path:     "/portmidi",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi " +
				"https://github.com/rakyll/portmidi/tree/master{/dir} " +
				"https://github.com/rakyll/portmidi/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "Bitbucket Mercurial",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /gopdf:\n" +
				"    repo: https://bitbucket.org/zombiezen/gopdf\n" +
				"    vcs: hg\n",
			path:     "/gopdf",
			goImport: "example.com/gopdf hg https://bitbucket.org/zombiezen/gopdf",
			goSource: "example.com/gopdf https://bitbucket.org/zombiezen/gopdf " +
				"https://bitbucket.org/zombiezen/gopdf/src/default{/dir} " +
				"https://bitbucket.org/zombiezen/gopdf/src/default{/dir}/{file}#{file}-{line}",
		},
		{
			name: "Bitbucket Git",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /mygit:\n" +
				"    repo: https://bitbucket.org/zombiezen/mygit\n" +
				"    vcs: git\n",
			path:     "/mygit",
			goImport: "example.com/mygit git https://bitbucket.org/zombiezen/mygit",
			goSource: "example.com/mygit https://bitbucket.org/zombiezen/mygit " +
				"https://bitbucket.org/zombiezen/mygit/src/default{/dir} " +
				"https://bitbucket.org/zombiezen/mygit/src/default{/dir}/{file}#{file}-{line}",
		},
		{
			name: "subpath",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/portmidi/foo",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi _ _",
		},
		{
			name: "subpath with trailing config slash",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi/:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/portmidi/foo",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi _ _",
		},
		{
			name: "root path",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/foo/foo",
			goImport: "example.com git https://github.com/rakyll/portmidi",
			goSource: "example.com https://github.com/rakyll/portmidi _ _",
		},
		{
			name: "wildcard with sub path",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /rakyll/:\n" +
				"    repo: https://github.com/rakyll/\n" +
				"    wildcard: true\n",
			path:     "/rakyll/repo/foo",
			goImport: "example.com/rakyll/repo git https://github.com/rakyll/repo",
			goSource: "example.com/rakyll/repo https://github.com/rakyll/repo " +
				"https://github.com/rakyll/repo/tree/master{/dir} " +
				"https://github.com/rakyll/repo/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "wildcard with no slashes",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /rakyll/:\n" +
				"    repo: https://github.com/rakyll/\n" +
				"    wildcard: true\n",
			path:     "/rakyll/repo",
			goImport: "example.com/rakyll/repo git https://github.com/rakyll/repo",
			goSource: "example.com/rakyll/repo https://github.com/rakyll/repo " +
				"https://github.com/rakyll/repo/tree/master{/dir} https://github.com/rakyll/repo/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "wildcard with dashes",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /rakyll-:\n" +
				"    repo: https://github.com/rakyll/\n" +
				"    wildcard: true\n",
			path:     "/rakyll-repo",
			goImport: "example.com/rakyll-repo git https://github.com/rakyll/repo",
			goSource: "example.com/rakyll-repo https://github.com/rakyll/repo " +
				"https://github.com/rakyll/repo/tree/master{/dir} https://github.com/rakyll/repo/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "wildcard bare word",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /rakyll:\n" +
				"    repo: https://github.com/rakyll/\n" +
				"    cache_max_age: 99\n" +
				"    wildcard: true\n",
			path:     "/rakyllrepo",
			goImport: "example.com/rakyllrepo git https://github.com/rakyll/repo",
			goSource: "example.com/rakyllrepo https://github.com/rakyll/repo " +
				"https://github.com/rakyll/repo/tree/master{/dir} https://github.com/rakyll/repo/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "wildcard dashed repo",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /group/:\n" +
				"    repo: https://github.com/rakyll/package-group-\n" +
				"    wildcard: true\n",
			path:     "/group/foo",
			goImport: "example.com/group/foo git https://github.com/rakyll/package-group-foo",
			goSource: "example.com/group/foo https://github.com/rakyll/package-group-foo " +
				"https://github.com/rakyll/package-group-foo/tree/master{/dir} " +
				"https://github.com/rakyll/package-group-foo/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "display Gitlab inference",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://gitlab.com/rakyll/portmidi\n",
			path:     "/portmidi",
			goImport: "example.com/portmidi git https://gitlab.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://gitlab.com/rakyll/portmidi " +
				"https://gitlab.com/rakyll/portmidi/tree/master{/dir} " +
				"https://gitlab.com/rakyll/portmidi/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "display Gitlab inference",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    vcs: gitlab\n" +
				"    repo: https://gitlab.com/rakyll/portmidi\n",
			path:     "/portmidi",
			goImport: "example.com/portmidi git https://gitlab.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://gitlab.com/rakyll/portmidi " +
				"https://gitlab.com/rakyll/portmidi/tree/master{/dir} " +
				"https://gitlab.com/rakyll/portmidi/blob/master{/dir}/{file}#L{line}",
		},
	}

	for _, test := range tests {
		h, err := handler.New(getTestConfig([]byte(test.config)))
		if err != nil {
			t.Errorf("%s: New: %v", test.name, err)
			continue
		}

		s := httptest.NewServer(h)

		resp, err := http.Get(s.URL + test.path)
		if err != nil {
			s.Close()
			t.Errorf("%s: http.Get: %v", test.name, err)

			continue
		}

		data, err := io.ReadAll(resp.Body)

		_ = resp.Body.Close()
		s.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status code = %s; want 200 OK", test.name, resp.Status)
		}

		if err != nil {
			t.Errorf("%s: io.ReadAll: %v", test.name, err)
			continue
		}

		if got := findMeta(data, "go-import"); got != test.goImport {
			t.Errorf("%s: meta go-import = %q; want %q", test.name, got, test.goImport)
		}

		if got := findMeta(data, "go-source"); got != test.goSource {
			t.Errorf("%s: meta go-source = %q; want %q", test.name, got, test.goSource)
		}
	}
}

func TestBadConfigs(t *testing.T) {
	t.Parallel()

	badConfigs := []string{
		"host: example.com\npaths:\n" +
			"  /missingvcs:\n" +
			"    repo: https://unknownbucket.org/zombiezen/gopdf\n",
		"host: example.com\npaths:\n" +
			"  /unknownvcs:\n" +
			"    repo: https://unknownbucket.org/zombiezen/gopdf\n" +
			"    vcs: xyzzy\n",
		"paths:\n" +
			"  /missinghost:\n" +
			"    repo: https://github.com/zombiezen/gopdf\n" +
			"    vcs: git\n",
	}

	for _, config := range badConfigs {
		_, err := handler.New(getTestConfig([]byte(config)))
		if err == nil {
			t.Errorf("expected config to produce an error, but did not:\n%s", config)
		}
	}
}

func getTestConfig(data []byte) *handler.Config {
	c := &handler.Config{}
	_ = yaml.Unmarshal(data, c)

	return c
}

func findMeta(data []byte, name string) string {
	sep := make([]byte, 0, len(name)+11)

	sep = append(sep, `<meta name="`...)
	sep = append(sep, name...)
	sep = append(sep, `" content="`...)

	i := bytes.Index(data, sep)
	if i == -1 {
		return ""
	}

	content := data[i+len(sep):]

	before, _, ok := bytes.Cut(content, []byte{'"'})
	if !ok {
		return ""
	}

	return string(before)
}

func TestPathConfigSetFind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		paths   []string
		query   string
		want    string
		subpath string
	}{
		{
			paths: []string{"/"},
			query: "/",
			want:  "/",
		},
		{
			paths: []string{"/portmidi"},
			query: "/portmidi",
			want:  "/portmidi",
		},
		{
			paths: []string{"/portmidi"},
			query: "/portmidi/",
			want:  "/portmidi",
		},
		{
			paths: []string{"/portmidi"},
			query: "/foo",
			want:  "",
		},
		{
			paths: []string{"/portmidi"},
			query: "/zzz",
			want:  "",
		},
		{
			paths: []string{"/abc", "/portmidi", "/xyz"},
			query: "/portmidi",
			want:  "/portmidi",
		},
		{
			paths:   []string{"/abc", "/portmidi", "/xyz"},
			query:   "/portmidi/foo",
			want:    "/portmidi",
			subpath: "foo",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/x",
			want:    "/",
			subpath: "x",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/",
			want:    "/",
			subpath: "",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/example",
			want:    "/",
			subpath: "example",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/example/foo",
			want:    "/",
			subpath: "example/foo",
		},
		{
			paths: []string{"/example/helloworld", "/", "/y", "/foo"},
			query: "/y",
			want:  "/y",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/x/y/",
			want:    "/",
			subpath: "x/y/",
		},
		{
			paths: []string{"/example/helloworld", "/y", "/foo"},
			query: "/x",
			want:  "",
		},
	}
	emptyToNil := func(s string) string {
		if s == "" {
			return "<nil>"
		}

		return s
	}

	for _, test := range tests {
		pset := make(handler.PathConfigs, len(test.paths))
		for i := range test.paths {
			pset[i] = &handler.PathConfig{Path: test.paths[i]}
		}

		var got string

		sort.Sort(pset)

		pc := pset.Find(test.query)
		if pc.PathConfig != nil {
			got = pc.Path
		}

		if got != test.want || pc.Subpath != test.subpath {
			t.Errorf("pathConfigSet(%v).find(%q) = %v, %v; want %v, %v",
				test.paths, test.query, emptyToNil(got), pc.Subpath, emptyToNil(test.want), test.subpath)
		}
	}
}

func TestCacheHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       string
		cacheControl string
	}{
		{
			name:         "default",
			cacheControl: "public, max-age=86400",
		},
		{
			name:         "specify time",
			config:       "cache_max_age: 60\n",
			cacheControl: "public, max-age=60",
		},
		{
			name:         "zero",
			config:       "cache_max_age: 0\n",
			cacheControl: "public, max-age=0",
		},
	}

	for _, test := range tests {
		h, err := handler.New(getTestConfig([]byte("host: example.com\npaths:\n  /portmidi:\n" +
			"    repo: https://github.com/rakyll/portmidi\n" +
			test.config)))
		if err != nil {
			t.Errorf("%s: New: %v", test.name, err)
			continue
		}

		s := httptest.NewServer(h)

		resp, err := http.Get(s.URL + "/portmidi")
		if err != nil {
			t.Errorf("%s: http.Get: %v", test.name, err)
			continue
		}

		_ = resp.Body.Close()

		got := resp.Header.Get("Cache-Control")
		if got != test.cacheControl {
			t.Errorf("%s: Cache-Control header = %q; want %q", test.name, got, test.cacheControl)
		}
	}
}

func TestServeHTTPRedirect(t *testing.T) {
	t.Parallel()

	h, err := handler.New(getTestConfig([]byte(`
host: example.com
redir_paths: ["releases"]
paths:
  /portmidi:
    repo: https://github.com/rakyll/portmidi
    redir: https://github.com/rakyll/portmidi
`)))
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/portmidi/releases", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	want := "https://github.com/rakyll/portmidi/releases"
	if loc := rec.Header().Get("Location"); loc != want {
		t.Fatalf("Location = %q, want %q", loc, want)
	}
}
