package params

import (
	"testing"

	"github.com/fredriklanga/wf/internal/template"
)

func TestParamTypeListStringRoundTrip(t *testing.T) {
	if template.ParamTypeFromString("list") != template.ParamList {
		t.Fatalf("expected list type to round-trip")
	}
	if template.ParamList.String() != "list" {
		t.Fatalf("got %q, want %q", template.ParamList.String(), "list")
	}
}
