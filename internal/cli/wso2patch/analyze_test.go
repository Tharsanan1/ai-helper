package wso2patch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGitHubIssueURL_Valid(t *testing.T) {
	owner, repo, issueNumber, err := parseGitHubIssueURL("https://github.com/wso2/product-apim/issues/123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if owner != "wso2" || repo != "product-apim" || issueNumber != 123 {
		t.Fatalf("unexpected parse result: owner=%q repo=%q issue=%d", owner, repo, issueNumber)
	}
}

func TestParseGitHubIssueURL_WithQueryAndFragment(t *testing.T) {
	owner, repo, issueNumber, err := parseGitHubIssueURL("https://github.com/wso2/carbon-apimgt/issues/456?foo=bar#section")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if owner != "wso2" || repo != "carbon-apimgt" || issueNumber != 456 {
		t.Fatalf("unexpected parse result: owner=%q repo=%q issue=%d", owner, repo, issueNumber)
	}
}

func TestParseGitHubIssueURL_InvalidNonIssue(t *testing.T) {
	_, _, _, err := parseGitHubIssueURL("https://github.com/wso2/product-apim/pull/123")
	if err == nil {
		t.Fatal("expected error for non-issue URL")
	}
	if !strings.Contains(err.Error(), "github issue URL") {
		t.Fatalf("expected github issue URL error, got %v", err)
	}
}

func TestValidatePatchRoot_AllReposPresent(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "carbon-apimgt"))
	mustMkdir(t, filepath.Join(root, "product-apim"))

	err := validatePatchRoot(root, []string{"carbon-apimgt", "product-apim"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidatePatchRoot_MissingRepo(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "carbon-apimgt"))

	err := validatePatchRoot(root, []string{"carbon-apimgt", "product-apim"})
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
	if !strings.Contains(err.Error(), "product-apim") {
		t.Fatalf("expected missing repo in error, got %v", err)
	}
}

func TestResolveAnalyzeScope_FromPatchRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "carbon-apimgt"))
	mustMkdir(t, filepath.Join(root, "product-apim"))

	scope, err := resolveAnalyzeScope(root, []string{"carbon-apimgt", "product-apim"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if scope.patchRoot != root || scope.runRoot != root {
		t.Fatalf("unexpected roots: patch=%q run=%q", scope.patchRoot, scope.runRoot)
	}
	if len(scope.repos) != 2 || scope.repos[0] != "carbon-apimgt" || scope.repos[1] != "product-apim" {
		t.Fatalf("unexpected repos: %#v", scope.repos)
	}
}

func TestResolveAnalyzeScope_FromRepoSubfolder(t *testing.T) {
	root := t.TempDir()
	carbonPath := filepath.Join(root, "carbon-apimgt")
	mustMkdir(t, carbonPath)
	mustMkdir(t, filepath.Join(root, "product-apim"))

	scope, err := resolveAnalyzeScope(carbonPath, []string{"carbon-apimgt", "product-apim"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if scope.patchRoot != root || scope.runRoot != carbonPath {
		t.Fatalf("unexpected roots: patch=%q run=%q", scope.patchRoot, scope.runRoot)
	}
	if len(scope.repos) != 1 || scope.repos[0] != "carbon-apimgt" {
		t.Fatalf("unexpected repos: %#v", scope.repos)
	}
}

func TestBuildAnalyzePrompt_AppendsUserPrompt(t *testing.T) {
	userPrompt := "Prioritize compatibility risks and migration concerns."
	prompt := buildAnalyzePrompt(
		"/tmp/patch-root/carbon-apimgt",
		[]string{"carbon-apimgt"},
		nil,
		"",
		"",
		userPrompt,
	)

	if !strings.Contains(prompt, "Additional user instructions:") {
		t.Fatalf("expected additional user instructions section in prompt")
	}
	if !strings.Contains(prompt, userPrompt) {
		t.Fatalf("expected user prompt to be appended in final prompt")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}
