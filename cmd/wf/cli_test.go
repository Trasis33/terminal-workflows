package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRoot creates a fresh Cobra root command wired to a YAMLStore
// backed by the given temp directory. Each test gets an isolated store
// so tests never interfere with each other or with ~/.config/wf/.
func newTestRoot(basePath string) (*cobra.Command, *store.YAMLStore) {
	s := store.NewYAMLStore(basePath)

	root := &cobra.Command{
		Use:   "wf",
		Short: "Terminal workflow manager",
	}

	// Register add command
	add := &cobra.Command{
		Use:   "add",
		Short: "Create a new workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			command, _ := cmd.Flags().GetString("command")
			description, _ := cmd.Flags().GetString("description")
			tags, _ := cmd.Flags().GetStringSlice("tag")
			folder, _ := cmd.Flags().GetString("folder")

			if name == "" {
				return cmd.Help()
			}
			if command == "" {
				return cmd.Help()
			}

			wf := &store.Workflow{
				Name:        name,
				Command:     command,
				Description: description,
				Tags:        tags,
			}

			// Auto-extract parameters
			params := template.ExtractParams(command)
			for _, p := range params {
				wf.Args = append(wf.Args, store.Arg{
					Name:    p.Name,
					Default: p.Default,
				})
			}

			storeName := name
			if folder != "" {
				storeName = folder + "/" + name
				wf.Name = storeName
			}

			if err := s.Save(wf); err != nil {
				return err
			}
			wf.Name = name
			cmd.Printf("Created %s\n", name)
			return nil
		},
	}
	add.Flags().StringP("name", "n", "", "workflow name")
	add.Flags().StringP("command", "c", "", "command template")
	add.Flags().StringP("description", "d", "", "description")
	add.Flags().StringSliceP("tag", "t", nil, "tags")
	add.Flags().StringP("folder", "f", "", "subfolder path")

	// Register edit command
	edit := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			wf, err := s.Get(name)
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("command") {
				command, _ := cmd.Flags().GetString("command")
				wf.Command = command
				params := template.ExtractParams(command)
				wf.Args = nil
				for _, p := range params {
					wf.Args = append(wf.Args, store.Arg{
						Name:    p.Name,
						Default: p.Default,
					})
				}
			}
			if cmd.Flags().Changed("description") {
				description, _ := cmd.Flags().GetString("description")
				wf.Description = description
			}
			if cmd.Flags().Changed("tag") {
				tags, _ := cmd.Flags().GetStringSlice("tag")
				wf.Tags = tags
			}
			if cmd.Flags().Changed("add-tag") {
				tag, _ := cmd.Flags().GetString("add-tag")
				found := false
				for _, t := range wf.Tags {
					if t == tag {
						found = true
						break
					}
				}
				if !found {
					wf.Tags = append(wf.Tags, tag)
				}
			}
			if cmd.Flags().Changed("remove-tag") {
				tag, _ := cmd.Flags().GetString("remove-tag")
				filtered := make([]string, 0, len(wf.Tags))
				for _, t := range wf.Tags {
					if t != tag {
						filtered = append(filtered, t)
					}
				}
				wf.Tags = filtered
			}

			if err := s.Save(wf); err != nil {
				return err
			}
			cmd.Printf("Updated %s\n", name)
			return nil
		},
	}
	edit.Flags().StringP("command", "c", "", "replace command")
	edit.Flags().StringP("description", "d", "", "replace description")
	edit.Flags().StringSliceP("tag", "t", nil, "replace all tags")
	edit.Flags().String("add-tag", "", "add a tag")
	edit.Flags().String("remove-tag", "", "remove a tag")

	// Register rm command (force-only for tests, no interactive prompt)
	rm := &cobra.Command{
		Use:   "rm [name]",
		Short: "Delete a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return cmd.Help()
			}
			if err := s.Delete(name); err != nil {
				return err
			}
			cmd.Printf("Deleted %s\n", name)
			return nil
		},
	}
	rm.Flags().BoolP("force", "f", false, "skip confirmation")

	// Register list command
	list := &cobra.Command{
		Use:   "list",
		Short: "List workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			workflows, err := s.List()
			if err != nil {
				return err
			}
			if len(workflows) == 0 {
				cmd.Println("No workflows found")
				return nil
			}
			for _, wf := range workflows {
				tags := ""
				if len(wf.Tags) > 0 {
					tags = "  [" + joinTags(wf.Tags) + "]"
				}
				cmd.Printf("%s  %s%s\n", wf.Name, wf.Description, tags)
			}
			return nil
		},
	}

	root.AddCommand(add, edit, rm, list)
	return root, s
}

func joinTags(tags []string) string {
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += ", "
		}
		result += t
	}
	return result
}

