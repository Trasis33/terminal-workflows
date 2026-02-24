package importer

import (
	"strings"
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Parameter Conversion Tests
// =============================================================================

func TestConvertPetParam_SimpleParam(t *testing.T) {
	got := convertPetParam("<host>")
	assert.Equal(t, "{{host}}", got)
}

func TestConvertPetParam_ParamWithDefault(t *testing.T) {
	got := convertPetParam("<host=localhost>")
	assert.Equal(t, "{{host:localhost}}", got)
}

func TestConvertPetParam_DefaultContainsEquals(t *testing.T) {
	got := convertPetParam("<url=http://example.com>")
	assert.Equal(t, "{{url:http://example.com}}", got)
}

func TestConvertPetParam_NoParams(t *testing.T) {
	got := convertPetParam("no params here")
	assert.Equal(t, "no params here", got)
}

func TestConvertPetParam_MultipleParams(t *testing.T) {
	got := convertPetParam("<a> and <b=x>")
	assert.Equal(t, "{{a}} and {{b:x}}", got)
}

func TestConvertPetParam_InContext(t *testing.T) {
	got := convertPetParam("kubectl apply -f <manifest=deploy.yaml> --namespace <env=staging>")
	assert.Equal(t, "kubectl apply -f {{manifest:deploy.yaml}} --namespace {{env:staging}}", got)
}

// =============================================================================
// Slugify Tests
// =============================================================================

func TestSlugifyName_SimplePhrase(t *testing.T) {
	got := slugifyName("Deploy to staging environment")
	assert.Equal(t, "deploy-to-staging-environment", got)
}

func TestSlugifyName_ShortPhrase(t *testing.T) {
	got := slugifyName("Show SSL cert")
	assert.Equal(t, "show-ssl-cert", got)
}

func TestSlugifyName_SpecialChars(t *testing.T) {
	got := slugifyName("My (special) workflow!")
	assert.Equal(t, "my-special-workflow", got)
}

func TestSlugifyName_Empty(t *testing.T) {
	got := slugifyName("")
	assert.Equal(t, "unnamed", got)
}

func TestSlugifyName_OnlySpecialChars(t *testing.T) {
	got := slugifyName("!@#$%")
	assert.Equal(t, "unnamed", got)
}

// =============================================================================
// Pet TOML Import Tests
// =============================================================================

func TestPetImport_SingleSnippet(t *testing.T) {
	toml := `
[[snippets]]
  description = "Deploy to staging"
  command = "kubectl apply -f <manifest=deploy.yaml> --namespace <env=staging>"
  tag = ["k8s", "deploy"]
  output = ""
`
	imp := &PetImporter{}
	result, err := imp.Import(strings.NewReader(toml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)

	wf := result.Workflows[0]
	assert.Equal(t, "deploy-to-staging", wf.Name)
	assert.Equal(t, "kubectl apply -f {{manifest:deploy.yaml}} --namespace {{env:staging}}", wf.Command)
	assert.Equal(t, "Deploy to staging", wf.Description)
	assert.Equal(t, []string{"k8s", "deploy"}, wf.Tags)

	// Args should be extracted from the converted command
	require.Len(t, wf.Args, 2)
	assert.Equal(t, "manifest", wf.Args[0].Name)
	assert.Equal(t, "deploy.yaml", wf.Args[0].Default)
	assert.Equal(t, "env", wf.Args[1].Name)
	assert.Equal(t, "staging", wf.Args[1].Default)
}

func TestPetImport_OutputFieldWarning(t *testing.T) {
	toml := `
[[snippets]]
  description = "Show SSL cert"
  command = "echo | openssl s_client -connect <host>:443"
  tag = ["ssl"]
  output = "notBefore=Nov 3 2015"
`
	imp := &PetImporter{}
	result, err := imp.Import(strings.NewReader(toml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "output")
	assert.Contains(t, result.Warnings[0], "notBefore=Nov 3 2015")
}

func TestPetImport_MultipleSnippets(t *testing.T) {
	toml := `
[[snippets]]
  description = "First command"
  command = "echo hello"
  tag = []
  output = ""

[[snippets]]
  description = "Second command"
  command = "echo world"
  tag = ["test"]
  output = ""
`
	imp := &PetImporter{}
	result, err := imp.Import(strings.NewReader(toml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 2)
	assert.Equal(t, "first-command", result.Workflows[0].Name)
	assert.Equal(t, "second-command", result.Workflows[1].Name)
}

func TestPetImport_EmptyCommand(t *testing.T) {
	toml := `
[[snippets]]
  description = "Empty cmd"
  command = ""
  tag = []
  output = ""
`
	imp := &PetImporter{}
	result, err := imp.Import(strings.NewReader(toml))
	require.NoError(t, err)
	assert.Empty(t, result.Workflows)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "empty command")
}

func TestPetImport_DescriptionSlugified(t *testing.T) {
	toml := `
[[snippets]]
  description = "Deploy to staging environment"
  command = "deploy.sh"
  tag = []
  output = ""
`
	imp := &PetImporter{}
	result, err := imp.Import(strings.NewReader(toml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	assert.Equal(t, "deploy-to-staging-environment", result.Workflows[0].Name)
}

func TestPetImport_TagsCopied(t *testing.T) {
	toml := `
[[snippets]]
  description = "Tagged workflow"
  command = "echo tagged"
  tag = ["k8s", "deploy"]
  output = ""
`
	imp := &PetImporter{}
	result, err := imp.Import(strings.NewReader(toml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	assert.Equal(t, []string{"k8s", "deploy"}, result.Workflows[0].Tags)
}

// =============================================================================
// Warp YAML Import Tests
// =============================================================================

func TestWarpImport_SingleWorkflow(t *testing.T) {
	yaml := `
name: Uninstall Homebrew package
command: |-
    brew tap beeftornado/rmtree
    brew rmtree {{package_name}}
tags:
  - homebrew
description: Remove a Homebrew package and dependencies
arguments:
  - name: package_name
    description: The package to remove
    default_value: ~
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)

	wf := result.Workflows[0]
	assert.Equal(t, "Uninstall Homebrew package", wf.Name)
	assert.Contains(t, wf.Command, "brew rmtree {{package_name}}")
	assert.Equal(t, "Remove a Homebrew package and dependencies", wf.Description)
	assert.Equal(t, []string{"homebrew"}, wf.Tags)

	// Args mapped from arguments
	require.Len(t, wf.Args, 1)
	assert.Equal(t, "package_name", wf.Args[0].Name)
	assert.Equal(t, "The package to remove", wf.Args[0].Description)
	assert.Equal(t, "", wf.Args[0].Default) // YAML null -> empty string
}

func TestWarpImport_SourceURLWarning(t *testing.T) {
	yaml := `
name: Test workflow
command: echo hello
tags: []
description: A test
source_url: "https://example.com"
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	assert.True(t, warningsContain(result.Warnings, "source_url"))
}

func TestWarpImport_AuthorWarnings(t *testing.T) {
	yaml := `
name: Test workflow
command: echo hello
tags: []
description: A test
author: John Doe
author_url: "https://johndoe.com"
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	assert.True(t, warningsContain(result.Warnings, "author"))
	assert.True(t, warningsContain(result.Warnings, "author_url"))
}

func TestWarpImport_ShellsWarning(t *testing.T) {
	yaml := `
name: Test workflow
command: echo hello
tags: []
description: A test
shells:
  - bash
  - zsh
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	assert.True(t, warningsContain(result.Warnings, "shells"))
}

func TestWarpImport_NullDefaultValue(t *testing.T) {
	yaml := `
name: Test
command: echo {{arg}}
tags: []
description: Test
arguments:
  - name: arg
    description: An argument
    default_value: ~
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	require.Len(t, result.Workflows[0].Args, 1)
	assert.Equal(t, "", result.Workflows[0].Args[0].Default)
}

func TestWarpImport_TemplateSyntaxUnchanged(t *testing.T) {
	yaml := `
name: Test
command: echo {{arg}}
tags: []
description: Test
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	assert.Equal(t, "echo {{arg}}", result.Workflows[0].Command)
}

func TestWarpImport_MultipleWorkflows(t *testing.T) {
	yaml := `
name: First
command: echo first
tags: []
description: First workflow
---
name: Second
command: echo second
tags: []
description: Second workflow
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 2)
	assert.Equal(t, "First", result.Workflows[0].Name)
	assert.Equal(t, "Second", result.Workflows[1].Name)
}

func TestWarpImport_ArgumentWithDefault(t *testing.T) {
	yaml := `
name: Test
command: echo {{greeting}}
tags: []
description: Test
arguments:
  - name: greeting
    description: What to say
    default_value: hello
`
	imp := &WarpImporter{}
	result, err := imp.Import(strings.NewReader(yaml))
	require.NoError(t, err)
	require.Len(t, result.Workflows, 1)
	require.Len(t, result.Workflows[0].Args, 1)
	assert.Equal(t, "hello", result.Workflows[0].Args[0].Default)
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestPetImporterImplementsImporter(t *testing.T) {
	var _ Importer = &PetImporter{}
}

func TestWarpImporterImplementsImporter(t *testing.T) {
	var _ Importer = &WarpImporter{}
}

// =============================================================================
// Helpers
// =============================================================================

func warningsContain(warnings []string, substr string) bool {
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}

// Ensure store.Workflow is used (avoid unused import)
var _ = store.Workflow{}
