package template

import (
	"reflect"
	"testing"
)

// --- ExtractParams tests ---

func TestExtractParams_BasicParams(t *testing.T) {
	params := ExtractParams("ssh {{host}} -p {{port}}")
	expected := []Param{
		{Name: "host"},
		{Name: "port"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_WithDefaults(t *testing.T) {
	params := ExtractParams("ssh {{host:localhost}} -p {{port:22}}")
	expected := []Param{
		{Name: "host", Default: "localhost"},
		{Name: "port", Default: "22"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_Deduplicated(t *testing.T) {
	params := ExtractParams("echo {{msg}} && echo {{msg}}")
	expected := []Param{
		{Name: "msg"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_NoParams(t *testing.T) {
	params := ExtractParams("echo hello")
	if len(params) != 0 {
		t.Errorf("expected empty slice, got %+v", params)
	}
}

func TestExtractParams_EmptyString(t *testing.T) {
	params := ExtractParams("")
	if len(params) != 0 {
		t.Errorf("expected empty slice, got %+v", params)
	}
}

func TestExtractParams_MultipleParams(t *testing.T) {
	params := ExtractParams("curl -H 'Auth: {{token}}' {{url}}/api/{{version}}/{{endpoint}}")
	if len(params) != 4 {
		t.Errorf("expected 4 params, got %d: %+v", len(params), params)
	}
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	expectedNames := []string{"token", "url", "version", "endpoint"}
	if !reflect.DeepEqual(names, expectedNames) {
		t.Errorf("got names %v, want %v", names, expectedNames)
	}
}

func TestExtractParams_DefaultWithSpaces(t *testing.T) {
	params := ExtractParams("{{name:default with spaces}}")
	expected := []Param{
		{Name: "name", Default: "default with spaces"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_DuplicateLastDefaultWins(t *testing.T) {
	params := ExtractParams("{{a}} {{b:x}} {{a:y}}")
	expected := []Param{
		{Name: "a", Default: "y"},
		{Name: "b", Default: "x"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_DefaultWithColon(t *testing.T) {
	params := ExtractParams("{{url:http://localhost:8080}}")
	expected := []Param{
		{Name: "url", Default: "http://localhost:8080"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

// --- Render tests ---

func TestRender_BasicSubstitution(t *testing.T) {
	result := Render("ssh {{host}}", map[string]string{"host": "prod"})
	if result != "ssh prod" {
		t.Errorf("got %q, want %q", result, "ssh prod")
	}
}

func TestRender_ValueOverridesDefault(t *testing.T) {
	result := Render("ssh {{host:localhost}}", map[string]string{"host": "prod"})
	if result != "ssh prod" {
		t.Errorf("got %q, want %q", result, "ssh prod")
	}
}

func TestRender_DefaultUsedWhenNoValue(t *testing.T) {
	result := Render("ssh {{host:localhost}}", map[string]string{})
	if result != "ssh localhost" {
		t.Errorf("got %q, want %q", result, "ssh localhost")
	}
}

func TestRender_AllOccurrencesSubstituted(t *testing.T) {
	result := Render("echo {{msg}} && echo {{msg}}", map[string]string{"msg": "hi"})
	if result != "echo hi && echo hi" {
		t.Errorf("got %q, want %q", result, "echo hi && echo hi")
	}
}

func TestRender_MissingParamPlaceholderStays(t *testing.T) {
	result := Render("ssh {{host}}", map[string]string{})
	if result != "ssh {{host}}" {
		t.Errorf("got %q, want %q", result, "ssh {{host}}")
	}
}

func TestRender_NoParamsPassthrough(t *testing.T) {
	result := Render("echo hello", map[string]string{})
	if result != "echo hello" {
		t.Errorf("got %q, want %q", result, "echo hello")
	}
}

func TestRender_NilValues(t *testing.T) {
	result := Render("ssh {{host:localhost}}", nil)
	if result != "ssh localhost" {
		t.Errorf("got %q, want %q", result, "ssh localhost")
	}
}

func TestRender_MixedProvidedAndDefault(t *testing.T) {
	result := Render("ssh {{host}} -p {{port:22}}", map[string]string{"host": "prod"})
	if result != "ssh prod -p 22" {
		t.Errorf("got %q, want %q", result, "ssh prod -p 22")
	}
}
