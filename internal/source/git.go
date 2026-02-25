package source

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// gitAvailable returns true if the git binary is on the PATH.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// gitClone runs a shallow clone of url into dest with a 2-minute timeout.
// GIT_TERMINAL_PROMPT=0 prevents interactive auth prompts from hanging.
func gitClone(ctx context.Context, url, dest string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, dest)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", url, err)
	}
	return nil
}

// gitPull runs git pull --ff-only in repoDir with a 30-second timeout.
// Returns the combined stdout+stderr output for display.
func gitPull(ctx context.Context, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "pull", "--ff-only")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git pull in %s: %w\n%s", repoDir, err, out)
	}
	return string(out), nil
}

// deriveAlias extracts a short alias from a git URL.
// "https://github.com/team/workflows.git" -> "workflows"
// "git@github.com:user/my-commands.git"   -> "my-commands"
func deriveAlias(url string) string {
	u := strings.TrimSuffix(url, ".git")
	parts := strings.FieldsFunc(u, func(r rune) bool {
		return r == '/' || r == ':'
	})
	if len(parts) == 0 {
		return "remote"
	}
	return parts[len(parts)-1]
}
