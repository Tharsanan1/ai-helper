package peertest

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	peerTestProductVersion       string
	peerTestIssueURL             string
	peerTestUsername             string
	peerTestPassword             string
	peerTestRunMode              bool
	peerTestSmokeMode            bool
	peerTestSmokeHeadless        bool
	peerTestSmokeKeepOpen        bool
	peerTestSmokeBaseURL         string
	peerTestSmokeAdminUser       string
	peerTestSmokeAdminPass       string
	peerTestSmokeTenant          string
	peerTestSmokeTenantAdm       string
	peerTestSmokeTenantPwd       string
	peerTestSmokeTenantMail      string
	peerTestSmokeTenantFirst     string
	peerTestSmokeTenantLast      string
	peerTestSmokeTenantUser      string
	peerTestSmokeUserPwd         string
	peerTestSmokeAPIEndpoint     string
	peerTestSmokeAPIName         string
	peerTestSmokeAPIVersion      string
	peerTestSmokeScreenshotDir   string
	peerTestSmokeScreenshotDelay int
	peerTestSmokeSlowMo          int
)

type resolvedProductConfig struct {
	version       string
	packPath      string
	workspaceRoot string
	workingDir    string
	steps         []string
	runWorkingDir string
	runSteps      []string
	smokeTest     config.PeerTestSmokeTestConfig
}

type renderedStepSet struct {
	exec    []string
	display []string
}

var PeerTestCmd = &cobra.Command{
	Use:   "peertest",
	Short: "Prepare, update, or run a WSO2 peer test product pack",
	Long: `Prepare a version-specific peer test workspace from a configured product pack,
run the configured update workflow, start an already prepared peer test pack,
or execute the version-specific browser smoke test.`,
	Example: `  aihelper peertest --product-version 4.4.0 --peertest-issue https://git.example.com/issues/15426 --username you@example.com --password '<secret>'
  aihelper peertest --run --product-version 4.4.0 --peertest-issue https://git.example.com/issues/15426
  aihelper peertest --smoketest --product-version 4.4.0 --headless --screenshot-dir /tmp/peertest-shots`,
	RunE: runPeerTest,
}

