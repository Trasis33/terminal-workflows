package main

import (
	"strings"
	"testing"

	"github.com/fredriklanga/wf/internal/register"
)

func TestSubstituteParams(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		picks    []string
		expected string
	}{
		{
			name:     "ip default",
			command:  "ssh root@10.0.0.1",
			picks:    []string{"10.0.0.1"},
			expected: "ssh root@{{host:10.0.0.1}}",
		},
		{
			name:     "url with colons",
			command:  "curl http://api.example.com:8080/v1",
			picks:    []string{"http://api.example.com:8080/v1"},
			expected: "curl {{url:http://api.example.com:8080/v1}}",
		},
		{
			name:     "multiple picks",
			command:  "ssh root@10.0.0.1 -p :2222",
			picks:    []string{"10.0.0.1", "2222"},
			expected: "ssh root@{{host:10.0.0.1}} -p :{{port:2222}}",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			suggestions := register.DetectParams(tc.command)
			if len(suggestions) == 0 {
				t.Fatalf("expected suggestions for %q", tc.command)
			}

			var selected []int
			for _, pick := range tc.picks {
				found := -1
				for i, s := range suggestions {
					if s.Original == pick {
						found = i
						break
					}
				}
				if found == -1 {
					t.Fatalf("missing suggestion for %q in %+v", pick, suggestions)
				}
				selected = append(selected, found)
			}

			got := substituteParams(tc.command, suggestions, selected)
			if got != tc.expected {
				t.Fatalf("substituteParams() mismatch\nexpected: %s\ngot:      %s", tc.expected, got)
			}
		})
	}
}

func TestSubstituteParams_UsesSuggestionOffsets(t *testing.T) {
	command := "echo host=10.0.0.1"
	target := "10.0.0.1"
	start := strings.Index(command, target)
	if start == -1 {
		t.Fatal("test setup failed: target not found")
	}

	suggestions := []register.Suggestion{{
		Original:  target,
		ParamName: "host",
		Start:     start,
		End:       start + len(target),
	}}

	got := substituteParams(command, suggestions, []int{0})
	want := "echo host={{host:10.0.0.1}}"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
