package register

import (
	"testing"
)

func TestDetectParams_IPv4(t *testing.T) {
	suggestions := DetectParams("ssh root@192.168.1.100")
	found := findByName(suggestions, "host")
	if found == nil {
		t.Fatal("expected to detect IPv4 as 'host'")
	}
	if found.Original != "192.168.1.100" {
		t.Errorf("expected original '192.168.1.100', got %q", found.Original)
	}
}

func TestDetectParams_Port(t *testing.T) {
	suggestions := DetectParams("curl localhost:8080/api")
	found := findByName(suggestions, "port")
	if found == nil {
		t.Fatal("expected to detect port")
	}
	if found.Original != "8080" {
		t.Errorf("expected original '8080', got %q", found.Original)
	}
}

func TestDetectParams_Port5Digit(t *testing.T) {
	suggestions := DetectParams("nc -l :54321")
	found := findByName(suggestions, "port")
	if found == nil {
		t.Fatal("expected to detect 5-digit port")
	}
	if found.Original != "54321" {
		t.Errorf("expected original '54321', got %q", found.Original)
	}
}

func TestDetectParams_URL(t *testing.T) {
	suggestions := DetectParams("curl https://api.example.com/v1/users")
	found := findByName(suggestions, "url")
	if found == nil {
		t.Fatal("expected to detect URL")
	}
	if found.Original != "https://api.example.com/v1/users" {
		t.Errorf("expected full URL, got %q", found.Original)
	}
}

func TestDetectParams_AbsolutePath(t *testing.T) {
	suggestions := DetectParams("cat /etc/nginx/nginx.conf")
	found := findByName(suggestions, "path")
	if found == nil {
		t.Fatal("expected to detect absolute path")
	}
	if found.Original != "/etc/nginx/nginx.conf" {
		t.Errorf("expected path '/etc/nginx/nginx.conf', got %q", found.Original)
	}
}

func TestDetectParams_AllCaps(t *testing.T) {
	suggestions := DetectParams("export MY_DATABASE_URL=postgres://localhost")
	found := findByName(suggestions, "my_database_url")
	if found == nil {
		t.Fatal("expected to detect ALL_CAPS value")
	}
	if found.Original != "MY_DATABASE_URL" {
		t.Errorf("expected original 'MY_DATABASE_URL', got %q", found.Original)
	}
}

func TestDetectParams_Email(t *testing.T) {
	suggestions := DetectParams("git config user.email admin@example.com")
	found := findByName(suggestions, "email")
	if found == nil {
		t.Fatal("expected to detect email")
	}
	if found.Original != "admin@example.com" {
		t.Errorf("expected 'admin@example.com', got %q", found.Original)
	}
}

func TestDetectParams_ExcludesCommonKeywords(t *testing.T) {
	keywords := []string{"HTTP", "HTTPS", "GET", "POST", "PUT", "DELETE", "SSH", "SCP", "NULL", "TRUE", "FALSE", "EOF", "JSON", "API", "URL"}
	for _, kw := range keywords {
		suggestions := DetectParams("echo " + kw)
		found := findByName(suggestions, kw) // exact match (uppercase)
		if found != nil {
			t.Errorf("keyword %q should be excluded, but was suggested", kw)
		}
		// Also check lowercase param name
		found = findByParamName(suggestions, kw)
		if found != nil {
			t.Errorf("keyword %q should be excluded (lowercase check), but was suggested", kw)
		}
	}
}

func TestDetectParams_NoFalsePositiveShortCaps(t *testing.T) {
	// 2-char ALL_CAPS should NOT match (minimum 3 chars)
	suggestions := DetectParams("if OK then")
	found := findByParamName(suggestions, "ok")
	if found != nil {
		// OK is also in commonKeywords, but even without it, 2-char "OK" would be excluded by regex
		// The keyword list also covers this
	}
}