func init() {
	PeerTestCmd.Flags().StringVar(&peerTestProductVersion, "product-version", "", "Product version to prepare or run (required)")
	PeerTestCmd.Flags().StringVar(&peerTestIssueURL, "peertest-issue", "", "Peer test issue URL used to derive the workspace folder (required for prepare/run)")
	PeerTestCmd.Flags().StringVar(&peerTestUsername, "username", "", "Username/email for the first updater login (required unless --run)")
	PeerTestCmd.Flags().StringVar(&peerTestPassword, "password", "", "Password for the first updater login (required unless --run)")
	PeerTestCmd.Flags().BoolVar(&peerTestRunMode, "run", false, "Run an already prepared peer test pack instead of preparing/updating one")
	PeerTestCmd.Flags().BoolVar(&peerTestSmokeMode, "smoketest", false, "Run the product-version specific Playwright smoke test")
	PeerTestCmd.Flags().BoolVar(&peerTestSmokeHeadless, "headless", false, "Run the smoke test browser in headless mode")
	PeerTestCmd.Flags().BoolVar(&peerTestSmokeKeepOpen, "keep-open", false, "Keep the smoke test browser open after completion")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeBaseURL, "base-url", "", "Base URL for the smoke test target product (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeAdminUser, "admin-user", "", "Super tenant admin username for the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeAdminPass, "admin-password", "", "Super tenant admin password for the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenant, "tenant-domain", "", "Tenant domain used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenantAdm, "tenant-admin-user", "", "Tenant admin username used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenantPwd, "tenant-admin-password", "", "Tenant admin password used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenantMail, "tenant-admin-email", "", "Tenant admin email used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenantFirst, "tenant-admin-first-name", "", "Tenant admin first name used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenantLast, "tenant-admin-last-name", "", "Tenant admin last name used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeTenantUser, "tenant-user", "", "Tenant application user created by the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeUserPwd, "tenant-user-password", "", "Tenant application user password used during the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeAPIEndpoint, "api-endpoint", "", "Endpoint used when creating the smoke test API (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeAPIName, "api-name-prefix", "", "API name prefix used by the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeAPIVersion, "api-version", "", "API version used by the smoke test (defaults to config)")
	PeerTestCmd.Flags().StringVar(&peerTestSmokeScreenshotDir, "screenshot-dir", "", "Directory to store smoke test screenshots (defaults to config)")
	PeerTestCmd.Flags().IntVar(&peerTestSmokeScreenshotDelay, "screenshot-delay-ms", 0, "Delay before each smoke test screenshot in milliseconds (defaults to config)")
	PeerTestCmd.Flags().IntVar(&peerTestSmokeSlowMo, "slow-mo", 0, "Playwright slow motion delay in milliseconds (defaults to config)")
	_ = PeerTestCmd.MarkFlagRequired("product-version")
}

func runPeerTest(cmd *cobra.Command, args []string) error {
	version := strings.TrimSpace(peerTestProductVersion)
	issueInput := strings.TrimSpace(peerTestIssueURL)
	username := strings.TrimSpace(peerTestUsername)
	password := peerTestPassword

	if version == "" {
		return fmt.Errorf("--product-version is required")
	}
	if peerTestRunMode && peerTestSmokeMode {
		return fmt.Errorf("--run and --smoketest cannot be used together")
	}
	if !peerTestSmokeMode && issueInput == "" {
		return fmt.Errorf("--peertest-issue is required unless --smoketest is used")
	}
	if !peerTestRunMode && !peerTestSmokeMode {
		if username == "" {
			return fmt.Errorf("--username is required unless --run is used")
		}
		if password == "" {
			return fmt.Errorf("--password is required unless --run or --smoketest is used")
		}
	}

	issueNumber := ""
	if issueInput != "" {
		var err error
		issueNumber, err = parsePeerTestIssueNumber(issueInput)
		if err != nil {
			return err
		}
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

	if peerTestSmokeMode {
		return runPeerTestSmoke(cmd, productCfg)
	}
	if peerTestRunMode {
		return runPreparedPeerTest(productCfg, issueNumber)
	}
	return preparePeerTest(productCfg, issueNumber, username, password)
}

func runPeerTestSmoke(cmd *cobra.Command, productCfg *resolvedProductConfig) error {
	toolDir, err := resolveSmokeTestToolDir(productCfg.version)
	if err != nil {
		return err
	}

	packageJSON := filepath.Join(toolDir, "package.json")
	scriptPath := filepath.Join(toolDir, "smoke-test.mjs")
	if _, err := os.Stat(packageJSON); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("smoke test tool is missing package.json: %s", packageJSON)
		}
		return fmt.Errorf("failed to inspect smoke test package.json %s: %w", packageJSON, err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("smoke test script is missing: %s", scriptPath)
		}
		return fmt.Errorf("failed to inspect smoke test script %s: %w", scriptPath, err)
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm is required for --smoketest but was not found in PATH")
	}

	smokeCfg := productCfg.smokeTest
	baseURL := smokeFlagString(cmd, "base-url", peerTestSmokeBaseURL, smokeCfg.BaseURL)
	adminUser := smokeFlagString(cmd, "admin-user", peerTestSmokeAdminUser, smokeCfg.AdminUser)
	adminPassword := smokeFlagString(cmd, "admin-password", peerTestSmokeAdminPass, smokeCfg.AdminPassword)
	tenantDomain := smokeFlagString(cmd, "tenant-domain", peerTestSmokeTenant, smokeCfg.TenantDomain)
	tenantAdminUser := smokeFlagString(cmd, "tenant-admin-user", peerTestSmokeTenantAdm, smokeCfg.TenantAdminUser)
	tenantAdminPassword := smokeFlagString(cmd, "tenant-admin-password", peerTestSmokeTenantPwd, smokeCfg.TenantAdminPassword)
	tenantAdminEmail := smokeFlagString(cmd, "tenant-admin-email", peerTestSmokeTenantMail, smokeCfg.TenantAdminEmail)
	tenantAdminFirstName := smokeFlagString(cmd, "tenant-admin-first-name", peerTestSmokeTenantFirst, smokeCfg.TenantAdminFirstName)
	tenantAdminLastName := smokeFlagString(cmd, "tenant-admin-last-name", peerTestSmokeTenantLast, smokeCfg.TenantAdminLastName)
	tenantUser := smokeFlagString(cmd, "tenant-user", peerTestSmokeTenantUser, smokeCfg.TenantUser)
	tenantUserPassword := smokeFlagString(cmd, "tenant-user-password", peerTestSmokeUserPwd, smokeCfg.TenantUserPassword)
	apiEndpoint := smokeFlagString(cmd, "api-endpoint", peerTestSmokeAPIEndpoint, smokeCfg.APIEndpoint)
	apiNamePrefix := smokeFlagString(cmd, "api-name-prefix", peerTestSmokeAPIName, smokeCfg.APINamePrefix)
	apiVersion := smokeFlagString(cmd, "api-version", peerTestSmokeAPIVersion, smokeCfg.APIVersion)
	screenshotDir := smokeFlagString(cmd, "screenshot-dir", peerTestSmokeScreenshotDir, smokeCfg.ScreenshotDir)
	screenshotDelay := smokeFlagInt(cmd, "screenshot-delay-ms", peerTestSmokeScreenshotDelay, smokeCfg.ScreenshotDelayMs)
	slowMo := smokeFlagInt(cmd, "slow-mo", peerTestSmokeSlowMo, smokeCfg.SlowMo)

	if tenantAdminEmail == "" && tenantAdminUser != "" && tenantDomain != "" {
		tenantAdminEmail = fmt.Sprintf("%s@%s", tenantAdminUser, tenantDomain)
	}

	smokeArgs := []string{
		"--base-url", baseURL,
		"--admin-user", adminUser,
		"--admin-password", adminPassword,
		"--tenant-domain", tenantDomain,
		"--tenant-admin-user", tenantAdminUser,
		"--tenant-admin-password", tenantAdminPassword,
		"--tenant-admin-email", tenantAdminEmail,
		"--tenant-admin-first-name", tenantAdminFirstName,
		"--tenant-admin-last-name", tenantAdminLastName,
		"--tenant-user", tenantUser,
		"--tenant-user-password", tenantUserPassword,
		"--api-endpoint", apiEndpoint,
		"--api-name-prefix", apiNamePrefix,
		"--api-version", apiVersion,
		"--screenshot-delay-ms", strconv.Itoa(screenshotDelay),
		"--slow-mo", strconv.Itoa(slowMo),
	}
	if peerTestSmokeHeadless {
		smokeArgs = append(smokeArgs, "--headless")
	}
	if peerTestSmokeKeepOpen {
		smokeArgs = append(smokeArgs, "--keep-open")
	}
	if trimmed := strings.TrimSpace(screenshotDir); trimmed != "" {
		smokeArgs = append(smokeArgs, "--screenshot-dir", trimmed)
	}

	printProgress("Running peer test smoke test for product %s", productCfg.version)
	fmt.Println()
	printSeparator()
	fmt.Printf("Smoke test tool: %s\n", toolDir)
	fmt.Printf("Smoke test script: %s\n", scriptPath)
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Printf("Tenant domain: %s\n", tenantDomain)
	fmt.Printf("Tenant user: %s\n", tenantUser)
	fmt.Printf("Headless: %t\n", peerTestSmokeHeadless)
	fmt.Printf("Screenshot dir: %s\n", strings.TrimSpace(screenshotDir))
	printSeparator()
	fmt.Println()

	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would execute these commands:")
		fmt.Printf("  cd %s && npm install\n", toolDir)
		fmt.Printf("  cd %s && node smoke-test.mjs %s\n", toolDir, strings.Join(maskSensitiveSmokeArgs(smokeArgs), " "))
		return nil
	}

	if _, err := os.Stat(filepath.Join(toolDir, "node_modules")); os.IsNotExist(err) {
		printProgress("Installing smoke test dependencies in %s", toolDir)
		if err := runStreamingCommand(toolDir, nil, "npm", "install"); err != nil {
			return fmt.Errorf("failed to install smoke test dependencies: %w", err)
		}
		printDone("Installed smoke test dependencies")
	}

	printProgress("Launching smoke test")
	if err := runStreamingCommand(toolDir, nil, "node", append([]string{"smoke-test.mjs"}, smokeArgs...)...); err != nil {
		return fmt.Errorf("smoke test failed: %w", err)
	}
	printDone("Smoke test completed")
	return nil
}

