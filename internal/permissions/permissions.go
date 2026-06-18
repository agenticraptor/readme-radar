// Package permissions surfaces the scariest capabilities a package asks for:
// code that runs on install, native compilation, process spawning, dynamic
// code execution, and binary downloads. These are the supply-chain signals you
// want to see *before* you `npm i` a stranger's package.
package permissions

import (
	"sort"
	"strings"

	"github.com/agenticraptor/readme-radar/internal/report"
)

// Input is the evidence permission analysis reasons over.
type Input struct {
	Ecosystem      report.Ecosystem
	InstallScripts map[string]string // npm lifecycle hooks (preinstall/install/postinstall/...)
	Deps           []string
	Files          map[string][]byte // manifest/source snippets for a light scan
	NativeBuild    bool              // gypfile / node-gyp present
}

// process-spawning helper libraries.
var spawnLibs = []string{"execa", "cross-spawn", "shelljs", "child-process-promise", "sudo-prompt", "node-pty"}

// dangerous source patterns (light text scan).
var dangerPatterns = map[string]string{
	"child_process": "spawns child processes",
	"subprocess.":   "spawns subprocesses",
	"os.system":     "runs shell commands",
	"eval(":         "uses eval() / dynamic code",
	"new Function(": "builds functions from strings",
	"vm.runin":      "runs code in a VM context",
}

// signs an install script is fetching/executing remote code.
var remoteFetch = []string{"curl ", "wget ", "http://", "https://", "iwr ", "invoke-webrequest"}

// Analyze produces a capability/permission verdict.
func Analyze(in Input) report.PermissionsResult {
	score := 100
	var caps []report.Factor
	var scripts []string

	// 1) Install-time scripts — the headline supply-chain risk.
	if len(in.InstallScripts) > 0 {
		names := keys(in.InstallScripts)
		remote := false
		for _, k := range names {
			body := in.InstallScripts[k]
			scripts = append(scripts, k+": "+truncate(strings.TrimSpace(body), 80))
			if containsAny(strings.ToLower(body), remoteFetch) {
				remote = true
			}
		}
		if remote {
			score -= 60
			caps = append(caps, fac("Install scripts fetch remote code", report.LevelBad,
				"runs "+strings.Join(names, ", ")+" that reach out to the network on install"))
		} else {
			score -= 28
			caps = append(caps, fac("Runs code on install", report.LevelBad,
				"executes "+strings.Join(names, ", ")+" automatically during `install`"))
		}
	} else if in.Ecosystem == report.EcosystemNPM {
		caps = append(caps, fac("Install scripts", report.LevelGood, "no preinstall/postinstall hooks"))
	}

	// 2) Native compilation.
	if in.NativeBuild {
		score -= 10
		caps = append(caps, fac("Native build step", report.LevelWarn,
			"compiles native code (node-gyp) — needs a toolchain and runs at install"))
	}

	// 3) Process spawning via known helpers.
	if hits := matchAny(in.Deps, spawnLibs); len(hits) > 0 {
		score -= 12
		caps = append(caps, fac("Spawns processes", report.LevelWarn,
			"depends on "+strings.Join(hits, ", ")))
	}

	// 4) Dangerous source patterns (best-effort manifest/source scan).
	for needle, desc := range dangerPatterns {
		if scan(in.Files, needle) {
			score -= 8
			caps = append(caps, fac("Dynamic execution", report.LevelWarn, desc))
			break // one such flag is enough; avoid noise
		}
	}

	if len(caps) == 0 {
		caps = append(caps, fac("Capabilities", report.LevelGood, "no install scripts, native builds, or process spawning detected"))
	}

	score = clamp(score)
	return report.PermissionsResult{
		Score:          score,
		InstallScripts: scripts,
		Capabilities:   caps,
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func matchAny(deps, patterns []string) []string {
	var hits []string
	seen := map[string]bool{}
	for _, d := range deps {
		name := strings.ToLower(strings.TrimSpace(d))
		for _, p := range patterns {
			if name == p && !seen[p] {
				seen[p] = true
				hits = append(hits, p)
			}
		}
	}
	sort.Strings(hits)
	return hits
}

func scan(files map[string][]byte, needle string) bool {
	n := strings.ToLower(needle)
	for _, b := range files {
		if strings.Contains(strings.ToLower(string(b)), n) {
			return true
		}
	}
	return false
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func fac(name string, lvl report.Level, detail string) report.Factor {
	return report.Factor{Name: name, Level: lvl, Status: lvl.String(), Detail: detail}
}

func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}