// execCmd runs a Cobra command with the given args and returns stdout+stderr output.
func execCmd(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// ─── Full CRUD Cycle ────────────────────────────────────────────────────────

func TestCLI_FullCRUDCycle(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// ADD
	out, err := execCmd(root, "add", "--name", "deploy-staging", "--command", "kubectl apply -f deploy.yaml", "--description", "Deploy to staging", "--tag", "k8s,deploy")
	require.NoError(t, err)
	assert.Contains(t, out, "Created deploy-staging")

	// Verify YAML file exists on disk
	wf, err := s.Get("deploy-staging")
	require.NoError(t, err)
	assert.Equal(t, "deploy-staging", wf.Name)
	assert.Equal(t, "kubectl apply -f deploy.yaml", wf.Command)
	assert.Equal(t, "Deploy to staging", wf.Description)
	assert.Equal(t, []string{"k8s", "deploy"}, wf.Tags)

	// EDIT (update description via flag)
	root2, _ := newTestRoot(dir)
	out, err = execCmd(root2, "edit", "deploy-staging", "--description", "Deploy to staging v2")
	require.NoError(t, err)
	assert.Contains(t, out, "Updated deploy-staging")

	// Verify update persisted
	wf, err = s.Get("deploy-staging")
	require.NoError(t, err)
	assert.Equal(t, "Deploy to staging v2", wf.Description)

	// DELETE
	root3, _ := newTestRoot(dir)
	out, err = execCmd(root3, "rm", "deploy-staging", "--force")
	require.NoError(t, err)
	assert.Contains(t, out, "Deleted deploy-staging")

	// Verify file removed
	_, err = s.Get("deploy-staging")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ─── Multiline Command ─────────────────────────────────────────────────────

func TestCLI_MultilineCommandRoundTrip(t *testing.T) {
	dir := t.TempDir()
	_, s := newTestRoot(dir)

	// CLI flags can't contain literal newlines, so test via store.Save directly
	// then verify the full pipeline reads it back correctly.
	multilineCmd := "echo \"step 1\"\necho \"step 2\"\necho \"step 3\""

	wf := &store.Workflow{
		Name:        "multi-step",
		Command:     multilineCmd,
		Description: "A multiline deploy script",
		Tags:        []string{"deploy"},
	}
	require.NoError(t, s.Save(wf))

	// Read back via store
	got, err := s.Get("multi-step")
	require.NoError(t, err)
	assert.Equal(t, multilineCmd, got.Command, "multiline command must survive YAML round-trip (STOR-04)")

	// Verify the file actually exists on disk
	_, err = os.Stat(filepath.Join(dir, "multi-step.yaml"))
	require.NoError(t, err)
}

// ─── Parameterized Workflow ─────────────────────────────────────────────────

func TestCLI_ParameterizedWorkflow(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Add workflow with parameters (PARM-01, PARM-02)
	_, err := execCmd(root, "add", "--name", "ssh-connect", "--command", "ssh {{host:localhost}} -p {{port:22}}")
	require.NoError(t, err)

	wf, err := s.Get("ssh-connect")
	require.NoError(t, err)
	assert.Equal(t, "ssh {{host:localhost}} -p {{port:22}}", wf.Command)

	// Verify auto-extracted args
	require.Len(t, wf.Args, 2)
	assert.Equal(t, "host", wf.Args[0].Name)
	assert.Equal(t, "localhost", wf.Args[0].Default)
	assert.Equal(t, "port", wf.Args[1].Name)
	assert.Equal(t, "22", wf.Args[1].Default)

	// Verify template rendering works with these args
	rendered := template.Render(wf.Command, map[string]string{
		"host": "prod-server",
		"port": "2222",
	})
	assert.Equal(t, "ssh prod-server -p 2222", rendered)
}

// ─── Duplicate Parameter Deduplication ──────────────────────────────────────

func TestCLI_DuplicateParamDeduplication(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Add workflow with duplicate param name (PARM-05)
	_, err := execCmd(root, "add", "--name", "echo-twice", "--command", "echo {{msg}} && echo {{msg}}")
	require.NoError(t, err)

	wf, err := s.Get("echo-twice")
	require.NoError(t, err)

	// Should have only ONE arg entry for "msg"
	require.Len(t, wf.Args, 1, "duplicate params should be deduplicated")
	assert.Equal(t, "msg", wf.Args[0].Name)

	// Verify both occurrences get rendered
	rendered := template.Render(wf.Command, map[string]string{"msg": "hello"})
	assert.Equal(t, "echo hello && echo hello", rendered)
}

// ─── Folder Organization ────────────────────────────────────────────────────

func TestCLI_FolderOrganization(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Save workflow in subfolder (ORGN-02)
	_, err := execCmd(root, "add", "--name", "deploy", "--command", "kubectl apply -f deploy.yaml", "--folder", "infra")
	require.NoError(t, err)

	// Verify file exists at expected nested path
	expectedPath := filepath.Join(dir, "infra", "deploy.yaml")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "workflow file should exist at infra/deploy.yaml")

	// Verify workflow is accessible via store
	wf, err := s.Get("infra/deploy")
	require.NoError(t, err)
	assert.Equal(t, "kubectl apply -f deploy.yaml", wf.Command)

	// List should include nested workflows
	workflows, err := s.List()
	require.NoError(t, err)
	assert.Len(t, workflows, 1, "list should find workflow in subfolder")
}

// ─── Two-level Nested Folder ────────────────────────────────────────────────

func TestCLI_TwoLevelNestedFolder(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Save in 2-level subfolder
	_, err := execCmd(root, "add", "--name", "restart", "--command", "docker restart {{container}}", "--folder", "infra/docker")
	require.NoError(t, err)

	expectedPath := filepath.Join(dir, "infra", "docker", "restart.yaml")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "workflow file should exist at infra/docker/restart.yaml")

	wf, err := s.Get("infra/docker/restart")
	require.NoError(t, err)
	assert.Equal(t, "docker restart {{container}}", wf.Command)

	// List should find it
	workflows, err := s.List()
	require.NoError(t, err)
	assert.Len(t, workflows, 1)
}