func smokeFlagString(cmd *cobra.Command, name, flagValue, configValue string) string {
	if cmd.Flags().Changed(name) {
		return strings.TrimSpace(flagValue)
	}
	return strings.TrimSpace(configValue)
}

func smokeFlagInt(cmd *cobra.Command, name string, flagValue, configValue int) int {
	if cmd.Flags().Changed(name) {
		return flagValue
	}
	return configValue
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
		smokeTest:     withDefaultSmokeTestConfig(entry.SmokeTest),
	}, nil
}

func withDefaultSmokeTestConfig(input config.PeerTestSmokeTestConfig) config.PeerTestSmokeTestConfig {
	if strings.TrimSpace(input.BaseURL) == "" {
		input.BaseURL = "https://localhost:9443"
	}
	if strings.TrimSpace(input.AdminUser) == "" {
		input.AdminUser = "admin"
	}
	if strings.TrimSpace(input.AdminPassword) == "" {
		input.AdminPassword = "admin"
	}
	if strings.TrimSpace(input.TenantDomain) == "" {
		input.TenantDomain = "peertest.com"
	}
	if strings.TrimSpace(input.TenantAdminUser) == "" {
		input.TenantAdminUser = "peer"
	}
	if strings.TrimSpace(input.TenantAdminPassword) == "" {
		input.TenantAdminPassword = "peer1"
	}
	if strings.TrimSpace(input.TenantAdminFirstName) == "" {
		input.TenantAdminFirstName = "peer"
	}
	if strings.TrimSpace(input.TenantAdminLastName) == "" {
		input.TenantAdminLastName = "admin"
	}
	if strings.TrimSpace(input.TenantUser) == "" {
		input.TenantUser = "peertestuser"
	}
	if strings.TrimSpace(input.TenantUserPassword) == "" {
		input.TenantUserPassword = "peer1"
	}
	if strings.TrimSpace(input.APIEndpoint) == "" {
		input.APIEndpoint = "https://httpbin.org/anything"
	}
	if strings.TrimSpace(input.APINamePrefix) == "" {
		input.APINamePrefix = "PeerTestAPI"
	}
	if strings.TrimSpace(input.APIVersion) == "" {
		input.APIVersion = "1.0.0"
	}
	if input.ScreenshotDelayMs < 0 {
		input.ScreenshotDelayMs = 0
	}
	if input.ScreenshotDelayMs == 0 {
		input.ScreenshotDelayMs = 1000
	}
	if input.SlowMo < 0 {
		input.SlowMo = 0
	}
	if input.SlowMo == 0 {
		input.SlowMo = 250
	}
	if strings.TrimSpace(input.TenantAdminEmail) == "" {
		input.TenantAdminEmail = fmt.Sprintf("%s@%s", input.TenantAdminUser, input.TenantDomain)
	}
	return input
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

func runStreamingCommand(workingDir string, envOverrides map[string]string, name string, args ...string) error {
	ctx, stop := signalContext()
	defer stop()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workingDir
	if envOverrides != nil {
		cmd.Env = mergedEnv(envOverrides)
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("command interrupted")
		}
		return err
	}
	return nil
}

