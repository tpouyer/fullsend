package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fetchOnlyAllowlistComment returns the explanatory comment at the top of
// TestLintCommand_FetchOnlyHarnessRemoteResourcesNotSubset in lint_test.go.
// The comment documents why lint checks a fetch-only harness's allowlist
// against config.yaml; it was introduced by PR #9291, so its citation must
// name that PR number, not a commit hash formatted as an issue link.
func fetchOnlyAllowlistComment(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("lint_test.go")
	require.NoError(t, err)
	src := string(raw)
	idx := strings.Index(src, "func TestLintCommand_FetchOnlyHarnessRemoteResourcesNotSubset(")
	require.NotEqual(t, -1, idx, "TestLintCommand_FetchOnlyHarnessRemoteResourcesNotSubset not found in lint_test.go")
	var lines []string
	for _, line := range strings.Split(src[idx:], "\n")[1:] {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			break
		}
		lines = append(lines, trimmed)
	}
	require.NotEmpty(t, lines, "expected an explanatory comment at the top of TestLintCommand_FetchOnlyHarnessRemoteResourcesNotSubset")
	return strings.Join(lines, "\n")
}

// TestLintCommand_FetchOnlyComment_CitesPRNumber: the comment inside
// TestLintCommand_FetchOnlyHarnessRemoteResourcesNotSubset must cite the PR
// number (#9291) that introduced the fetch-only allowlist lint check.
func TestLintCommand_FetchOnlyComment_CitesPRNumber(t *testing.T) {
	assert.Contains(t, fetchOnlyAllowlistComment(t), "(#9291)",
		"the comment must cite the PR number (#9291), not a commit hash")
}

// TestLintCommand_FetchOnlyComment_CitesNoCommitHashAsIssueLink: the comment
// must not cite a commit hash formatted as an issue link.
func TestLintCommand_FetchOnlyComment_CitesNoCommitHashAsIssueLink(t *testing.T) {
	assert.NotContains(t, fetchOnlyAllowlistComment(t), "(#7395801)",
		"the comment must not cite commit 7395801 formatted as an issue link")
}