// ─── Tags ───────────────────────────────────────────────────────────────────

func TestCLI_Tags(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Add workflow with multiple tags (ORGN-01)
	_, err := execCmd(root, "add", "--name", "docker-build", "--command", "docker build -t {{image}} .", "--tag", "docker,build,ci")
	require.NoError(t, err)

	wf, err := s.Get("docker-build")
	require.NoError(t, err)
	assert.Equal(t, []string{"docker", "build", "ci"}, wf.Tags)

	// Edit: add a tag
	root2, _ := newTestRoot(dir)
	_, err = execCmd(root2, "edit", "docker-build", "--add-tag", "staging")
	require.NoError(t, err)

	wf, err = s.Get("docker-build")
	require.NoError(t, err)
	assert.Contains(t, wf.Tags, "staging")
	assert.Len(t, wf.Tags, 4)

	// Edit: remove a tag
	root3, _ := newTestRoot(dir)
	_, err = execCmd(root3, "edit", "docker-build", "--remove-tag", "ci")
	require.NoError(t, err)

	wf, err = s.Get("docker-build")
	require.NoError(t, err)
	assert.NotContains(t, wf.Tags, "ci")
	assert.Len(t, wf.Tags, 3)

	// Edit: replace all tags
	root4, _ := newTestRoot(dir)
	_, err = execCmd(root4, "edit", "docker-build", "--tag", "production,final")
	require.NoError(t, err)

	wf, err = s.Get("docker-build")
	require.NoError(t, err)
	assert.Equal(t, []string{"production", "final"}, wf.Tags)
}

// ─── Norway Problem (Regression) ────────────────────────────────────────────

func TestCLI_NorwayProblem(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Commands containing words that YAML might coerce to booleans
	testCases := []struct {
		name    string
		command string
	}{
		{"redis-no", "redis-cli CONFIG SET appendonly no"},
		{"set-yes", "echo yes"},
		{"toggle-off", "systemctl disable --now off"},
		{"toggle-on", "echo on"},
		{"bare-true", "true"},
		{"bare-false", "false"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Fresh root for each to avoid flag state carryover
			root, _ = newTestRoot(dir)
			_, err := execCmd(root, "add", "--name", tc.name, "--command", tc.command)
			require.NoError(t, err)

			wf, err := s.Get(tc.name)
			require.NoError(t, err)
			assert.Equal(t, tc.command, wf.Command,
				"Norway Problem: command %q must not be corrupted to boolean", tc.command)
		})
	}
}

// ─── Nested Quotes ──────────────────────────────────────────────────────────