func runPeerTestScript(workingDir string, execSteps, displaySteps []string) error {
	ctx, stop := signalContext()
	defer stop()

	envOverrides := map[string]string{}
	for i, step := range execSteps {
		fmt.Println("============================================================")
		fmt.Printf("Running step %d/%d: %s\n", i+1, len(execSteps), displaySteps[i])
		fmt.Println("============================================================")

		if key, value, ok := parseExportStep(step); ok {
			envOverrides[key] = value
			continue
		}

		if err := runPeerTestCommand(ctx, workingDir, envOverrides, step, displaySteps[i]); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("peer test command interrupted")
			}
			return err
		}
	}
	return nil
}

func runPeerTestCommand(ctx context.Context, workingDir string, envOverrides map[string]string, execStep, displayStep string) error {
	shell, shellArgs := shellCommand()
	attempts := 1
	if shouldAllowStepRerun(execStep) {
		attempts = 2
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		cmd := exec.CommandContext(ctx, shell, append(shellArgs, execStep)...)
		cmd.Dir = workingDir
		cmd.Env = mergedEnv(envOverrides)
		cmd.Stdin = os.Stdin

		var stdoutBuf bytes.Buffer
		var stderrBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

		err := cmd.Run()
		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return fmt.Errorf("peer test command interrupted")
		}

		combinedOutput := stdoutBuf.String() + "\n" + stderrBuf.String()
		if attempt < attempts && shouldRerunAfterUpdateToolSelfUpdate(combinedOutput) {
			printProgress("WSO2 update tool self-updated during %q; rerunning step once", displayStep)
			continue
		}

		return fmt.Errorf("peer test command failed for %q: %w", displayStep, err)
	}

	return nil
}

