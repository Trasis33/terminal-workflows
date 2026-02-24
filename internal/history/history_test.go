package history

import (
	"testing"
	"time"
)

// ============================================================
// Zsh Parser Tests
// ============================================================

func TestParseZshExtended(t *testing.T) {
	data := []byte(": 1471766804:3;git push origin master\n: 1471766900:0;ls -la\n: 1471767000:15;docker build -t myapp .\n")

	entries := parseZshHistory(data)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First entry
	if entries[0].Command != "git push origin master" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "git push origin master")
	}
	expectedTS := time.Unix(1471766804, 0)
	if !entries[0].Timestamp.Equal(expectedTS) {
		t.Errorf("entry 0 timestamp = %v, want %v", entries[0].Timestamp, expectedTS)
	}

	// Third entry
	if entries[2].Command != "docker build -t myapp ." {
		t.Errorf("entry 2 command = %q, want %q", entries[2].Command, "docker build -t myapp .")
	}
}

func TestParseZshPlain(t *testing.T) {
	data := []byte("git status\nls -la\ncd /tmp\n")

	entries := parseZshHistory(data)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Command != "git status" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "git status")
	}
	if !entries[0].Timestamp.IsZero() {
		t.Errorf("entry 0 timestamp should be zero for plain format, got %v", entries[0].Timestamp)
	}

	if entries[2].Command != "cd /tmp" {
		t.Errorf("entry 2 command = %q, want %q", entries[2].Command, "cd /tmp")
	}
}

func TestParseZshMixed(t *testing.T) {
	// Mixed format: some extended, some plain
	data := []byte(": 1471766804:3;git push origin master\nls -la\n: 1471767000:15;docker build -t myapp .\n")

	entries := parseZshHistory(data)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First is extended
	if entries[0].Command != "git push origin master" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "git push origin master")
	}
	if entries[0].Timestamp.IsZero() {
		t.Error("entry 0 should have a timestamp (extended format)")
	}

	// Second is plain
	if entries[1].Command != "ls -la" {
		t.Errorf("entry 1 command = %q, want %q", entries[1].Command, "ls -la")
	}
	if !entries[1].Timestamp.IsZero() {
		t.Errorf("entry 1 timestamp should be zero for plain format, got %v", entries[1].Timestamp)
	}

	// Third is extended
	if entries[2].Command != "docker build -t myapp ." {
		t.Errorf("entry 2 command = %q, want %q", entries[2].Command, "docker build -t myapp .")
	}
	if entries[2].Timestamp.IsZero() {
		t.Error("entry 2 should have a timestamp (extended format)")
	}
}

func TestParseZshMultiline(t *testing.T) {
	// Multiline command: continuation line does NOT start with ": "
	data := []byte(": 1471766804:3;for i in 1 2 3; do\necho $i\ndone\n: 1471767000:0;ls\n")

	entries := parseZshHistory(data)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	expectedCmd := "for i in 1 2 3; do\necho $i\ndone"
	if entries[0].Command != expectedCmd {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, expectedCmd)
	}
	if entries[1].Command != "ls" {
		t.Errorf("entry 1 command = %q, want %q", entries[1].Command, "ls")
	}
}

func TestUnmetafy(t *testing.T) {
	// 0x83 followed by a byte → XOR with 0x20
	// Example: 0x83 0xa0 → 0xa0 ^ 0x20 = 0x80
	input := []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x83, 0xa0, 0x77, 0x6f, 0x72, 0x6c, 0x64}
	result := unmetafy(input)

	expected := []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x80, 0x77, 0x6f, 0x72, 0x6c, 0x64}
	if len(result) != len(expected) {
		t.Fatalf("unmetafy length = %d, want %d", len(result), len(expected))
	}
	for i, b := range result {
		if b != expected[i] {
			t.Errorf("unmetafy[%d] = 0x%02x, want 0x%02x", i, b, expected[i])
		}
	}
}

func TestUnmetafyNoMeta(t *testing.T) {
	// No meta bytes → unchanged
	input := []byte("hello world")
	result := unmetafy(input)
	if string(result) != "hello world" {
		t.Errorf("unmetafy = %q, want %q", string(result), "hello world")
	}
}

func TestParseZshEmpty(t *testing.T) {
	entries := parseZshHistory([]byte{})

	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for empty file, got %d", len(entries))
	}
}

