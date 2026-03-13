package peertest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGitHubIssueURL(t *testing.T) {
	owner, repo, number, err := parseGitHubIssueURL("https://github.com/wso2-enterprise/wso2-apim-internal/issues/15738")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if owner != "wso2-enterprise" || repo != "wso2-apim-internal" || number != 15738 {
		t.Fatalf("unexpected parsed issue ref: %s %s %d", owner, repo, number)
	}
}

func TestExtractPeerTestIssueSpec(t *testing.T) {
	body := "Notes\n\n```yaml\npeer_test_updates:\n  product_version: \"4.4.0\"\n  updates:\n    - update_id: \"17125\"\n      deliverables:\n        - path: \"repository/components/plugins/org.wso2.carbon.apimgt.gateway_9.30.67.160.jar\"\n          action: \"modified\"\n```\n"

	spec, err := extractPeerTestIssueSpec(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if spec.ProductVersion != "4.4.0" {
		t.Fatalf("expected product version 4.4.0, got %s", spec.ProductVersion)
	}
	if len(spec.Updates) != 1 || spec.Updates[0].UpdateID != "17125" {
		t.Fatalf("unexpected updates payload: %+v", spec.Updates)
	}
}

func TestCompareVersionStrings(t *testing.T) {
	if compareVersionStrings("9.30.67.160", "9.30.67.159") <= 0 {
		t.Fatal("expected 160 to be greater than 159")
	}
	if compareVersionStrings("1.2.11.wso2v25_8", "1.2.11.wso2v25_7") <= 0 {
		t.Fatal("expected wso2v25_8 to be greater than wso2v25_7")
	}
}

func TestVerifyPeerTestIssueAgainstProduct(t *testing.T) {
	productRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(productRoot, "repository/components/plugins/org.wso2.carbon.apimgt.gateway_9.30.67.160.jar"))
	mustWriteFile(t, filepath.Join(productRoot, "repository/deployment/server/webapps/api#am#publisher.war"))

	spec := &peerTestIssueSpec{
		ProductVersion: "4.4.0",
		Updates: []peerTestIssueUpdate{
			{
				UpdateID: "17124",
				Deliverables: []peerTestIssueDeliverable{
					{Path: "repository/components/plugins/org.wso2.carbon.apimgt.gateway_9.30.67.159.jar"},
				},
			},
			{
				UpdateID: "17125",
				Deliverables: []peerTestIssueDeliverable{
					{Path: "repository/components/plugins/org.wso2.carbon.apimgt.gateway_9.30.67.160.jar"},
					{Path: "repository/deployment/server/webapps/api#am#publisher.war"},
				},
			},
			{
				UpdateID: "17126",
				Deliverables: []peerTestIssueDeliverable{
					{Path: "repository/deployment/server/webapps/api#am#publisher.war"},
				},
			},
		},
	}

	report := verifyPeerTestIssueAgainstProduct(productRoot, spec)
	if len(report.LatestJarMissing) != 0 {
		t.Fatalf("expected latest jar to exist, got %+v", report.LatestJarMissing)
	}
	if len(report.Missing) != 0 {
		t.Fatalf("expected no missing deliverables after latest-version suppression, got %+v", report.Missing)
	}
	if len(report.JarConflicts) != 1 {
		t.Fatalf("expected one jar conflict group, got %+v", report.JarConflicts)
	}
	if len(report.ExactDuplicates) != 1 {
		t.Fatalf("expected one exact duplicate path, got %+v", report.ExactDuplicates)
	}
}

func TestResolveSmokeArtifactDirRequiresIssueFolder(t *testing.T) {
	productCfg := &resolvedProductConfig{workspaceRoot: t.TempDir()}
	_, err := resolveSmokeArtifactDir(productCfg, "15426", "")
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing issue dir error, got %v", err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
