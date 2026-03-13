package peertest

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	peerTestProductVersion string
	peerTestIssueURL       string
	peerTestUsername       string
	peerTestPassword       string
	peerTestRunMode        bool
)

type resolvedProductConfig struct {
	version       string
	packPath      string
	workspaceRoot string
	workingDir    string
	steps         []string
	runWorkingDir string
	runSteps      []string
}

type renderedStepSet struct {
	exec    []string
	display []string
}

var PeerTestCmd = &cobra.Command{
	Use:   "peertest",
	Short: "Prepare, update, or run a WSO2 peer test product pack",
	Long: `Prepare a version-specific peer test workspace from a configured product pack,
run the configured update workflow, or start an already prepared peer test pack.`,
	Example: `  aihelper peertest --product-version 4.4.0 --peertest-issue https://git.example.com/issues/15426 --username you@example.com --password '<secret>'
  aihelper peertest --run --product-version 4.4.0 --peertest-issue https://git.example.com/issues/15426`,
	RunE: runPeerTest,
}

func init() {
	PeerTestCmd.Flags().StringVar(&peerTestProductVersion, "product-version", "", "Product version to prepare or run (required)")
	PeerTestCmd.Flags().StringVar(&peerTestIssueURL, "peertest-issue", "", "Peer test issue URL used to derive the workspace folder (required)")
	PeerTestCmd.Flags().StringVar(&peerTestUsername, "username", "", "Username/email for the first updater login (required unless --run)")
	PeerTestCmd.Flags().StringVar(&peerTestPassword, "password", "", "Password for the first updater login (required unless --run)")
	PeerTestCmd.Flags().BoolVar(&peerTestRunMode, "run", false, "Run an already prepared peer test pack instead of preparing/updating one")
	_ = PeerTestCmd.MarkFlagRequired("product-version")
	_ = PeerTestCmd.MarkFlagRequired("peertest-issue")
}

func runPeerTest(cmd *cobra.Command, args []string) error {
	version := strings.TrimSpace(peerTestProductVersion)
	issueInput := strings.TrimSpace(peerTestIssueURL)
	username := strings.TrimSpace(peerTestUsername)
	password := peerTestPassword

	if version == "" {
		return fmt.Errorf("--product-version is required")
	}
	if issueInput == "" {
		return fmt.Errorf("--peertest-issue is required")
	}
	if !peerTestRunMode {
		if username == "" {
			return fmt.Errorf("--username is required unless --run is used")
		}
		if password == "" {
			return fmt.Errorf("--password is required unless --run is used")
		}
	}

	issueNumber, err := parsePeerTestIssueNumber(issueInput)
	if err != nil {
		return err
	}

	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return fmt.Errorf("failed to get config manager: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	productCfg, err := resolveProductConfig(cfg, version)
	if err != nil {
		return err
	}

	if peerTestRunMode {
		return runPreparedPeerTest(productCfg, issueNumber)
	}
	return preparePeerTest(productCfg, issueNumber, username, password)
}

func preparePeerTest(productCfg *resolvedProductConfig, issueNumber, username, password string) error {
	if productCfg.packPath == "" {
		return fmt.Errorf("peertest.products.%s.pack_path must be configured for prepare mode", productCfg.version)
	}
	if len(productCfg.steps) == 0 {
		return fmt.Errorf("peertest.products.%s.steps must contain at least one command for prepare mode", productCfg.version)
	}

	printProgress("Preparing peer test %s for product %s", issueNumber, productCfg.version)

	if _, err := os.Stat(productCfg.packPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("configured pack path does not exist for version %s: %s", productCfg.version, productCfg.packPath)
		}
		return fmt.Errorf("failed to inspect configured pack path %s: %w", productCfg.packPath, err)
	}

	issueDir := resolveIssueDir(productCfg, issueNumber)
	copiedZipPath := filepath.Join(issueDir, filepath.Base(productCfg.packPath))

	if _, err := os.Stat(issueDir); err == nil {
		return fmt.Errorf("peer test directory already exists: %s", issueDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect peer test directory %s: %w", issueDir, err)
	}

	renderedSteps := renderSteps(productCfg.steps, productCfg, username, password, issueNumber)

	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would prepare peer test workspace with these settings:")
		fmt.Printf("  Product version: %s\n", productCfg.version)
		fmt.Printf("  Issue number: %s\n", issueNumber)
		fmt.Printf("  Source pack: %s\n", productCfg.packPath)
		fmt.Printf("  Peer test root: %s\n", issueDir)
		fmt.Printf("  Copied zip: %s\n", copiedZipPath)
		fmt.Printf("  Working dir: %s\n", productCfg.workingDir)
		for i, step := range renderedSteps.display {
			fmt.Printf("  Step %d: %s\n", i+1, step)
		}
		return nil
	}

	printProgress("Creating peer test directory %s", issueDir)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		return fmt.Errorf("failed to create peer test directory %s: %w", issueDir, err)
	}
	printDone("Created peer test directory")

	printProgress("Copying product pack to %s", copiedZipPath)
	if err := copyFile(productCfg.packPath, copiedZipPath); err != nil {
		return fmt.Errorf("failed to copy pack zip to %s: %w", copiedZipPath, err)
	}
	printDone("Copied product pack")

	printProgress("Extracting %s", copiedZipPath)
	if err := unzipFile(copiedZipPath, issueDir); err != nil {
		return fmt.Errorf("failed to extract %s: %w", copiedZipPath, err)
	}
	printDone("Extracted product pack")

	productRoot, err := findExtractedProductDir(issueDir)
	if err != nil {
		return err
	}

	workingDir, err := validateWorkingDir(productRoot, productCfg.workingDir)
	if err != nil {
		return err
	}

	printPeerTestPlan(issueDir, productRoot, workingDir, renderedSteps.display)

	printProgress("Running configured peer test steps")
	if err := runPeerTestScript(workingDir, renderedSteps.exec, renderedSteps.display); err != nil {
		return err
	}
	printDone("Peer test steps completed")

	printProgress("Switching to %s", productRoot)
	return execShellInDir(productRoot)
}