func TestZshReaderLastN(t *testing.T) {
	// 10 entries, request last 3
	var lines string
	for i := 0; i < 10; i++ {
		lines += ": 147176680" + string(rune('0'+i)) + ":0;cmd" + string(rune('0'+i)) + "\n"
	}

	reader := &zshReader{data: []byte(lines)}
	entries, err := reader.LastN(3)
	if err != nil {
		t.Fatalf("LastN error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Newest first
	if entries[0].Command != "cmd9" {
		t.Errorf("entries[0].Command = %q, want %q", entries[0].Command, "cmd9")
	}
	if entries[2].Command != "cmd7" {
		t.Errorf("entries[2].Command = %q, want %q", entries[2].Command, "cmd7")
	}
}

// ============================================================
// Bash Parser Tests
// ============================================================

func TestParseBashPlain(t *testing.T) {
	data := []byte("ls -la\ngit status\ndocker ps\n")

	entries := parseBashHistory(data)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Command != "ls -la" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "ls -la")
	}
	if !entries[0].Timestamp.IsZero() {
		t.Errorf("entry 0 timestamp should be zero for plain format, got %v", entries[0].Timestamp)
	}
}

func TestParseBashTimestamped(t *testing.T) {
	data := []byte("#1678886400\nls -la\n#1678886460\ngit status\n")

	entries := parseBashHistory(data)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Command != "ls -la" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "ls -la")
	}
	expectedTS := time.Unix(1678886400, 0)
	if !entries[0].Timestamp.Equal(expectedTS) {
		t.Errorf("entry 0 timestamp = %v, want %v", entries[0].Timestamp, expectedTS)
	}

	if entries[1].Command != "git status" {
		t.Errorf("entry 1 command = %q, want %q", entries[1].Command, "git status")
	}
	expectedTS2 := time.Unix(1678886460, 0)
	if !entries[1].Timestamp.Equal(expectedTS2) {
		t.Errorf("entry 1 timestamp = %v, want %v", entries[1].Timestamp, expectedTS2)
	}
}

func TestParseBashEmptyLinesSkipped(t *testing.T) {
	data := []byte("ls\n\n\ngit status\n\n")

	entries := parseBashHistory(data)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (empty lines skipped), got %d", len(entries))
	}
	if entries[0].Command != "ls" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "ls")
	}
	if entries[1].Command != "git status" {
		t.Errorf("entry 1 command = %q, want %q", entries[1].Command, "git status")
	}
}

func TestParseBashNonNumericHashTreatedAsCommand(t *testing.T) {
	// Lines starting with # that are NOT timestamps are regular commands
	data := []byte("ls\n#this is a comment command\ngit status\n")

	entries := parseBashHistory(data)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[1].Command != "#this is a comment command" {
		t.Errorf("entry 1 command = %q, want %q", entries[1].Command, "#this is a comment command")
	}
}

func TestBashReaderLastN(t *testing.T) {
	var lines string
	for i := 0; i < 20; i++ {
		lines += "cmd" + string(rune('a'+i)) + "\n"
	}

	reader := &bashReader{data: []byte(lines)}
	entries, err := reader.LastN(5)
	if err != nil {
		t.Fatalf("LastN error: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}
	// Newest first
	if entries[0].Command != "cmdt" {
		t.Errorf("entries[0].Command = %q, want %q", entries[0].Command, "cmdt")
	}
}

// ============================================================
// Fish Parser Tests
// ============================================================

func TestParseFishHistory(t *testing.T) {
	data := []byte("- cmd: git push origin master\n  when: 1678886400\n- cmd: ls -la\n  when: 1678886460\n")

	entries := parseFishHistory(data)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Command != "git push origin master" {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "git push origin master")
	}
	expectedTS := time.Unix(1678886400, 0)
	if !entries[0].Timestamp.Equal(expectedTS) {
		t.Errorf("entry 0 timestamp = %v, want %v", entries[0].Timestamp, expectedTS)
	}
}

func TestParseFishHistoryNoWhen(t *testing.T) {
	data := []byte("- cmd: git push\n- cmd: ls\n  when: 1678886460\n")

	entries := parseFishHistory(data)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if !entries[0].Timestamp.IsZero() {
		t.Errorf("entry 0 timestamp should be zero when 'when:' is absent, got %v", entries[0].Timestamp)
	}
	if entries[1].Timestamp.IsZero() {
		t.Error("entry 1 should have a timestamp")
	}
}

func TestParseFishHistoryPathsIgnored(t *testing.T) {
	data := []byte("- cmd: docker build -t myapp .\n  when: 1678886520\n  paths:\n    - Dockerfile\n- cmd: ls\n  when: 1678886600\n")

	entries := parseFishHistory(data)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (paths ignored), got %d", len(entries))
	}
	if entries[0].Command != "docker build -t myapp ." {
		t.Errorf("entry 0 command = %q, want %q", entries[0].Command, "docker build -t myapp .")
	}
}

