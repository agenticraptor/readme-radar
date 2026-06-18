package github

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(h http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(h)
	return &Client{BaseURL: srv.URL, HTTP: srv.Client()}, srv
}

func TestRepo(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widget" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"full_name":"acme/widget","description":"  a widget  ","stargazers_count":1234,
			"forks_count":56,"open_issues_count":7,"language":"Go","default_branch":"main",
			"size":900,"archived":false,"license":{"spdx_id":"MIT"},
			"pushed_at":"2026-05-01T00:00:00Z","created_at":"2020-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	info, err := c.Repo(context.Background(), "acme", "widget")
	if err != nil {
		t.Fatal(err)
	}
	if info.Stars != 1234 || info.License != "MIT" || info.PrimaryLang != "Go" {
		t.Errorf("unexpected info: %+v", info)
	}
	if info.Description != "a widget" {
		t.Errorf("description not trimmed: %q", info.Description)
	}
}

func TestFetchFileBase64(t *testing.T) {
	raw := `{"name":"x"}`
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"type":"file","encoding":"base64","content":"` +
			base64.StdEncoding.EncodeToString([]byte(raw)) + `"}`))
	}))
	defer srv.Close()

	data, ok, err := c.FetchFile(context.Background(), "a", "b", "", "package.json")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if string(data) != raw {
		t.Errorf("got %q", data)
	}
}

func TestFetchFileMissing(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	_, ok, err := c.FetchFile(context.Background(), "a", "b", "", "nope.json")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if ok {
		t.Errorf("expected ok=false")
	}
}

func TestRateLimit(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "4070908800")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := c.Repo(context.Background(), "a", "b")
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
}

func TestLastPage(t *testing.T) {
	link := `<https://api.github.com/repositories/1/contributors?per_page=1&page=2>; rel="next", ` +
		`<https://api.github.com/repositories/1/contributors?per_page=1&page=42>; rel="last"`
	if n := lastPage(link); n != 42 {
		t.Errorf("lastPage = %d, want 42", n)
	}
	if n := lastPage(""); n != 0 {
		t.Errorf("lastPage empty = %d, want 0", n)
	}
}

func TestCountContributors(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", `<u?page=2>; rel="next", <u?page=17>; rel="last"`)
		_, _ = w.Write([]byte(`[{"login":"a"}]`))
	}))
	defer srv.Close()

	n, err := c.CountContributors(context.Background(), "a", "b")
	if err != nil || n != 17 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
