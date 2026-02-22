package picker

import (
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTag   string
		wantQuery string
	}{
		{
			name:      "tag and query",
			input:     "@docker deploy",
			wantTag:   "docker",
			wantQuery: "deploy",
		},
		{
			name:      "query only",
			input:     "deploy",
			wantTag:   "",
			wantQuery: "deploy",
		},
		{
			name:      "tag only",
			input:     "@docker",
			wantTag:   "docker",
			wantQuery: "",
		},
		{
			name:      "empty string",
			input:     "",
			wantTag:   "",
			wantQuery: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, query := ParseQuery(tt.input)
			assert.Equal(t, tt.wantTag, tag)
			assert.Equal(t, tt.wantQuery, query)
		})
	}
}

func sampleWorkflows() []store.Workflow {
	return []store.Workflow{
		{
			Name:        "deploy-app",
			Command:     "kubectl apply -f deployment.yaml",
			Description: "Deploy application to kubernetes",
			Tags:        []string{"k8s", "deploy"},
		},
		{
			Name:        "docker-build",
			Command:     "docker build -t myimage .",
			Description: "Build docker image",
			Tags:        []string{"docker", "build"},
		},
		{
			Name:        "git-push",
			Command:     "git push origin main",
			Description: "Push to main branch",
			Tags:        []string{"git"},
		},
	}
}

func TestSearch_FuzzyMatch(t *testing.T) {
	workflows := sampleWorkflows()

	results := Search("deploy", "", workflows)

	require.NotEmpty(t, results, "should match at least one workflow")
	// The top result should be the deploy-app workflow
	assert.Equal(t, 0, results[0].Index, "deploy-app should be the best match")
}

func TestSearch_TagFilter(t *testing.T) {
	workflows := sampleWorkflows()

	results := Search("", "docker", workflows)

	require.Len(t, results, 1, "should return only docker-tagged workflows")
	assert.Equal(t, 1, results[0].Index, "docker-build should be the only result")
}

func TestSearch_TagFilterPlusFuzzy(t *testing.T) {
	workflows := []store.Workflow{
		{
			Name:        "deploy-staging",
			Command:     "deploy staging",
			Description: "Deploy to staging",
			Tags:        []string{"docker", "deploy"},
		},
		{
			Name:        "deploy-prod",
			Command:     "deploy prod",
			Description: "Deploy to production",
			Tags:        []string{"k8s", "deploy"},
		},
		{
			Name:        "docker-logs",
			Command:     "docker logs -f container",
			Description: "View docker logs",
			Tags:        []string{"docker", "logs"},
		},
	}

	results := Search("dep", "docker", workflows)

	require.NotEmpty(t, results, "should match docker-tagged workflows with 'dep'")
	// Should only match docker-tagged workflows that fuzzy-match "dep"
	for _, r := range results {
		wf := workflows[r.Index]
		hasDockerTag := false
		for _, tag := range wf.Tags {
			if tag == "docker" {
				hasDockerTag = true
				break
			}
		}
		assert.True(t, hasDockerTag, "result %q should have docker tag", wf.Name)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	workflows := sampleWorkflows()

	results := Search("", "", workflows)

	assert.Len(t, results, 3, "empty query should return all workflows")
	// Should preserve original order
	for i, r := range results {
		assert.Equal(t, i, r.Index, "should preserve original order")
	}
}

func TestSearch_NoMatch(t *testing.T) {
	workflows := sampleWorkflows()

	results := Search("zzzzz", "", workflows)

	assert.Empty(t, results, "nonsense query should return no results")
}

func TestSearch_CommandContent(t *testing.T) {
	workflows := []store.Workflow{
		{
			Name:        "build-image",
			Command:     "docker build -t myimage .",
			Description: "Build a container image",
			Tags:        []string{"ci"},
		},
		{
			Name:        "run-tests",
			Command:     "go test ./...",
			Description: "Run all tests",
			Tags:        []string{"ci"},
		},
	}

	results := Search("docker build", "", workflows)

	require.NotEmpty(t, results, "should match workflow by command content")
	assert.Equal(t, 0, results[0].Index, "build-image should match via command content")
}

func TestWorkflowSource(t *testing.T) {
	workflows := sampleWorkflows()
	src := WorkflowSource(workflows)

	t.Run("Len", func(t *testing.T) {
		assert.Equal(t, 3, src.Len())
	})

	t.Run("String", func(t *testing.T) {
		s := src.String(0)
		assert.Contains(t, s, "deploy-app")
		assert.Contains(t, s, "Deploy application to kubernetes")
		assert.Contains(t, s, "k8s")
		assert.Contains(t, s, "deploy")
		assert.Contains(t, s, "kubectl apply -f deployment.yaml")
	})
}
