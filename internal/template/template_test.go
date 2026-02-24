package template

import (
	"reflect"
	"testing"
)

// --- ExtractParams tests: backward compatibility ---

func TestExtractParams_BasicParams(t *testing.T) {
	params := ExtractParams("ssh {{host}} -p {{port}}")
	expected := []Param{
		{Name: "host", Type: ParamText},
		{Name: "port", Type: ParamText},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_WithDefaults(t *testing.T) {
	params := ExtractParams("ssh {{host:localhost}} -p {{port:22}}")
	expected := []Param{
		{Name: "host", Type: ParamText, Default: "localhost"},
		{Name: "port", Type: ParamText, Default: "22"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_Deduplicated(t *testing.T) {
	params := ExtractParams("echo {{msg}} && echo {{msg}}")
	expected := []Param{
		{Name: "msg", Type: ParamText},
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
		{Name: "name", Type: ParamText, Default: "default with spaces"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_DuplicateLastDefaultWins(t *testing.T) {
	params := ExtractParams("{{a}} {{b:x}} {{a:y}}")
	expected := []Param{
		{Name: "a", Type: ParamText, Default: "y"},
		{Name: "b", Type: ParamText, Default: "x"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_DefaultWithColon(t *testing.T) {
	params := ExtractParams("{{url:http://localhost:8080}}")
	expected := []Param{
		{Name: "url", Type: ParamText, Default: "http://localhost:8080"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

// --- ExtractParams tests: enum parameters ---

func TestExtractParams_EnumBasic(t *testing.T) {
	params := ExtractParams("deploy to {{env|dev|staging|prod}}")
	expected := []Param{
		{Name: "env", Type: ParamEnum, Options: []string{"dev", "staging", "prod"}},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_EnumWithDefault(t *testing.T) {
	params := ExtractParams("deploy to {{env|dev|staging|*prod}}")
	expected := []Param{
		{Name: "env", Type: ParamEnum, Options: []string{"dev", "staging", "prod"}, Default: "prod"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_EnumDefaultMiddle(t *testing.T) {
	params := ExtractParams("{{size|s|m|*l|xl}}")
	expected := []Param{
		{Name: "size", Type: ParamEnum, Options: []string{"s", "m", "l", "xl"}, Default: "l"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_EnumWithColonsInOptions(t *testing.T) {
	params := ExtractParams("{{env|dev:3000|staging:8080|*prod:443}}")
	expected := []Param{
		{Name: "env", Type: ParamEnum, Options: []string{"dev:3000", "staging:8080", "prod:443"}, Default: "prod:443"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_EnumDeduplicated(t *testing.T) {
	params := ExtractParams("{{env|a|b}} and {{env|a|b}}")
	expected := []Param{
		{Name: "env", Type: ParamEnum, Options: []string{"a", "b"}},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

// --- ExtractParams tests: dynamic parameters ---

func TestExtractParams_DynamicBasic(t *testing.T) {
	params := ExtractParams("checkout {{branch!git branch --list}}")
	expected := []Param{
		{Name: "branch", Type: ParamDynamic, DynamicCmd: "git branch --list"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_DynamicKubectl(t *testing.T) {
	params := ExtractParams("use ns {{ns!kubectl get ns -o name}}")
	expected := []Param{
		{Name: "ns", Type: ParamDynamic, DynamicCmd: "kubectl get ns -o name"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

func TestExtractParams_DynamicWithPipeInCommand(t *testing.T) {
	// Pipe in the shell command must NOT be interpreted as enum delimiter.
	// The bang takes priority over the pipe.
	params := ExtractParams("checkout {{branch!git branch | grep feature}}")
	expected := []Param{
		{Name: "branch", Type: ParamDynamic, DynamicCmd: "git branch | grep feature"},
	}
	if !reflect.DeepEqual(params, expected) {
		t.Errorf("got %+v, want %+v", params, expected)
	}
}

// --- Render tests: backward compatibility ---

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

// --- Render tests: enum and dynamic params ---

func TestRender_EnumWithValue(t *testing.T) {
	result := Render("deploy {{env|dev|staging|*prod}}", map[string]string{"env": "staging"})
	if result != "deploy staging" {
		t.Errorf("got %q, want %q", result, "deploy staging")
	}
}

func TestRender_EnumDefaultUsed(t *testing.T) {
	result := Render("deploy {{env|dev|*prod}}", map[string]string{})
	if result != "deploy prod" {
		t.Errorf("got %q, want %q", result, "deploy prod")
	}
}

func TestRender_DynamicWithValue(t *testing.T) {
	result := Render("checkout {{branch!git branch}}", map[string]string{"branch": "main"})
	if result != "checkout main" {
		t.Errorf("got %q, want %q", result, "checkout main")
	}
}

func TestRender_DynamicNoDefaultPlaceholderStays(t *testing.T) {
	result := Render("checkout {{branch!git branch}}", map[string]string{})
	if result != "checkout {{branch!git branch}}" {
		t.Errorf("got %q, want %q", result, "checkout {{branch!git branch}}")
	}
}

func TestRender_EnumNoDefaultPlaceholderStays(t *testing.T) {
	result := Render("deploy {{env|dev|staging|prod}}", map[string]string{})
	if result != "deploy {{env|dev|staging|prod}}" {
		t.Errorf("got %q, want %q", result, "deploy {{env|dev|staging|prod}}")
	}
}