func shouldAllowStepRerun(step string) bool {
	return strings.Contains(step, "wso2update_")
}

func shouldRerunAfterUpdateToolSelfUpdate(output string) bool {
	return strings.Contains(output, "Update tool client has been updated. Please re-run the tool with necessary parameters")
}

func parseExportStep(step string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(step)
	if !strings.HasPrefix(trimmed, "export ") {
		return "", "", false
	}

	assignment := strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	parts := strings.SplitN(assignment, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	key = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}

	value = strings.Trim(value, `"'`)
	return key, value, true
}

func mergedEnv(overrides map[string]string) []string {
	envMap := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		envMap[parts[0]] = parts[1]
	}
	for key, value := range overrides {
		envMap[key] = value
	}

	merged := make([]string, 0, len(envMap))
	for key, value := range envMap {
		merged = append(merged, key+"="+value)
	}
	return merged
}

func signalContext() (context.Context, context.CancelFunc) {
	return signalNotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

var signalNotifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, signals...)
}

var runtimeCaller = runtime.Caller

func resolveSmokeTestToolDir(version string) (string, error) {
	_, sourceFile, _, ok := runtimeCaller(0)
	if !ok {
		return "", fmt.Errorf("failed to resolve peertest smoke test tool directory")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", ".."))
	toolDir := filepath.Join(repoRoot, "tools", "peertest-smoketest-"+version)
	if info, err := os.Stat(toolDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no smoke test tool is available for product version %s at %s", version, toolDir)
		}
		return "", fmt.Errorf("failed to inspect smoke test tool directory %s: %w", toolDir, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("smoke test tool path is not a directory: %s", toolDir)
	}

	return toolDir, nil
}

func maskSensitiveSmokeArgs(args []string) []string {
	masked := make([]string, len(args))
	copy(masked, args)
	for i := 0; i < len(masked)-1; i++ {
		switch masked[i] {
		case "--admin-password", "--tenant-admin-password", "--tenant-user-password":
			masked[i+1] = "<hidden>"
		}
	}
	return masked
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
