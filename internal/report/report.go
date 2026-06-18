// Package report defines the shared vocabulary of a readme-radar scan: the
// target being analyzed, the per-axis findings, the combined score, and the
// final narrated verdict. Every other internal package speaks in these types,
// which keeps the analysis pipeline a clean one-way dependency graph.
package report

import "time"

// Ecosystem identifies where a target lives.
type Ecosystem string

const (
	EcosystemGitHub Ecosystem = "github"
	EcosystemNPM    Ecosystem = "npm"
	EcosystemPyPI   Ecosystem = "pypi"
)

// Level is a coarse traffic-light rating used by individual findings.
type Level int

const (
	LevelInfo Level = iota // neutral, informational
	LevelGood              // a reassuring signal
	LevelWarn              // worth a second look
	LevelBad               // a genuine red flag
)

// String renders a Level as a lowercase word, handy for JSON and tests.
func (l Level) String() string {
	switch l {
	case LevelGood:
		return "good"
	case LevelWarn:
		return "warn"
	case LevelBad:
		return "bad"
	default:
		return "info"
	}
}

// Symbol returns a single glyph for a Level (used by the terminal renderer).
func (l Level) Symbol() string {
	switch l {
	case LevelGood:
		return "✓"
	case LevelWarn:
		return "!"
	case LevelBad:
		return "✗"
	default:
		return "·"
	}
}

// Verdict is the headline recommendation derived from the overall score.
type Verdict string

const (
	VerdictTrust   Verdict = "looks trustworthy"
	VerdictCaution Verdict = "proceed with caution"
	VerdictAvoid   Verdict = "red flags — look closely"
)

// Target is the resolved thing being analyzed.
type Target struct {
	Raw       string    `json:"raw"`             // exactly what the user typed
	Ecosystem Ecosystem `json:"ecosystem"`       // github | npm | pypi
	Owner     string    `json:"owner,omitempty"` // GitHub owner/org
	Repo      string    `json:"repo,omitempty"`  // GitHub repository name
	Ref       string    `json:"ref,omitempty"`   // optional branch/tag/commit
	Package   string    `json:"package,omitempty"`
	URL       string    `json:"url"` // canonical https URL
}

// Slug is the "owner/repo" identifier, or the package name when there is no repo.
func (t Target) Slug() string {
	if t.Owner != "" && t.Repo != "" {
		return t.Owner + "/" + t.Repo
	}
	if t.Package != "" {
		return t.Package
	}
	return t.Raw
}

// Factor is one named observation within an analysis axis.
type Factor struct {
	Name   string `json:"name"`
	Level  Level  `json:"-"`
	Status string `json:"status"` // mirrors Level.String() in JSON
	Detail string `json:"detail"`
}

// RepoInfo holds the GitHub facts the rest of the pipeline reasons about.
type RepoInfo struct {
	FullName      string     `json:"full_name"`
	Description   string     `json:"description"`
	Homepage      string     `json:"homepage,omitempty"`
	Stars         int        `json:"stars"`
	Forks         int        `json:"forks"`
	OpenIssues    int        `json:"open_issues"`
	License       string     `json:"license"`
	PrimaryLang   string     `json:"primary_language"`
	Languages     []LangSize `json:"languages,omitempty"`
	Topics        []string   `json:"topics,omitempty"`
	Archived      bool       `json:"archived"`
	Disabled      bool       `json:"disabled"`
	Fork          bool       `json:"fork"`
	DefaultBranch string     `json:"default_branch"`
	CreatedAt     time.Time  `json:"created_at"`
	PushedAt      time.Time  `json:"pushed_at"`
	SizeKB        int        `json:"size_kb"`
	Contributors  int        `json:"contributors"` // 0 means "unknown"
	Releases      int        `json:"releases"`     // 0 means "none/unknown"
	LatestRelease string     `json:"latest_release,omitempty"`
	LatestTagAt   time.Time  `json:"latest_release_at,omitempty"`
}

// LangSize is a single language and its byte count in the repo.
type LangSize struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

// HealthResult scores how actively and safely a project is maintained.
type HealthResult struct {
	Score      int      `json:"score"` // 0-100
	Factors    []Factor `json:"factors"`
	Maintained bool     `json:"maintained"`
	Deprecated string   `json:"deprecated,omitempty"` // registry deprecation reason, if any
}

// TelemetryResult captures how "phone-home-y" the project looks.
type TelemetryResult struct {
	Score         int      `json:"score"` // 0-100, higher = quieter / more private
	Level         Level    `json:"-"`
	Status        string   `json:"status"`
	AnalyticsLibs []string `json:"analytics_libs,omitempty"`
	NetworkLibs   []string `json:"network_libs,omitempty"`
	Factors       []Factor `json:"factors"`
}

// PermissionsResult captures the scariest capabilities a package asks for.
type PermissionsResult struct {
	Score          int      `json:"score"` // 0-100, higher = fewer scary powers
	InstallScripts []string `json:"install_scripts,omitempty"`
	Capabilities   []Factor `json:"capabilities"`
}

// SizeResult is the install footprint.
type SizeResult struct {
	Known        bool     `json:"known"`
	Source       string   `json:"source"` // "npm registry", "pypi", "repo"
	Version      string   `json:"version,omitempty"`
	InstallBytes int64    `json:"install_bytes"`
	DirectDeps   int      `json:"direct_deps"`
	Score        int      `json:"score"` // 0-100, higher = leaner
	Factors      []Factor `json:"factors"`
}

// Axis is one scored dimension shown on the verdict card.
type Axis struct {
	Name   string `json:"name"`
	Score  int    `json:"score"` // 0-100
	Grade  Grade  `json:"grade"`
	Detail string `json:"detail"`
}

// ScoreResult is the combined verdict.
type ScoreResult struct {
	Overall  int     `json:"overall"` // 0-100
	Grade    Grade   `json:"grade"`
	Verdict  Verdict `json:"verdict"`
	Headline string  `json:"headline"`
	Axes     []Axis  `json:"axes"`
}

// Narrative is the plain-English "what is it & should you trust it" blurb.
type Narrative struct {
	Summary string   `json:"summary"`
	Bullets []string `json:"bullets,omitempty"`
	Source  string   `json:"source"` // "anthropic" | "openai" | "ollama" | "offline"
}

// Report is the full result of a scan.
type Report struct {
	Target      Target            `json:"target"`
	Repo        RepoInfo          `json:"repo"`
	Health      HealthResult      `json:"health"`
	Telemetry   TelemetryResult   `json:"telemetry"`
	Permissions PermissionsResult `json:"permissions"`
	Size        SizeResult        `json:"size"`
	Score       ScoreResult       `json:"score"`
	Narrative   Narrative         `json:"narrative"`
	Warnings    []string          `json:"warnings,omitempty"`
	GeneratedAt time.Time         `json:"generated_at"`
}
