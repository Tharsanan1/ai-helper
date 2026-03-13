package peertest

import (
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
)

func TestParsePeerTestIssueNumber_GitHubIssue(t *testing.T) {
	got, err := parsePeerTestIssueNumber("https://github.com/wso2/product-apim/issues/15426")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "15426" {
		t.Fatalf("expected issue number 15426, got %q", got)
	}
}

func TestParsePeerTestIssueNumber_JiraStyleIssue(t *testing.T) {
	got, err := parsePeerTestIssueNumber("https://git.example.com/browse/PEERTEST-421")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "421" {
		t.Fatalf("expected issue number 421, got %q", got)
	}
}

func TestParsePeerTestIssueNumber_Invalid(t *testing.T) {
	_, err := parsePeerTestIssueNumber("not-a-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "full URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveProductConfig_DerivesWorkspaceAndRunWorkingDirs(t *testing.T) {
	cfg := &config.Config{
		PeerTest: config.PeerTestConfig{
			Products: map[string]config.PeerTestProductConfig{
				"4.4.0": {
					PackPath: "~/Documents/wso2/apim/4.4.0/wso2am-4.4.0.13.zip",
					Steps:    []string{"./wso2update_darwin"},
					RunSteps: []string{"sh api-manager.sh"},
				},
			},
		},
	}

	resolved, err := resolveProductConfig(cfg, "4.4.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	home, err := expandPath("~")
	if err != nil {
		t.Fatalf("failed to resolve home: %v", err)
	}
	wantPack := filepath.Join(home, "Documents", "wso2", "apim", "4.4.0", "wso2am-4.4.0.13.zip")
	wantWorkspace := filepath.Join(home, "Documents", "wso2", "apim", "4.4.0", "peertests")
	if resolved.packPath != wantPack {
		t.Fatalf("expected pack path %s, got %s", wantPack, resolved.packPath)
	}
	if resolved.workspaceRoot != wantWorkspace {
		t.Fatalf("expected workspace root %s, got %s", wantWorkspace, resolved.workspaceRoot)
	}
	if resolved.workingDir != "bin" {
		t.Fatalf("expected default working dir bin, got %s", resolved.workingDir)
	}
	if resolved.runWorkingDir != "bin" {
		t.Fatalf("expected default run working dir bin, got %s", resolved.runWorkingDir)
	}
}

func TestResolveProductConfig_RunModeCanWorkWithoutPackPathWhenWorkspaceConfigured(t *testing.T) {
	cfg := &config.Config{
		PeerTest: config.PeerTestConfig{
			Products: map[string]config.PeerTestProductConfig{
				"4.4.0": {
					WorkspaceRoot: "/tmp/peertests",
					RunSteps:      []string{"sh api-manager.sh"},
				},
			},
		},
	}

	resolved, err := resolveProductConfig(cfg, "4.4.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved.packPath != "" {
		t.Fatalf("expected empty pack path, got %s", resolved.packPath)
	}
	if resolved.workspaceRoot != "/tmp/peertests" {
		t.Fatalf("expected workspace root /tmp/peertests, got %s", resolved.workspaceRoot)
	}
}

func TestResolveProductConfig_SmokeDefaultsApplied(t *testing.T) {
	cfg := &config.Config{
		PeerTest: config.PeerTestConfig{
			Products: map[string]config.PeerTestProductConfig{
				"4.4.0": {
					WorkspaceRoot: "/tmp/peertests",
					RunSteps:      []string{"sh api-manager.sh"},
				},
			},
		},
	}

	resolved, err := resolveProductConfig(cfg, "4.4.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved.smokeTest.TenantDomain != "peertest.com" {
		t.Fatalf("expected default tenant domain peertest.com, got %s", resolved.smokeTest.TenantDomain)
	}
	if resolved.smokeTest.TenantAdminEmail != "peer@peertest.com" {
		t.Fatalf("expected derived tenant admin email, got %s", resolved.smokeTest.TenantAdminEmail)
	}
	if resolved.smokeTest.ScreenshotDelayMs != 1000 {
		t.Fatalf("expected default screenshot delay 1000, got %d", resolved.smokeTest.ScreenshotDelayMs)
	}
	if resolved.smokeTest.GIFFrameDelayMs != 1000 {
		t.Fatalf("expected default GIF frame delay 1000, got %d", resolved.smokeTest.GIFFrameDelayMs)
	}
	if resolved.smokeTest.SlowMo != 250 {
		t.Fatalf("expected default slow mo 250, got %d", resolved.smokeTest.SlowMo)
	}
}

func TestRenderSteps_RendersAndMasksSensitiveValues(t *testing.T) {
	cfg := &resolvedProductConfig{
		version:       "4.4.0",
		packPath:      "/tmp/wso2am.zip",
		workspaceRoot: "/tmp/peertests",
		workingDir:    "bin",
		steps: []string{
			"./wso2update_darwin -u {{username}} -p {{password}}",
			`grep "Applied " ../updates/logs/wso2update-{{today}}.log`,
		},
	}

	rendered := renderSteps(cfg.steps, cfg, "user@example.com", "super-secret", "15426")
	if len(rendered.exec) != 2 || len(rendered.display) != 2 {
		t.Fatalf("unexpected rendered step counts: exec=%d display=%d", len(rendered.exec), len(rendered.display))
	}
	if !strings.Contains(rendered.exec[0], "'user@example.com'") {
		t.Fatalf("expected quoted username in exec step, got %s", rendered.exec[0])
	}
	if !strings.Contains(rendered.exec[0], "'super-secret'") {
		t.Fatalf("expected quoted password in exec step, got %s", rendered.exec[0])
	}
	if strings.Contains(rendered.display[0], "super-secret") {
		t.Fatalf("expected password to be masked in display step, got %s", rendered.display[0])
	}
	if !strings.Contains(rendered.display[0], "<hidden>") {
		t.Fatalf("expected masked password in display step, got %s", rendered.display[0])
	}
	if !strings.Contains(rendered.display[1], time.Now().Format("02-01-2006")) {
		t.Fatalf("expected current date in rendered grep step, got %s", rendered.display[1])
	}
}

func TestRenderSteps_RunModeLeavesBlankSecretsOutOfDisplay(t *testing.T) {
	cfg := &resolvedProductConfig{
		version:       "4.4.0",
		workspaceRoot: "/tmp/peertests",
		runWorkingDir: "bin",
		runSteps:      []string{"sh api-manager.sh"},
	}

	rendered := renderSteps(cfg.runSteps, cfg, "", "", "15426")
	if len(rendered.display) != 1 {
		t.Fatalf("expected one display step, got %d", len(rendered.display))
	}
	if rendered.display[0] != "sh api-manager.sh" {
		t.Fatalf("unexpected display step: %s", rendered.display[0])
	}
}

func TestParseExportStep(t *testing.T) {
	key, value, ok := parseExportStep("export WSO2_UPDATES_UPDATE_LEVEL_STATE=TESTING")
	if !ok {
		t.Fatal("expected export step to be parsed")
	}
	if key != "WSO2_UPDATES_UPDATE_LEVEL_STATE" {
		t.Fatalf("unexpected export key: %s", key)
	}
	if value != "TESTING" {
		t.Fatalf("unexpected export value: %s", value)
	}
}

func TestParseExportStep_Invalid(t *testing.T) {
	_, _, ok := parseExportStep("./wso2update_darwin")
	if ok {
		t.Fatal("did not expect non-export step to parse as export")
	}
}

func TestShouldRerunAfterUpdateToolSelfUpdate(t *testing.T) {
	output := "Update tool client has been updated. Please re-run the tool with necessary parameters"
	if !shouldRerunAfterUpdateToolSelfUpdate(output) {
		t.Fatal("expected self-update output to trigger rerun")
	}
}

func TestShouldAllowStepRerun(t *testing.T) {
	if !shouldAllowStepRerun("./wso2update_darwin") {
		t.Fatal("expected wso2update step to allow rerun")
	}
	if shouldAllowStepRerun("grep \"Applied \" ../updates/logs/wso2update-13-03-2026.log") {
		t.Fatal("did not expect non-update command to allow rerun")
	}
}

func TestCreateGitSnapshot(t *testing.T) {
	if _, err := os.Stat("/usr/bin/git"); err != nil {
		if _, pathErr := exec.LookPath("git"); pathErr != nil {
			t.Skip("git not available in PATH")
		}
	}

	repoDir := t.TempDir()
	productFile := filepath.Join(repoDir, "product.txt")
	if err := os.WriteFile(productFile, []byte("live-updated"), 0644); err != nil {
		t.Fatalf("failed to write product file: %v", err)
	}

	if err := createGitSnapshot(repoDir, "updated live"); err != nil {
		t.Fatalf("expected git snapshot to succeed, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		t.Fatalf("expected .git directory to exist, got %v", err)
	}

	log, err := runGitInDir(repoDir, "log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatalf("failed to inspect git log: %v", err)
	}
	if log != "updated live" {
		t.Fatalf("expected commit message 'updated live', got %q", log)
	}
}

func TestSnapshotLiveUpdatedProductHook_FiresOnlyAfterSecondUpdateStep(t *testing.T) {
	if _, err := os.Stat("/usr/bin/git"); err != nil {
		if _, pathErr := exec.LookPath("git"); pathErr != nil {
			t.Skip("git not available in PATH")
		}
	}

	repoDir := t.TempDir()
	productFile := filepath.Join(repoDir, "product.txt")
	if err := os.WriteFile(productFile, []byte("after-live-update"), 0644); err != nil {
		t.Fatalf("failed to write product file: %v", err)
	}

	hook := snapshotLiveUpdatedProductHook(repoDir)
	if err := hook(0, 1, "./wso2update_darwin -u user -p secret", "./wso2update_darwin -u user -p <hidden>"); err != nil {
		t.Fatalf("expected first update hook call to succeed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("did not expect git repo to be created after first update step")
	}

	if err := hook(1, 2, "./wso2update_darwin", "./wso2update_darwin"); err != nil {
		t.Fatalf("expected second update hook call to succeed, got %v", err)
	}
	log, err := runGitInDir(repoDir, "log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatalf("failed to inspect git log after hook: %v", err)
	}
	if log != "updated live" {
		t.Fatalf("expected commit message 'updated live', got %q", log)
	}

	secondFile := filepath.Join(repoDir, "later.txt")
	if err := os.WriteFile(secondFile, []byte("testing"), 0644); err != nil {
		t.Fatalf("failed to write second file: %v", err)
	}
	if err := hook(3, 3, "./wso2update_darwin", "./wso2update_darwin"); err != nil {
		t.Fatalf("expected later hook call to be ignored, got %v", err)
	}
	status, err := runGitInDir(repoDir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("failed to inspect git status after later hook call: %v", err)
	}
	if !strings.Contains(status, "later.txt") {
		t.Fatalf("expected later file to remain uncommitted, got %q", status)
	}
}

func TestSmokeFlagHelpers(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("tenant-domain", "", "")
	cmd.Flags().Int("slow-mo", 0, "")
	cmd.Flags().Int("gif-frame-delay-ms", 0, "")

	if got := smokeFlagString(cmd, "tenant-domain", "cli.example.com", "config.example.com"); got != "config.example.com" {
		t.Fatalf("expected config value when flag not changed, got %s", got)
	}
	if got := smokeFlagInt(cmd, "slow-mo", 0, 250); got != 250 {
		t.Fatalf("expected config int when flag not changed, got %d", got)
	}
	if got := smokeFlagInt(cmd, "gif-frame-delay-ms", 0, 1000); got != 1000 {
		t.Fatalf("expected config gif delay when flag not changed, got %d", got)
	}

	if err := cmd.Flags().Set("tenant-domain", "cli.example.com"); err != nil {
		t.Fatalf("failed to set tenant-domain: %v", err)
	}
	if err := cmd.Flags().Set("slow-mo", "0"); err != nil {
		t.Fatalf("failed to set slow-mo: %v", err)
	}
	if err := cmd.Flags().Set("gif-frame-delay-ms", "500"); err != nil {
		t.Fatalf("failed to set gif-frame-delay-ms: %v", err)
	}
	if got := smokeFlagString(cmd, "tenant-domain", "cli.example.com", "config.example.com"); got != "cli.example.com" {
		t.Fatalf("expected cli value when flag changed, got %s", got)
	}
	if got := smokeFlagInt(cmd, "slow-mo", 0, 250); got != 0 {
		t.Fatalf("expected cli int when flag changed, got %d", got)
	}
	if got := smokeFlagInt(cmd, "gif-frame-delay-ms", 500, 1000); got != 500 {
		t.Fatalf("expected cli gif delay when flag changed, got %d", got)
	}
}

func TestResolveSmokeArtifactDir(t *testing.T) {
	productCfg := &resolvedProductConfig{
		version:       "4.4.0",
		workspaceRoot: t.TempDir(),
	}
	issueDir := filepath.Join(productCfg.workspaceRoot, "15426")
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		t.Fatalf("failed to create issue dir: %v", err)
	}

	gotDefault, err := resolveSmokeArtifactDir(productCfg, "15426", "")
	if err != nil {
		t.Fatalf("expected no error for default dir, got %v", err)
	}
	wantDefault := filepath.Join(issueDir, "smoketest-artifacts", "screenshots")
	if gotDefault != wantDefault {
		t.Fatalf("expected %s, got %s", wantDefault, gotDefault)
	}

	gotRelative, err := resolveSmokeArtifactDir(productCfg, "15426", "custom/screens")
	if err != nil {
		t.Fatalf("expected no error for relative dir, got %v", err)
	}
	wantRelative := filepath.Join(issueDir, "custom", "screens")
	if gotRelative != wantRelative {
		t.Fatalf("expected %s, got %s", wantRelative, gotRelative)
	}
}

func TestCreateSmokeTestGIF(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "000-existing.png")
	if err := writeTestPNG(existing, color.RGBA{R: 255, A: 255}); err != nil {
		t.Fatalf("failed to write existing png: %v", err)
	}
	baseline, err := snapshotPNGSet(dir)
	if err != nil {
		t.Fatalf("failed to snapshot dir: %v", err)
	}
	first := filepath.Join(dir, "001-first.png")
	second := filepath.Join(dir, "002-second.png")
	if err := writeTestPNG(first, color.RGBA{G: 255, A: 255}); err != nil {
		t.Fatalf("failed to write first png: %v", err)
	}
	if err := writeTestPNG(second, color.RGBA{B: 255, A: 255}); err != nil {
		t.Fatalf("failed to write second png: %v", err)
	}

	gifPath, err := createSmokeTestGIF(dir, baseline, 500)
	if err != nil {
		t.Fatalf("expected gif creation to succeed, got %v", err)
	}
	if filepath.Ext(gifPath) != ".gif" {
		t.Fatalf("expected gif output, got %s", gifPath)
	}

	file, err := os.Open(gifPath)
	if err != nil {
		t.Fatalf("failed to open gif: %v", err)
	}
	defer file.Close()

	decoded, err := gif.DecodeAll(file)
	if err != nil {
		t.Fatalf("failed to decode gif: %v", err)
	}
	if len(decoded.Image) != 2 {
		t.Fatalf("expected 2 frames in gif, got %d", len(decoded.Image))
	}
	if len(decoded.Delay) != 2 || decoded.Delay[0] != 50 {
		t.Fatalf("expected gif delay 50, got %v", decoded.Delay)
	}
}

func TestResolveSmokeTestToolDir(t *testing.T) {
	tempRoot := t.TempDir()
	toolDir := filepath.Join(tempRoot, "tools", "peertest-smoketest-4.4.0")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	original := runtimeCaller
	runtimeCaller = func(skip int) (uintptr, string, int, bool) {
		return 0, filepath.Join(tempRoot, "internal", "cli", "peertest", "peertest.go"), 0, true
	}
	t.Cleanup(func() {
		runtimeCaller = original
	})

	got, err := resolveSmokeTestToolDir("4.4.0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != toolDir {
		t.Fatalf("expected %s, got %s", toolDir, got)
	}
}

func writeTestPNG(path string, fill color.Color) error {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, fill)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func TestMaskSensitiveSmokeArgs(t *testing.T) {
	args := []string{
		"--admin-user", "admin",
		"--admin-password", "admin-secret",
		"--tenant-admin-password", "tenant-secret",
		"--tenant-user-password", "user-secret",
	}

	masked := maskSensitiveSmokeArgs(args)
	if masked[3] != "<hidden>" || masked[5] != "<hidden>" || masked[7] != "<hidden>" {
		t.Fatalf("expected passwords to be masked, got %v", masked)
	}
	if masked[1] != "admin" {
		t.Fatalf("expected non-sensitive values to remain unchanged, got %v", masked)
	}
}