func runPreparedPeerTest(productCfg *resolvedProductConfig, issueNumber string) error {
	if len(productCfg.runSteps) == 0 {
		return fmt.Errorf("peertest.products.%s.run_steps must contain at least one command for --run mode", productCfg.version)
	}

	issueDir := resolveIssueDir(productCfg, issueNumber)
	printProgress("Running prepared peer test %s for product %s", issueNumber, productCfg.version)

	if info, err := os.Stat(issueDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("peer test directory does not exist: %s", issueDir)
		}
		return fmt.Errorf("failed to inspect peer test directory %s: %w", issueDir, err)
	} else if !info.IsDir() {
		return fmt.Errorf("peer test directory is not a directory: %s", issueDir)
	}

	productRoot, err := findExtractedProductDir(issueDir)
	if err != nil {
		return err
	}

	workingDir, err := validateWorkingDir(productRoot, productCfg.runWorkingDir)
	if err != nil {
		return err
	}

	renderedSteps := renderSteps(productCfg.runSteps, productCfg, "", "", issueNumber)

	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would run prepared peer test workspace with these settings:")
		fmt.Printf("  Product version: %s\n", productCfg.version)
		fmt.Printf("  Issue number: %s\n", issueNumber)
		fmt.Printf("  Peer test root: %s\n", issueDir)
		fmt.Printf("  Product root: %s\n", productRoot)
		fmt.Printf("  Working dir: %s\n", workingDir)
		for i, step := range renderedSteps.display {
			fmt.Printf("  Step %d: %s\n", i+1, step)
		}
		return nil
	}

	printPeerTestPlan(issueDir, productRoot, workingDir, renderedSteps.display)
	printProgress("Starting configured peer test run steps")
	if err := runPeerTestScript(workingDir, renderedSteps.exec, renderedSteps.display); err != nil {
		return err
	}
	printDone("Peer test run steps completed")
	return nil
}

