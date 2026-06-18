package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNPM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/express/latest" {
			t.Errorf("path = %q, want /express/latest", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
		  "name":"express","version":"5.0.0","description":"web framework",
		  "repository":{"type":"git","url":"git+https://github.com/expressjs/express.git"},
		  "license":"MIT",
		  "dependencies":{"accepts":"^1","cookie":"^0.6"},
		  "scripts":{"test":"mocha","postinstall":"node setup.js"},
		  "dist":{"unpackedSize":210000,"fileCount":16}
		}`))
	}))
	defer srv.Close()

	c := &Client{NPMBase: srv.URL, HTTP: srv.Client()}
	res, err := c.NPM(context.Background(), "express")
	if err != nil {
		t.Fatal(err)
	}
	if res.Version != "5.0.0" || res.InstallBytes != 210000 || len(res.DirectDeps) != 2 {
		t.Errorf("unexpected: %+v", res)
	}
	if res.Owner != "expressjs" || res.Repo != "express" {
		t.Errorf("repo parse: %s/%s", res.Owner, res.Repo)
	}
	if _, ok := res.Scripts["postinstall"]; !ok {
		t.Errorf("expected postinstall script, got %v", res.Scripts)
	}
	if _, ok := res.Scripts["test"]; ok {
		t.Errorf("non-lifecycle script leaked: %v", res.Scripts)
	}
	if res.License != "MIT" {
		t.Errorf("license = %q", res.License)
	}
}

func TestPyPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "info":{
		    "version":"2.32.0","summary":"HTTP for Humans","license":"Apache-2.0",
		    "home_page":"https://requests.readthedocs.io",
		    "project_urls":{"Source":"https://github.com/psf/requests"},
		    "requires_dist":["idna (>=2.5)","certifi","pytest ; extra == 'test'"]
		  },
		  "urls":[
		    {"packagetype":"sdist","size":100000},
		    {"packagetype":"bdist_wheel","size":62000}
		  ]
		}`))
	}))
	defer srv.Close()

	c := &Client{PyPIBase: srv.URL, HTTP: srv.Client()}
	res, err := c.PyPI(context.Background(), "requests")
	if err != nil {
		t.Fatal(err)
	}
	if res.InstallBytes != 62000 { // wheel preferred
		t.Errorf("size = %d", res.InstallBytes)
	}
	if len(res.DirectDeps) != 2 { // pytest excluded (extra)
		t.Errorf("deps = %v", res.DirectDeps)
	}
	if res.Owner != "psf" || res.Repo != "requests" {
		t.Errorf("repo: %s/%s", res.Owner, res.Repo)
	}
}

func TestNPMDeprecated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "version":"2.88.2",
		  "deprecated":"request has been deprecated, see #3142",
		  "dist":{"unpackedSize":200000}
		}`))
	}))
	defer srv.Close()
	c := &Client{NPMBase: srv.URL, HTTP: srv.Client()}
	res, err := c.NPM(context.Background(), "request")
	if err != nil {
		t.Fatal(err)
	}
	if res.Deprecated == "" || !strings.Contains(res.Deprecated, "deprecated") {
		t.Errorf("expected deprecation message, got %q", res.Deprecated)
	}
}

func TestDeprecatedMsg(t *testing.T) {
	if deprecatedMsg([]byte(`"use foo"`)) != "use foo" {
		t.Error("string message")
	}
	if deprecatedMsg([]byte(`true`)) == "" {
		t.Error("bool true should be non-empty")
	}
	if deprecatedMsg([]byte(`false`)) != "" || deprecatedMsg([]byte(`null`)) != "" || deprecatedMsg(nil) != "" {
		t.Error("false/null/empty should be empty")
	}
}

func TestRepoFrom(t *testing.T) {
	cases := map[string]string{
		"git+https://github.com/a/b.git":   "a/b",
		"git://github.com/a/b.git":         "a/b",
		"https://github.com/a/b":           "a/b",
		"git+ssh://git@github.com/a/b.git": "a/b",
	}
	for in, want := range cases {
		_, o, r := repoFrom(in)
		if o+"/"+r != want {
			t.Errorf("repoFrom(%q) = %s/%s, want %s", in, o, r, want)
		}
	}
}
