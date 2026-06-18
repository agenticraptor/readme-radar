// Package github is a tiny, dependency-free REST client for the handful of
// GitHub endpoints readme-radar needs. It speaks plain HTTP+JSON rather than
// pulling in an SDK, works unauthenticated (subject to the 60 req/hr limit),
// and uses GITHUB_TOKEN / GH_TOKEN when present to lift that to 5,000/hr.
package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/agenticraptor/readme-radar/internal/report"
	"github.com/agenticraptor/readme-radar/internal/textutil"
)

// ErrNotFound means the repository or path does not exist (HTTP 404).
var ErrNotFound = errors.New("not found")

// RateLimitError is returned when GitHub rejects a request for rate-limit
// reasons. Setting GITHUB_TOKEN raises the limit substantially.
type RateLimitError struct {
	ResetAt time.Time
}

func (e *RateLimitError) Error() string {
	if e.ResetAt.IsZero() {
		return "github API rate limit exceeded — set GITHUB_TOKEN to raise it"
	}
	return fmt.Sprintf("github API rate limit exceeded (resets %s) — set GITHUB_TOKEN to raise it",
		e.ResetAt.Format(time.Kitchen))
}

// Client talks to the GitHub REST API.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New returns a Client configured from the environment.
func New() *Client {
	return &Client{
		BaseURL: "https://api.github.com",
		Token:   firstEnv("GITHUB_TOKEN", "GH_TOKEN"),
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Authenticated reports whether a token is configured.
func (c *Client) Authenticated() bool { return c.Token != "" }

type repoPayload struct {
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	Homepage    string    `json:"homepage"`
	Stars       int       `json:"stargazers_count"`
	Forks       int       `json:"forks_count"`
	OpenIssues  int       `json:"open_issues_count"`
	Language    string    `json:"language"`
	Topics      []string  `json:"topics"`
	Archived    bool      `json:"archived"`
	Disabled    bool      `json:"disabled"`
	Fork        bool      `json:"fork"`
	Default     string    `json:"default_branch"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	PushedAt    time.Time `json:"pushed_at"`
	License     *struct {
		SPDX string `json:"spdx_id"`
		Name string `json:"name"`
	} `json:"license"`
}

// Repo fetches the primary repository metadata.
func (c *Client) Repo(ctx context.Context, owner, repo string) (report.RepoInfo, error) {
	var p repoPayload
	if _, err := c.getJSON(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), &p); err != nil {
		return report.RepoInfo{}, err
	}
	info := report.RepoInfo{
		FullName:      p.FullName,
		Description:   textutil.Safe(p.Description),
		Homepage:      textutil.Safe(p.Homepage),
		Stars:         p.Stars,
		Forks:         p.Forks,
		OpenIssues:    p.OpenIssues,
		PrimaryLang:   p.Language,
		Topics:        p.Topics,
		Archived:      p.Archived,
		Disabled:      p.Disabled,
		Fork:          p.Fork,
		DefaultBranch: p.Default,
		SizeKB:        p.Size,
		CreatedAt:     p.CreatedAt,
		PushedAt:      p.PushedAt,
		License:       "none",
	}
	if p.License != nil && p.License.SPDX != "" && p.License.SPDX != "NOASSERTION" {
		info.License = p.License.SPDX
	}
	return info, nil
}

// Languages returns the repository's language breakdown, largest first.
func (c *Client) Languages(ctx context.Context, owner, repo string) ([]report.LangSize, error) {
	var m map[string]int64
	if _, err := c.getJSON(ctx, fmt.Sprintf("/repos/%s/%s/languages", owner, repo), &m); err != nil {
		return nil, err
	}
	out := make([]report.LangSize, 0, len(m))
	for name, b := range m {
		out = append(out, report.LangSize{Name: name, Bytes: b})
	}
	// simple descending sort by bytes
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Bytes > out[j-1].Bytes; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

// CountContributors returns an approximate contributor count using the
// Link-header pagination trick (cheap: one request).
func (c *Client) CountContributors(ctx context.Context, owner, repo string) (int, error) {
	return c.countViaLink(ctx, fmt.Sprintf("/repos/%s/%s/contributors?per_page=1&anon=true", owner, repo))
}

type releasePayload struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
}

// LatestRelease returns the most recent release tag and date, plus the total
// number of releases. A repo with no releases yields ("", zero-time, 0, nil).
func (c *Client) LatestRelease(ctx context.Context, owner, repo string) (tag string, at time.Time, total int, err error) {
	var rels []releasePayload
	hdr, err := c.getJSON(ctx, fmt.Sprintf("/repos/%s/%s/releases?per_page=1", owner, repo), &rels)
	if err != nil {
		return "", time.Time{}, 0, err
	}
	total = lastPage(hdr.Get("Link"))
	if total == 0 {
		total = len(rels)
	}
	if len(rels) > 0 {
		return rels[0].TagName, rels[0].PublishedAt, total, nil
	}
	return "", time.Time{}, total, nil
}

type contentPayload struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Size     int    `json:"size"`
}

// FetchFile returns the decoded contents of a file at an optional ref. The
// boolean is false (with a nil error) when the file simply does not exist.
func (c *Client) FetchFile(ctx context.Context, owner, repo, ref, path string) ([]byte, bool, error) {
	u := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" {
		u += "?ref=" + url.QueryEscape(ref)
	}
	var p contentPayload
	_, err := c.getJSON(ctx, u, &p)
	if errors.Is(err, ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if p.Type != "file" {
		return nil, false, nil
	}
	if p.Encoding == "base64" {
		data, decErr := base64.StdEncoding.DecodeString(strings.ReplaceAll(p.Content, "\n", ""))
		if decErr != nil {
			return nil, false, fmt.Errorf("decode %s: %w", path, decErr)
		}
		return data, true, nil
	}
	return []byte(p.Content), true, nil
}

// --- internals ---

func (c *Client) countViaLink(ctx context.Context, path string) (int, error) {
	var discard json.RawMessage
	hdr, err := c.getJSON(ctx, path, &discard)
	if err != nil {
		return 0, err
	}
	if n := lastPage(hdr.Get("Link")); n > 0 {
		return n, nil
	}
	// No Link header → at most one page; count the array length.
	var arr []json.RawMessage
	if json.Unmarshal(discard, &arr) == nil {
		return len(arr), nil
	}
	return 0, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "readme-radar")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return resp.Header, ErrNotFound
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusTooManyRequests:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return resp.Header, &RateLimitError{ResetAt: resetTime(resp.Header)}
		}
		return resp.Header, fmt.Errorf("github: %s", resp.Status)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return resp.Header, fmt.Errorf("github: %s: %s", resp.Status, snippet(body))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return resp.Header, fmt.Errorf("decode %s: %w", path, err)
		}
	}
	return resp.Header, nil
}

var lastPageRe = regexp.MustCompile(`[?&]page=(\d+)[^>]*>;\s*rel="last"`)

// lastPage extracts the rel="last" page number from a Link header, or 0.
func lastPage(link string) int {
	if link == "" {
		return 0
	}
	if m := lastPageRe.FindStringSubmatch(link); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n
		}
	}
	return 0
}

func resetTime(h http.Header) time.Time {
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if sec, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(sec, 0)
		}
	}
	return time.Time{}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