func resolveProductConfig(cfg *config.Config, version string) (*resolvedProductConfig, error) {
	entry, ok := cfg.PeerTest.Products[version]
	if !ok {
		return nil, fmt.Errorf("peertest.products.%s is not configured", version)
	}

	packPath := ""
	if strings.TrimSpace(entry.PackPath) != "" {
		var err error
		packPath, err = expandPath(entry.PackPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve peertest pack path for version %s: %w", version, err)
		}
	}

	workspaceRoot := strings.TrimSpace(entry.WorkspaceRoot)
	if workspaceRoot == "" && packPath != "" {
		workspaceRoot = filepath.Join(filepath.Dir(packPath), "peertests")
	}
	if workspaceRoot == "" {
		return nil, fmt.Errorf("peertest.products.%s.workspace_root must be configured or derivable from pack_path", version)
	}
	var err error
	workspaceRoot, err = expandPath(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve peertest workspace root for version %s: %w", version, err)
	}
	if workspaceRoot == "" {
		return nil, fmt.Errorf("peertest.products.%s.workspace_root must resolve to a non-empty path", version)
	}

	workingDir := normalizeWorkingDir(entry.WorkingDir)
	runWorkingDir := normalizeWorkingDir(entry.RunWorkingDir)

	return &resolvedProductConfig{
		version:       version,
		packPath:      packPath,
		workspaceRoot: workspaceRoot,
		workingDir:    workingDir,
		steps:         sanitizeSteps(entry.Steps),
		runWorkingDir: runWorkingDir,
		runSteps:      sanitizeSteps(entry.RunSteps),
	}, nil
}

func sanitizeSteps(input []string) []string {
	steps := make([]string, 0, len(input))
	for _, step := range input {
		trimmed := strings.TrimSpace(step)
		if trimmed != "" {
			steps = append(steps, trimmed)
		}
	}
	return steps
}

func normalizeWorkingDir(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "bin"
	}
	return filepath.Clean(trimmed)
}

func resolveIssueDir(productCfg *resolvedProductConfig, issueNumber string) string {
	return filepath.Join(productCfg.workspaceRoot, issueNumber)
}

func validateWorkingDir(productRoot, relativeWorkingDir string) (string, error) {
	workingDir := filepath.Join(productRoot, relativeWorkingDir)
	if info, err := os.Stat(workingDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("configured working directory does not exist in extracted pack: %s", workingDir)
		}
		return "", fmt.Errorf("failed to inspect working directory %s: %w", workingDir, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("configured working directory is not a directory: %s", workingDir)
	}
	return workingDir, nil
}