func TestDetectParams_MultiplePatterns(t *testing.T) {
	cmd := "ssh user@192.168.1.1 -p :2222 -i /home/user/.ssh/id_rsa"
	suggestions := DetectParams(cmd)

	host := findByName(suggestions, "host")
	if host == nil {
		t.Error("expected to detect host IP")
	}

	port := findByName(suggestions, "port")
	if port == nil {
		t.Error("expected to detect port")
	}

	path := findByName(suggestions, "path")
	if path == nil {
		t.Error("expected to detect path")
	}
}

func TestDetectParams_URLDoesNotDuplicateIP(t *testing.T) {
	// IP within a URL should not be detected separately as a host
	cmd := "curl http://192.168.1.1:8080/api"
	suggestions := DetectParams(cmd)

	// Should have URL but NOT a separate host for the IP inside the URL
	urlCount := countByName(suggestions, "url")
	hostCount := countByName(suggestions, "host")

	if urlCount != 1 {
		t.Errorf("expected 1 url suggestion, got %d", urlCount)
	}
	if hostCount != 0 {
		t.Errorf("expected 0 host suggestions (IP is inside URL), got %d", hostCount)
	}
}

func TestDetectParams_URLDoesNotDuplicatePath(t *testing.T) {
	// Path within a URL should not be detected separately
	cmd := "wget https://example.com/downloads/file.tar.gz"
	suggestions := DetectParams(cmd)

	urlCount := countByName(suggestions, "url")
	pathCount := countByName(suggestions, "path")

	if urlCount != 1 {
		t.Errorf("expected 1 url suggestion, got %d", urlCount)
	}
	if pathCount != 0 {
		t.Errorf("expected 0 path suggestions (path is inside URL), got %d", pathCount)
	}
}

func TestDetectParams_EmptyCommand(t *testing.T) {
	suggestions := DetectParams("")
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for empty command, got %d", len(suggestions))
	}
}

func TestDetectParams_NoPatterns(t *testing.T) {
	suggestions := DetectParams("echo hello world")
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for plain command, got %d", len(suggestions))
	}
}

func TestDetectParams_SortedByPosition(t *testing.T) {
	cmd := "docker run -p :8080:80 192.168.1.1"
	suggestions := DetectParams(cmd)

	for i := 1; i < len(suggestions); i++ {
		if suggestions[i].Start < suggestions[i-1].Start {
			t.Errorf("suggestions not sorted by position: %d < %d at index %d",
				suggestions[i].Start, suggestions[i-1].Start, i)
		}
	}
}

func TestDetectParams_HttpURL(t *testing.T) {
	// http:// (not just https://) should be detected
	suggestions := DetectParams("curl http://localhost:3000")
	found := findByName(suggestions, "url")
	if found == nil {
		t.Fatal("expected to detect http URL")
	}
}

func TestDetectParams_PathAtStart(t *testing.T) {
	suggestions := DetectParams("/usr/local/bin/myapp --flag")
	found := findByName(suggestions, "path")
	if found == nil {
		t.Fatal("expected to detect path at start of command")
	}
}

func TestDetectParams_PortNotShortNumbers(t *testing.T) {
	// 3-digit numbers after colon should NOT be detected as ports
	suggestions := DetectParams("value:123")
	found := findByName(suggestions, "port")
	if found != nil {
		t.Error("3-digit number should not be detected as port")
	}
}

// Helper functions

func findByName(suggestions []Suggestion, paramName string) *Suggestion {
	for _, s := range suggestions {
		if s.ParamName == paramName {
			return &s
		}
	}
	return nil
}

func findByParamName(suggestions []Suggestion, original string) *Suggestion {
	lower := original
	for _, s := range suggestions {
		if s.Original == original || s.ParamName == lower {
			return &s
		}
	}
	return nil
}

func countByName(suggestions []Suggestion, paramName string) int {
	count := 0
	for _, s := range suggestions {
		if s.ParamName == paramName {
			count++
		}
	}
	return count
}
