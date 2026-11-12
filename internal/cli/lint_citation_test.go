package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readLintTestSource returns the whole of lint_test.go. The citation
// assertions below run against the entire file, so they hold wherever the
// explanatory comment sits.
func readLintTestSource(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("lint_test.go")
	require.NoError(t, err)
	return string(raw)
}

// TestLintCommand_FetchOnlyComment_CitesPRNumber: the comment inside
// TestLintCommand_FetchOnlyHarnessRemoteResourcesNotSubset must cite the PR
// number (#9291) that introduced the fetch-only allowlist lint check.
func TestLintCommand_FetchOnlyComment_CitesPRNumber(t *testing.T) {
	assert.Contains(t, readLintTestSource(t), "(#9291)",
		"the comment must name the PR number (#9291), not an issue-style reference")
}

// TestLintCommand_FetchOnlyComment_CitesNoIssueStyleReference: the comment
// must not cite the PR through an issue-style reference in place of the
// PR number.
func TestLintCommand_FetchOnlyComment_CitesNoIssueStyleReference(t *testing.T) {
	assert.NotRegexp(t, `\(#\d{7,}\)`, readLintTestSource(t),
		"the comment must name the PR number (#9291), not an issue-style reference")
}
