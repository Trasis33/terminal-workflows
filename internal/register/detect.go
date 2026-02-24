// Package register provides auto-detection of parameterizable patterns in commands.
package register

import (
	"regexp"
	"sort"
	"strings"
)

// Suggestion represents a detected parameterizable pattern in a command string.
type Suggestion struct {
	Original  string // The matched text
	ParamName string // Suggested parameter name
	Start     int    // Start position in command string
	End       int    // End position in command string
}

// commonKeywords are ALL_CAPS values that should NOT be suggested as parameters.
// These are common shell/HTTP/programming keywords.
var commonKeywords = map[string]bool{
	"HTTP":    true,
	"HTTPS":   true,
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
	"SSH":     true,
	"SCP":     true,
	"NULL":    true,
	"TRUE":    true,
	"FALSE":   true,
	"EOF":     true,
	"STDIN":   true,
	"STDOUT":  true,
	"STDERR":  true,
	"ASCII":   true,
	"UTF":     true,
	"JSON":    true,
	"XML":     true,
	"HTML":    true,
	"CSS":     true,
	"SQL":     true,
	"API":     true,
	"URL":     true,
	"URI":     true,
	"TCP":     true,
	"UDP":     true,
	"DNS":     true,
	"TLS":     true,
	"SSL":     true,
	"FTP":     true,
	"SFTP":    true,
	"AWS":     true,
	"GCP":     true,
	"PID":     true,
	"TTY":     true,
	"NFS":     true,
	"ACL":     true,
	"ENV":     true,
	"PATH":    true,
	"HOME":    true,
	"USER":    true,
	"TERM":    true,
	"SHELL":   true,
	"LANG":    true,
	"SUDO":    true,
	"CRON":    true,
	"YAML":    true,
	"TOML":    true,
	"CSV":     true,
	"OK":      true,
}

var (
	// IPv4 address pattern
	ipv4Re = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)

	// Port after colon (4-5 digits)
	portRe = regexp.MustCompile(`(?::)(\d{4,5})\b`)

	// URL pattern
	urlRe = regexp.MustCompile(`https?://\S+`)

	// Absolute path (3+ chars after /)
	absPathRe = regexp.MustCompile(`(?:^|\s)(/[\w./-]{3,})`)

	// ALL_CAPS values (3+ chars, letters/digits/underscore, starts with letter)
	allCapsRe = regexp.MustCompile(`\b([A-Z][A-Z0-9_]{2,})\b`)

	// Email-like pattern
	emailRe = regexp.MustCompile(`\b([\w.]+@[\w.]+\.\w+)\b`)
)

// DetectParams scans a command string for obvious parameterizable patterns
// and returns suggestions. Detection is conservative to avoid false positives.
func DetectParams(command string) []Suggestion {
	var suggestions []Suggestion

	// Detect URLs first (so we can skip IP/path detection within URLs)
	urlMatches := urlRe.FindAllStringIndex(command, -1)
	urlRanges := make([][2]int, len(urlMatches))
	for i, m := range urlMatches {
		urlRanges[i] = [2]int{m[0], m[1]}
		suggestions = append(suggestions, Suggestion{
			Original:  command[m[0]:m[1]],
			ParamName: "url",
			Start:     m[0],
			End:       m[1],
		})
	}

	// Detect IPv4 addresses (skip if inside a URL)
	for _, m := range ipv4Re.FindAllStringIndex(command, -1) {
		if inRange(m[0], urlRanges) {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Original:  command[m[0]:m[1]],
			ParamName: "host",
			Start:     m[0],
			End:       m[1],
		})
	}

	// Detect ports (skip if inside a URL)
	for _, m := range portRe.FindAllStringSubmatchIndex(command, -1) {
		// m[2]:m[3] is the capture group (digits only)
		if inRange(m[2], urlRanges) {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Original:  command[m[2]:m[3]],
			ParamName: "port",
			Start:     m[2],
			End:       m[3],
		})
	}

	// Detect absolute paths (skip if inside a URL)
	for _, m := range absPathRe.FindAllStringSubmatchIndex(command, -1) {
		// m[2]:m[3] is the capture group (path only, no leading space)
		if inRange(m[2], urlRanges) {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Original:  command[m[2]:m[3]],
			ParamName: "path",
			Start:     m[2],
			End:       m[3],
		})
	}

	// Detect email addresses
	for _, m := range emailRe.FindAllStringIndex(command, -1) {
		suggestions = append(suggestions, Suggestion{
			Original:  command[m[0]:m[1]],
			ParamName: "email",
			Start:     m[0],
			End:       m[1],
		})
	}

	// Detect ALL_CAPS values (exclude common keywords)
	for _, m := range allCapsRe.FindAllStringSubmatchIndex(command, -1) {
		word := command[m[2]:m[3]]
		if commonKeywords[word] {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Original:  word,
			ParamName: strings.ToLower(word),
			Start:     m[2],
			End:       m[3],
		})
	}

	// Sort by position in command string for consistent ordering
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Start < suggestions[j].Start
	})

	return suggestions
}

// inRange checks if a position falls within any of the given ranges.
func inRange(pos int, ranges [][2]int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}