func parsePeerTestIssueNumber(issueURL string) (string, error) {
	parsed, err := url.Parse(issueURL)
	if err != nil {
		return "", fmt.Errorf("invalid --peertest-issue URL %q: %w", issueURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid --peertest-issue URL: expected a full URL")
	}

	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for i := len(segments) - 1; i >= 0; i-- {
		segment := strings.TrimSpace(segments[i])
		if segment == "" {
			continue
		}
		if match := trailingNumberPattern.FindStringSubmatch(segment); len(match) == 2 {
			return match[1], nil
		}
	}

	return "", fmt.Errorf("invalid --peertest-issue URL: could not derive an issue number from %q", issueURL)
}

var trailingNumberPattern = regexp.MustCompile(`(?i)(?:^|[^0-9])(\d+)$`)

func renderSteps(steps []string, cfg *resolvedProductConfig, username, password, issueNumber string) renderedStepSet {
	today := time.Now().Format("02-01-2006")

	valuesExec := map[string]string{
		"username":        shellQuote(username),
		"password":        shellQuote(password),
		"today":           shellQuote(today),
		"product_version": shellQuote(cfg.version),
		"issue_number":    shellQuote(issueNumber),
		"pack_path":       shellQuote(cfg.packPath),
		"workspace_root":  shellQuote(cfg.workspaceRoot),
	}
	valuesDisplay := map[string]string{
		"username":        username,
		"password":        "<hidden>",
		"today":           today,
		"product_version": cfg.version,
		"issue_number":    issueNumber,
		"pack_path":       cfg.packPath,
		"workspace_root":  cfg.workspaceRoot,
	}

	execSteps := make([]string, 0, len(steps))
	displaySteps := make([]string, 0, len(steps))
	for _, step := range steps {
		execSteps = append(execSteps, applyTemplate(step, valuesExec))
		displaySteps = append(displaySteps, applyTemplate(step, valuesDisplay))
	}

	return renderedStepSet{exec: execSteps, display: displaySteps}
}

func applyTemplate(input string, values map[string]string) string {
	rendered := input
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return rendered
}

func printPeerTestPlan(issueDir, productRoot, workingDir string, steps []string) {
	fmt.Println()
	printSeparator()
	fmt.Printf("Peer test root: %s\n", issueDir)
	fmt.Printf("Product root:   %s\n", productRoot)
	fmt.Printf("Working dir:    %s\n", workingDir)
	printSeparator()
	for i, step := range steps {
		fmt.Printf("Step %d/%d: %s\n", i+1, len(steps), step)
	}
	printSeparator()
	fmt.Println()
}

func printSeparator() {
	fmt.Println("============================================================")
}

func runPeerTestScript(workingDir string, execSteps, displaySteps []string) error {
	shell, shellArgs := shellCommand()
	var script strings.Builder
	script.WriteString("set -e\n")
	for i, step := range execSteps {
		script.WriteString("printf '%s\\n' ")
		script.WriteString(shellQuote("============================================================"))
		script.WriteString("\n")
		script.WriteString("printf '%s\\n' ")
		script.WriteString(shellQuote(fmt.Sprintf("Running step %d/%d: %s", i+1, len(execSteps), displaySteps[i])))
		script.WriteString("\n")
		script.WriteString("printf '%s\\n' ")
		script.WriteString(shellQuote("============================================================"))
		script.WriteString("\n")
		script.WriteString(step)
		script.WriteString("\n")
	}

	ctx, stop := signalContext()
	defer stop()

	cmd := exec.CommandContext(ctx, shell, append(shellArgs, script.String())...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("peer test command interrupted")
		}
		return fmt.Errorf("peer test commands failed: %w", err)
	}
	return nil
}

func signalContext() (context.Context, context.CancelFunc) {
	return signalNotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

var signalNotifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, signals...)
}

func shellCommand() (string, []string) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
		if _, err := os.Stat(shell); err != nil {
			shell = "/bin/sh"
		}
	}

	base := filepath.Base(shell)
	if strings.Contains(base, "fish") {
		return shell, []string{"-c"}
	}
	return shell, []string{"-lc"}
}

func findExtractedProductDir(root string) (string, error) {
	type candidate struct {
		path  string
		depth int
	}
	var matches []candidate
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		binDir := filepath.Join(path, "bin")
		if info, err := os.Stat(binDir); err == nil && info.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			depth := strings.Count(filepath.ToSlash(rel), "/")
			matches = append(matches, candidate{path: path, depth: depth})
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("failed while locating extracted product under %s: %w", root, err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("could not locate an extracted product directory under %s", root)
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].depth == matches[j].depth {
			return matches[i].path < matches[j].path
		}
		return matches[i].depth < matches[j].depth
	})

	return matches[0].path, nil
}

func unzipFile(srcZip, destDir string) error {
	reader, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath := filepath.Join(destDir, file.Name)
		if !isSubPath(destDir, targetPath) {
			return fmt.Errorf("zip entry escapes destination: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode().Perm()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode().Perm())
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rcErr := rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if rcErr != nil {
			return rcErr
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func isSubPath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func expandPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		if trimmed == "~" {
			return home, nil
		}
		trimmed = filepath.Join(home, strings.TrimPrefix(trimmed, "~/"))
	}

	return filepath.Clean(trimmed), nil
}

func execShellInDir(dir string) error {
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		for _, candidate := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
			if _, err := os.Stat(candidate); err == nil {
				shell = candidate
				break
			}
		}
	}
	if shell == "" {
		return fmt.Errorf("could not determine shell")
	}

	args := []string{filepath.Base(shell)}
	return syscall.Exec(shell, args, os.Environ())
}

func printProgress(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("→ %s\n", msg)
		return
	}
	fmt.Printf("-> %s\n", msg)
}

func printDone(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ %s\n", msg)
		return
	}
	fmt.Printf("✓ %s\n", msg)
}
