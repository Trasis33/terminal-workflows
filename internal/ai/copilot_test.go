package ai

import (
	"context"
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGenerator implements Generator with canned responses for testing.
type mockGenerator struct {
	generateResult *GenerateResult
	generateErr    error
	autofillResult *AutofillResult
	autofillErr    error
	available      bool
}

func (m *mockGenerator) Generate(_ context.Context, _ GenerateRequest) (*GenerateResult, error) {
	return m.generateResult, m.generateErr
}

func (m *mockGenerator) Autofill(_ context.Context, _ AutofillRequest) (*AutofillResult, error) {
	return m.autofillResult, m.autofillErr
}

func (m *mockGenerator) Available() bool {
	return m.available
}

func (m *mockGenerator) Close() error {
	return nil
}

func TestMockGenerate(t *testing.T) {
	mock := &mockGenerator{
		generateResult: &GenerateResult{
			Name:        "deploy-staging",
			Command:     "kubectl rollout restart deployment/{{app}} -n staging",
			Description: "Deploy app to staging",
			Tags:        []string{"deploy", "kubernetes"},
			Args: []store.Arg{
				{Name: "app", Description: "Application name"},
			},
		},
		available: true,
	}

	var gen Generator = mock // Verify interface satisfaction
	result, err := gen.Generate(context.Background(), GenerateRequest{
		Description: "deploy to staging",
	})

	require.NoError(t, err)
	assert.Equal(t, "deploy-staging", result.Name)
	assert.Equal(t, "kubectl rollout restart deployment/{{app}} -n staging", result.Command)
	assert.Equal(t, "Deploy app to staging", result.Description)
	assert.Equal(t, []string{"deploy", "kubernetes"}, result.Tags)
	assert.Len(t, result.Args, 1)
	assert.Equal(t, "app", result.Args[0].Name)
	assert.True(t, gen.Available())
}

func TestMockAutofill(t *testing.T) {
	name := "docker-run"
	desc := "Run a Docker container with port mapping"
	mock := &mockGenerator{
		autofillResult: &AutofillResult{
			Name:        &name,
			Description: &desc,
			Tags:        []string{"docker", "container"},
			Args: []store.Arg{
				{Name: "port", Description: "Host port to bind", Default: "8080"},
				{Name: "image", Description: "Docker image name"},
			},
		},
		available: true,
	}

	var gen Generator = mock
	result, err := gen.Autofill(context.Background(), AutofillRequest{
		Workflow: &store.Workflow{
			Command: "docker run -p {{port}}:80 {{image}}",
		},
		Fields: []string{"name", "description", "tags", "args"},
	})

	require.NoError(t, err)
	require.NotNil(t, result.Name)
	assert.Equal(t, "docker-run", *result.Name)
	require.NotNil(t, result.Description)
	assert.Equal(t, "Run a Docker container with port mapping", *result.Description)
	assert.Equal(t, []string{"docker", "container"}, result.Tags)
	assert.Len(t, result.Args, 2)
}

func TestMockUnavailable(t *testing.T) {
	mock := &mockGenerator{
		generateErr: ErrUnavailable,
		autofillErr: ErrUnavailable,
		available:   false,
	}

	var gen Generator = mock
	assert.False(t, gen.Available())

	_, err := gen.Generate(context.Background(), GenerateRequest{Description: "test"})
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.Contains(t, err.Error(), "gh extension install github/gh-copilot")

	_, err = gen.Autofill(context.Background(), AutofillRequest{})
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestBuildGeneratePrompt(t *testing.T) {
	tests := []struct {
		name     string
		req      GenerateRequest
		contains []string
		absent   []string
	}{
		{
			name: "description only",
			req:  GenerateRequest{Description: "deploy to staging"},
			contains: []string{
				"Generate a workflow for: deploy to staging",
			},
			absent: []string{
				"Target shell:",
				"Suggested tags:",
				"Folder/category:",
			},
		},
		{
			name: "all fields",
			req: GenerateRequest{
				Description: "build docker image",
				Shell:       "bash",
				Tags:        []string{"docker", "ci"},
				Folder:      "deployment",
			},
			contains: []string{
				"Generate a workflow for: build docker image",
				"Target shell: bash",
				"Suggested tags: docker, ci",
				"Folder/category: deployment",
			},
		},
		{
			name: "shell only",
			req: GenerateRequest{
				Description: "list files",
				Shell:       "zsh",
			},
			contains: []string{
				"Generate a workflow for: list files",
				"Target shell: zsh",
			},
			absent: []string{
				"Suggested tags:",
				"Folder/category:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildGeneratePrompt(tt.req)
			for _, s := range tt.contains {
				assert.Contains(t, prompt, s)
			}
			for _, s := range tt.absent {
				assert.NotContains(t, prompt, s)
			}
		})
	}
}

func TestBuildAutofillPrompt(t *testing.T) {
	tests := []struct {
		name     string
		req      AutofillRequest
		contains []string
		absent   []string
	}{
		{
			name: "command and fields only",
			req: AutofillRequest{
				Workflow: &store.Workflow{
					Command: "docker run -p {{port}}:80 {{image}}",
				},
				Fields: []string{"name", "description", "tags"},
			},
			contains: []string{
				"Workflow command: docker run -p {{port}}:80 {{image}}",
				"Fill these fields: name, description, tags",
			},
			absent: []string{
				"Existing name:",
				"Existing description:",
				"Existing tags:",
			},
		},
		{
			name: "with existing metadata",
			req: AutofillRequest{
				Workflow: &store.Workflow{
					Name:        "docker-run",
					Command:     "docker run -p {{port}}:80 {{image}}",
					Description: "Run container",
					Tags:        []string{"docker"},
				},
				Fields: []string{"args"},
			},
			contains: []string{
				"Workflow command: docker run -p {{port}}:80 {{image}}",
				"Existing name: docker-run",
				"Existing description: Run container",
				"Existing tags: docker",
				"Fill these fields: args",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildAutofillPrompt(tt.req)
			for _, s := range tt.contains {
				assert.Contains(t, prompt, s)
			}
			for _, s := range tt.absent {
				assert.NotContains(t, prompt, s)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean JSON",
			input:    `{"name":"test"}`,
			expected: `{"name":"test"}`,
		},
		{
			name:     "markdown json fence",
			input:    "Here is the result:\n```json\n{\"name\":\"test\"}\n```\nDone.",
			expected: `{"name":"test"}`,
		},
		{
			name:     "markdown plain fence",
			input:    "```\n{\"name\":\"test\"}\n```",
			expected: `{"name":"test"}`,
		},
		{
			name:     "text before JSON",
			input:    "Sure! Here is your workflow:\n{\"name\":\"test\",\"command\":\"ls\"}",
			expected: `{"name":"test","command":"ls"}`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no JSON",
			input:    "I cannot generate that workflow.",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModelConfigForTask(t *testing.T) {
	tests := []struct {
		name     string
		config   ModelConfig
		task     string
		expected string
	}{
		{
			name:     "generate with config",
			config:   ModelConfig{Generate: "gpt-5.2", Autofill: "gpt-4o-mini", Fallback: "gpt-4o-mini"},
			task:     "generate",
			expected: "gpt-5.2",
		},
		{
			name:     "autofill with config",
			config:   ModelConfig{Generate: "gpt-5.2", Autofill: "gpt-4o-mini", Fallback: "gpt-4o-mini"},
			task:     "autofill",
			expected: "gpt-4o-mini",
		},
		{
			name:     "unknown task falls to fallback",
			config:   ModelConfig{Generate: "gpt-5.2", Autofill: "gpt-4o-mini", Fallback: "gpt-4o"},
			task:     "unknown",
			expected: "gpt-4o",
		},
		{
			name:     "empty generate falls to fallback",
			config:   ModelConfig{Fallback: "gpt-4o"},
			task:     "generate",
			expected: "gpt-4o",
		},
		{
			name:     "all empty falls to default",
			config:   ModelConfig{},
			task:     "generate",
			expected: "gpt-4.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.ForTask(tt.task))
		})
	}
}

func TestDefaultModelConfig(t *testing.T) {
	cfg := DefaultModelConfig()
	assert.Equal(t, "gpt-5-mini", cfg.Generate)
	assert.Equal(t, "gpt-4.1", cfg.Autofill)
	assert.Equal(t, "gpt-4.1", cfg.Fallback)
}
