package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/pkg/behaviourtest/world"
)

func TestJiraPoll_CommentSlashCommand(t *testing.T) {
	t.Parallel()

	w := &world.World{Logf: t.Logf}

	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() {
		CleanupScenario(w)
	})

	require.NoError(t, givenJiraIssue(w, "PROJ-101", "bug"))
	require.NoError(t, whenJiraComment(w, "PROJ-101", "/fs-triage check acceptance criteria"))
	require.NoError(t, whenJiraPollerRuns(context.Background(), w))
	require.NoError(t, thenDispatchContains(w, "triage", "PROJ-101"))
}

func TestJiraPoll_LabelChange(t *testing.T) {
	t.Parallel()

	w := &world.World{Logf: t.Logf}

	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() {
		CleanupScenario(w)
	})

	require.NoError(t, givenJiraIssue(w, "PROJ-202", "backlog"))
	require.NoError(t, whenJiraLabelAdded(w, "PROJ-202", "ready-to-code"))
	require.NoError(t, whenJiraPollerRuns(context.Background(), w))
	require.NoError(t, thenDispatchContains(w, "code", "PROJ-202"))
	require.NoError(t, thenDispatchNotContains(w, "PROJ-999"))
}

func TestJiraPoll_NoDispatchForUnknownIssue(t *testing.T) {
	t.Parallel()

	w := &world.World{Logf: t.Logf}

	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() {
		CleanupScenario(w)
	})

	require.NoError(t, givenJiraIssue(w, "PROJ-300", "bug"))
	// No comment or label change — only "opened" event, which doesn't route.
	require.NoError(t, whenJiraPollerRuns(context.Background(), w))
	require.NoError(t, thenDispatchNotContains(w, "PROJ-300"))
}

func TestJiraPoll_InvalidEventSkipped(t *testing.T) {
	t.Parallel()

	w := &world.World{Logf: t.Logf}

	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() {
		CleanupScenario(w)
	})

	require.NoError(t, givenJiraIssue(w, "PROJ-505", "bug"))
	require.NoError(t, givenJiraIssueID(w, "PROJ-505", "not-a-number"))
	require.NoError(t, givenJiraIssue(w, "PROJ-506", "bug"))
	require.NoError(t, whenJiraComment(w, "PROJ-505", "/fs-triage check"))
	require.NoError(t, whenJiraComment(w, "PROJ-506", "/fs-triage check"))
	require.NoError(t, whenJiraPollerRuns(context.Background(), w))
	require.NoError(t, thenDispatchNotContains(w, "PROJ-505"))
	require.NoError(t, thenDispatchContains(w, "triage", "PROJ-506"))
}

func TestJiraPoll_IssueIDUnknownIssue(t *testing.T) {
	t.Parallel()

	w := &world.World{Logf: t.Logf}

	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() {
		CleanupScenario(w)
	})

	err := givenJiraIssueID(w, "PROJ-404", "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Error(t, givenJiraIssueID(&world.World{Logf: t.Logf}, "PROJ-1", "1"))
}

// stubCLI writes a shell script that prints out and exits with code, and
// returns its path, standing in for the suite's fullsend binary.
func stubCLI(t *testing.T, out string, code int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fullsend")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' %q\nexit %d\n", out, code)
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestJiraPollCLI_RefusalRecordedAndChecked(t *testing.T) {
	t.Parallel()
	w := &world.World{Logf: t.Logf}
	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() { CleanupScenario(w) })

	assert.ErrorContains(t, thenJiraPollCLIFails(w, "x"), "has not run")

	w.CLIBinary = stubCLI(t, "Error: --target-repo \"test-org/..\" must not contain '..'", 1)
	require.NoError(t, whenJiraPollCLIRuns(context.Background(), w, "test-org/.."))
	assert.Equal(t, 1, w.JiraPollCLIExitCode)
	require.NoError(t, thenJiraPollCLIFails(w, "must not contain '..'"))
	assert.ErrorContains(t, thenJiraPollCLIFails(w, "something else"), "without \"something else\"")

	// A run that succeeds fails the step, and so does one that wrote output.
	w.CLIBinary = stubCLI(t, "poll complete", 0)
	require.NoError(t, whenJiraPollCLIRuns(context.Background(), w, "test-org/.."))
	assert.ErrorContains(t, thenJiraPollCLIFails(w, "must not contain"), "succeeded")
	w.JiraPollCLIExitCode = 1
	w.JiraPollCLIOutput = "must not contain '..'"
	require.NoError(t, os.WriteFile(w.JiraPollCLIOutputPath, []byte("[]"), 0o644))
	assert.ErrorContains(t, thenJiraPollCLIFails(w, "must not contain"), "wrote dispatch output")
}

func TestJiraPollCLI_Preconditions(t *testing.T) {
	t.Parallel()
	assert.ErrorContains(t, whenJiraPollCLIRuns(context.Background(), &world.World{}, "a/b"), "not initialized")
	w := &world.World{Logf: t.Logf}
	require.NoError(t, givenMockJiraServer(w))
	t.Cleanup(func() { CleanupScenario(w) })
	assert.ErrorContains(t, whenJiraPollCLIRuns(context.Background(), w, "a/b"), "no fullsend CLI binary")
}

func TestJiraPoll_MissingMockServer(t *testing.T) {
	t.Parallel()

	w := &world.World{Logf: t.Logf}

	err := givenJiraIssue(w, "PROJ-1", "bug")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}