func TestFishReaderLastN(t *testing.T) {
	data := []byte("- cmd: cmd1\n  when: 100\n- cmd: cmd2\n  when: 200\n- cmd: cmd3\n  when: 300\n- cmd: cmd4\n  when: 400\n- cmd: cmd5\n  when: 500\n")

	reader := &fishReader{data: data}
	entries, err := reader.LastN(2)
	if err != nil {
		t.Fatalf("LastN error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Newest first
	if entries[0].Command != "cmd5" {
		t.Errorf("entries[0].Command = %q, want %q", entries[0].Command, "cmd5")
	}
	if entries[1].Command != "cmd4" {
		t.Errorf("entries[1].Command = %q, want %q", entries[1].Command, "cmd4")
	}
}

// ============================================================
// Shell Detection Tests
// ============================================================

func TestDetectShellZsh(t *testing.T) {
	result := detectShellFromPath("/bin/zsh")
	if result != "zsh" {
		t.Errorf("detectShellFromPath(%q) = %q, want %q", "/bin/zsh", result, "zsh")
	}
}

func TestDetectShellFish(t *testing.T) {
	result := detectShellFromPath("/usr/local/bin/fish")
	if result != "fish" {
		t.Errorf("detectShellFromPath(%q) = %q, want %q", "/usr/local/bin/fish", result, "fish")
	}
}

func TestDetectShellBash(t *testing.T) {
	result := detectShellFromPath("/bin/bash")
	if result != "bash" {
		t.Errorf("detectShellFromPath(%q) = %q, want %q", "/bin/bash", result, "bash")
	}
}

func TestDetectShellEmpty(t *testing.T) {
	result := detectShellFromPath("")
	if result != "bash" {
		t.Errorf("detectShellFromPath(%q) = %q, want %q (fallback)", "", result, "bash")
	}
}

func TestDetectShellUnknown(t *testing.T) {
	result := detectShellFromPath("/usr/bin/tcsh")
	if result != "bash" {
		t.Errorf("detectShellFromPath(%q) = %q, want %q (fallback)", "/usr/bin/tcsh", result, "bash")
	}
}

// ============================================================
// Last() convenience method
// ============================================================

func TestZshReaderLast(t *testing.T) {
	data := []byte(": 100:0;first\n: 200:0;second\n: 300:0;third\n")
	reader := &zshReader{data: data}
	entry, err := reader.Last()
	if err != nil {
		t.Fatalf("Last error: %v", err)
	}
	if entry.Command != "third" {
		t.Errorf("Last().Command = %q, want %q", entry.Command, "third")
	}
}

func TestBashReaderLast(t *testing.T) {
	data := []byte("first\nsecond\nthird\n")
	reader := &bashReader{data: data}
	entry, err := reader.Last()
	if err != nil {
		t.Fatalf("Last error: %v", err)
	}
	if entry.Command != "third" {
		t.Errorf("Last().Command = %q, want %q", entry.Command, "third")
	}
}

func TestFishReaderLast(t *testing.T) {
	data := []byte("- cmd: first\n  when: 100\n- cmd: second\n  when: 200\n- cmd: third\n  when: 300\n")
	reader := &fishReader{data: data}
	entry, err := reader.Last()
	if err != nil {
		t.Fatalf("Last error: %v", err)
	}
	if entry.Command != "third" {
		t.Errorf("Last().Command = %q, want %q", entry.Command, "third")
	}
}

// ============================================================
// Edge Cases
// ============================================================

func TestLastNMoreThanAvailable(t *testing.T) {
	data := []byte("cmd1\ncmd2\n")
	reader := &bashReader{data: data}
	entries, err := reader.LastN(10)
	if err != nil {
		t.Fatalf("LastN error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (all available), got %d", len(entries))
	}
}

func TestLastOnEmptyHistory(t *testing.T) {
	reader := &bashReader{data: []byte{}}
	_, err := reader.Last()
	if err == nil {
		t.Error("Last() on empty history should return error")
	}
}

func TestLastNZero(t *testing.T) {
	data := []byte("cmd1\ncmd2\n")
	reader := &bashReader{data: data}
	entries, err := reader.LastN(0)
	if err != nil {
		t.Fatalf("LastN(0) error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for LastN(0), got %d", len(entries))
	}
}