func TestCLI_NestedQuotes(t *testing.T) {
	dir := t.TempDir()
	_, s := newTestRoot(dir)

	testCases := []struct {
		name    string
		command string
	}{
		{
			"docker-psql",
			`docker exec -it mydb psql -c "SELECT * FROM users WHERE name = 'test'"`,
		},
		{
			"pipe-chain",
			`cat foo.txt | grep "pattern" | awk '{print $2}' > output.txt`,
		},
		{
			"subshell",
			`echo "today is $(date +%Y-%m-%d)"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wf := &store.Workflow{
				Name:    tc.name,
				Command: tc.command,
			}
			require.NoError(t, s.Save(wf))

			got, err := s.Get(tc.name)
			require.NoError(t, err)
			assert.Equal(t, tc.command, got.Command,
				"nested quotes must survive YAML round-trip")
		})
	}
}

// ─── Edit Re-extracts Params ────────────────────────────────────────────────

func TestCLI_EditReextractsParams(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Create workflow with params
	_, err := execCmd(root, "add", "--name", "connect", "--command", "ssh {{host}}")
	require.NoError(t, err)

	wf, err := s.Get("connect")
	require.NoError(t, err)
	require.Len(t, wf.Args, 1)
	assert.Equal(t, "host", wf.Args[0].Name)

	// Edit command to have different params
	root2, _ := newTestRoot(dir)
	_, err = execCmd(root2, "edit", "connect", "--command", "curl {{url:http://localhost}} -H 'Token: {{token}}'")
	require.NoError(t, err)

	wf, err = s.Get("connect")
	require.NoError(t, err)
	require.Len(t, wf.Args, 2, "args should be re-extracted after command edit")
	assert.Equal(t, "url", wf.Args[0].Name)
	assert.Equal(t, "http://localhost", wf.Args[0].Default)
	assert.Equal(t, "token", wf.Args[1].Name)
}

// ─── List Command ───────────────────────────────────────────────────────────

func TestCLI_ListWorkflows(t *testing.T) {
	dir := t.TempDir()
	root, _ := newTestRoot(dir)

	// Empty list
	out, err := execCmd(root, "list")
	require.NoError(t, err)
	assert.Contains(t, out, "No workflows found")

	// Add workflows
	root2, _ := newTestRoot(dir)
	_, err = execCmd(root2, "add", "--name", "alpha", "--command", "echo alpha", "--description", "First workflow", "--tag", "test")
	require.NoError(t, err)

	root3, _ := newTestRoot(dir)
	_, err = execCmd(root3, "add", "--name", "beta", "--command", "echo beta", "--description", "Second workflow")
	require.NoError(t, err)

	// List should show both
	root4, _ := newTestRoot(dir)
	out, err = execCmd(root4, "list")
	require.NoError(t, err)
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
	assert.Contains(t, out, "First workflow")
	assert.Contains(t, out, "test")
}

// ─── Error: Get Non-Existent ────────────────────────────────────────────────

func TestCLI_ErrorNonExistentWorkflow(t *testing.T) {
	dir := t.TempDir()
	root, _ := newTestRoot(dir)

	// Edit non-existent
	_, err := execCmd(root, "edit", "does-not-exist", "--description", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Rm non-existent
	root2, _ := newTestRoot(dir)
	_, err = execCmd(root2, "rm", "does-not-exist", "--force")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ─── Empty Fields ───────────────────────────────────────────────────────────

func TestCLI_EmptyOptionalFields(t *testing.T) {
	dir := t.TempDir()
	root, s := newTestRoot(dir)

	// Add with only required fields (name, command)
	_, err := execCmd(root, "add", "--name", "minimal", "--command", "echo hello")
	require.NoError(t, err)

	wf, err := s.Get("minimal")
	require.NoError(t, err)
	assert.Equal(t, "minimal", wf.Name)
	assert.Equal(t, "echo hello", wf.Command)
	assert.Empty(t, wf.Description)
	assert.Empty(t, wf.Tags)
	assert.Empty(t, wf.Args)
}

// ─── Special Characters in Command ──────────────────────────────────────────

func TestCLI_SpecialCharactersInCommand(t *testing.T) {
	dir := t.TempDir()
	_, s := newTestRoot(dir)

	testCases := []struct {
		name    string
		command string
	}{
		{"ampersand", "cmd1 && cmd2 || cmd3"},
		{"semicolons", "echo a; echo b; echo c"},
		{"backticks", "echo `hostname`"},
		{"dollar-signs", "echo $HOME $USER"},
		{"backslashes", `echo "line1\nline2"`},
		{"unicode", "echo こんにちは 🚀"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wf := &store.Workflow{
				Name:    tc.name,
				Command: tc.command,
			}
			require.NoError(t, s.Save(wf))

			got, err := s.Get(tc.name)
			require.NoError(t, err)
			assert.Equal(t, tc.command, got.Command,
				"special characters must survive YAML round-trip")
		})
	}
}

// ─── Full Pipeline: Add → List → Rm → List ─────────────────────────────────

func TestCLI_FullPipelineAddListRmList(t *testing.T) {
	dir := t.TempDir()

	// Add a workflow
	root1, _ := newTestRoot(dir)
	_, err := execCmd(root1, "add", "--name", "demo", "--command", "echo {{name}}", "--tag", "test")
	require.NoError(t, err)

	// List shows it
	root2, _ := newTestRoot(dir)
	out, err := execCmd(root2, "list")
	require.NoError(t, err)
	assert.Contains(t, out, "demo")

	// Remove it
	root3, _ := newTestRoot(dir)
	_, err = execCmd(root3, "rm", "demo", "--force")
	require.NoError(t, err)

	// List shows empty
	root4, _ := newTestRoot(dir)
	out, err = execCmd(root4, "list")
	require.NoError(t, err)
	assert.Contains(t, out, "No workflows found")
}
