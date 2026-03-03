package wso2patch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	analyzeGitIssueURL string
	analyzeFolderPath  string
	analyzeUserPrompt  string
)

type githubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type analyzeScope struct {
	patchRoot string
	runRoot   string
	repos     []string
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze a WSO2 patch worktree with Codex",
	Long: `Analyze all repositories under the current WSO2 patch root.

Context can be provided from a GitHub issue URL, a local details folder, or both.
The command invokes codex in read-only mode and writes a timestamped markdown report.`,
	Example: `  aihelper wso2-patch analyze --git https://github.com/wso2/product-apim/issues/123
  aihelper wso2-patch analyze --folder ./notes
  aihelper wso2-patch analyze --git https://github.com/wso2/product-apim/issues/123 --folder /tmp/fix-notes
  aihelper wso2-patch analyze --git https://github.com/wso2/product-apim/issues/123 --prompt "Focus on gateway mediation flow"`,
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVar(&analyzeGitIssueURL, "git", "", "GitHub issue URL (https://github.com/<owner>/<repo>/issues/<number>)")
	analyzeCmd.Flags().StringVar(&analyzeFolderPath, "folder", "", "Folder path with patch/fix details")
	analyzeCmd.Flags().StringVar(&analyzeUserPrompt, "prompt", "", "Additional user prompt appended to Codex analysis instructions")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	issueURL := strings.TrimSpace(analyzeGitIssueURL)
	folderInput := strings.TrimSpace(analyzeFolderPath)
	userPrompt := strings.TrimSpace(analyzeUserPrompt)
	if issueURL == "" && folderInput == "" {
		return fmt.Errorf("at least one of --git or --folder is required")
	}

	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return fmt.Errorf("failed to get config manager: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	repoNames := resolveRepoNamesFromConfig(cfg)
	if len(repoNames) == 0 {
		return fmt.Errorf("wso2-patch.repos is empty; configure at least one repository")
	}

	patchRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	patchRoot, err = filepath.Abs(patchRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve current directory: %w", err)
	}

	printProgress("Resolving analysis scope from %s", patchRoot)
	scope, err := resolveAnalyzeScope(patchRoot, repoNames)
	if err != nil {
		return err
	}
	printDone("Analysis scope: %s", strings.Join(scope.repos, ", "))

	var issue *githubIssue
	if issueURL != "" {
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("gh cli not found. Please install GitHub CLI to use --git")
		}

		owner, repo, issueNumber, err := parseGitHubIssueURL(issueURL)
		if err != nil {
			return err
		}

		printProgress("Fetching GitHub issue %s/%s#%d", owner, repo, issueNumber)
		issue, err = fetchGitHubIssue(owner, repo, issueNumber)
		if err != nil {
			return err
		}
		printDone("Fetched GitHub issue")
	}

	folderPath := ""
	if folderInput != "" {
		folderPath, err = resolveFolderPath(patchRoot, folderInput)
		if err != nil {
			return err
		}
	}

	if _, err := exec.LookPath("codex"); err != nil {
		return fmt.Errorf("codex cli not found. Please install Codex CLI to use analyze")
	}

	prompt := buildAnalyzePrompt(scope.runRoot, scope.repos, issue, issueURL, folderPath, userPrompt)

	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would run codex analyze with these inputs:")
		fmt.Printf("  Patch root: %s\n", scope.patchRoot)
		fmt.Printf("  Run root: %s\n", scope.runRoot)
		if issue != nil {
			fmt.Printf("  Issue: %s (#%d)\n", issueURL, issue.Number)
		}
		if folderPath != "" {
			fmt.Printf("  Folder: %s\n", folderPath)
		}
		if userPrompt != "" {
			fmt.Printf("  User prompt: %s\n", userPrompt)
		}
		fmt.Printf("  Repos: %s\n", strings.Join(scope.repos, ", "))
		fmt.Printf("  Prompt length: %d chars\n", len(prompt))
		return nil
	}

	printProgress("Running Codex analysis")
	analysis, err := runCodexAnalyze(scope.runRoot, folderPath, prompt)
	if err != nil {
		return err
	}
	printDone("Codex analysis complete")

	reportPath := filepath.Join(scope.patchRoot, fmt.Sprintf("analysis-%s.md", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(reportPath, []byte(analysis), 0644); err != nil {
		return fmt.Errorf("failed to write report file %s: %w", reportPath, err)
	}

	fmt.Println(analysis)
	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Analysis report written to: %s\n", reportPath)
	} else {
		fmt.Printf("Analysis report written to: %s\n", reportPath)
	}

	return nil
}

func parseGitHubIssueURL(issueURL string) (owner, repo string, issueNumber int, err error) {
	parsed, err := urlParse(issueURL)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid --git URL %q: %w", issueURL, err)
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return "", "", 0, fmt.Errorf("invalid --git URL: expected github.com issue URL like https://github.com/<owner>/<repo>/issues/<number>")
	}

	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(segments) != 4 || segments[2] != "issues" {
		return "", "", 0, fmt.Errorf("invalid --git URL: expected github issue URL like https://github.com/<owner>/<repo>/issues/<number>")
	}

	issueNumber, err = strconv.Atoi(segments[3])
	if err != nil || issueNumber <= 0 {
		return "", "", 0, fmt.Errorf("invalid --git URL: issue number must be a positive integer")
	}

	owner = segments[0]
	repo = segments[1]
	if owner == "" || repo == "" {
		return "", "", 0, fmt.Errorf("invalid --git URL: owner/repo cannot be empty")
	}

	return owner, repo, issueNumber, nil
}

func resolveRepoNamesFromConfig(cfg *config.Config) []string {
	if cfg == nil || len(cfg.WSO2Patch.Repos) == 0 {
		return nil
	}

	names := make([]string, 0, len(cfg.WSO2Patch.Repos))
	seen := make(map[string]struct{}, len(cfg.WSO2Patch.Repos))
	for _, repoCfg := range cfg.WSO2Patch.Repos {
		repoName := strings.TrimSpace(repoCfg.Name)
		if repoName == "" {
			repoName = filepath.Base(strings.TrimSpace(repoCfg.Path))
		}
		if repoName == "" {
			continue
		}
		if _, ok := seen[repoName]; ok {
			continue
		}
		seen[repoName] = struct{}{}
		names = append(names, repoName)
	}

	return names
}

func validatePatchRoot(cwd string, repoNames []string) error {
	if strings.TrimSpace(cwd) == "" {
		return fmt.Errorf("invalid patch root: empty path")
	}
	if len(repoNames) == 0 {
		return fmt.Errorf("invalid patch root validation: no repo names configured")
	}

	info, err := os.Stat(cwd)
	if err != nil {
		return fmt.Errorf("failed to inspect patch root %s: %w", cwd, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("patch root is not a directory: %s", cwd)
	}

	missing := make([]string, 0)
	for _, repoName := range repoNames {
		repoPath := filepath.Join(cwd, repoName)
		repoInfo, err := os.Stat(repoPath)
		if err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, repoName)
				continue
			}
			return fmt.Errorf("failed to inspect repository folder %s: %w", repoPath, err)
		}
		if !repoInfo.IsDir() {
			return fmt.Errorf("repository folder is not a directory: %s", repoPath)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("current directory is not a valid patch root; missing repo folders: %s", strings.Join(missing, ", "))
	}

	return nil
}

func resolveAnalyzeScope(cwd string, repoNames []string) (*analyzeScope, error) {
	if err := validatePatchRoot(cwd, repoNames); err == nil {
		return &analyzeScope{
			patchRoot: cwd,
			runRoot:   cwd,
			repos:     append([]string(nil), repoNames...),
		}, nil
	}

	repoName := filepath.Base(cwd)
	if !containsString(repoNames, repoName) {
		return nil, fmt.Errorf("run analyze from a patch root or a repo subfolder (%s)", strings.Join(repoNames, ", "))
	}

	parent := filepath.Dir(cwd)
	if err := validatePatchRoot(parent, repoNames); err != nil {
		return nil, fmt.Errorf("invalid patch root parent for repo %q: %w", repoName, err)
	}

	return &analyzeScope{
		patchRoot: parent,
		runRoot:   cwd,
		repos:     []string{repoName},
	}, nil
}

func isSubPath(parent, child string) bool {
	cleanParent := filepath.Clean(parent)
	cleanChild := filepath.Clean(child)
	rel, err := filepath.Rel(cleanParent, cleanChild)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..") && rel != ""
}

func runCodexAnalyze(patchRoot, folderPath, prompt string) (string, error) {
	tmpFile, err := os.CreateTemp("", "aihelper-wso2-analysis-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary output file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	args := []string{"exec", "--sandbox", "read-only", "--skip-git-repo-check", "--cd", patchRoot, "--output-last-message", tmpPath}
	if folderPath != "" && !isSubPath(patchRoot, folderPath) {
		args = append(args, "--add-dir", folderPath)
	}
	args = append(args, prompt)

	cmd := exec.Command("codex", args...)
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				printProgress("Codex analysis is still running...")
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		close(done)
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("codex analyze failed: %s", errMsg)
	}
	close(done)

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read codex output: %w", err)
	}
	analysis := strings.TrimSpace(string(data))
	if analysis == "" {
		return "", fmt.Errorf("codex analyze returned empty output")
	}

	return analysis, nil
}

func fetchGitHubIssue(owner, repo string, issueNumber int) (*githubIssue, error) {
	fullRepo := fmt.Sprintf("%s/%s", owner, repo)
	args := []string{"issue", "view", strconv.Itoa(issueNumber), "--repo", fullRepo, "--json", "number,title,body"}
	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("failed to fetch GitHub issue %s#%d: %s", fullRepo, issueNumber, errMsg)
	}

	var issue githubIssue
	if err := json.Unmarshal(stdout.Bytes(), &issue); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub issue response: %w", err)
	}

	return &issue, nil
}

func resolveFolderPath(baseDir, input string) (string, error) {
	expanded, err := expandPath(input)
	if err != nil {
		return "", fmt.Errorf("failed to resolve --folder path %q: %w", input, err)
	}
	if expanded == "" {
		return "", fmt.Errorf("--folder path cannot be empty")
	}

	folderPath := expanded
	if !filepath.IsAbs(folderPath) {
		folderPath = filepath.Join(baseDir, folderPath)
	}
	folderPath, err = filepath.Abs(folderPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute folder path %q: %w", folderPath, err)
	}

	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("--folder path does not exist: %s", folderPath)
		}
		return "", fmt.Errorf("failed to inspect --folder path %s: %w", folderPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--folder path is not a directory: %s", folderPath)
	}

	return folderPath, nil
}

func buildAnalyzePrompt(runRoot string, repoNames []string, issue *githubIssue, issueURL, folderPath, userPrompt string) string {
	var b strings.Builder

	b.WriteString("You are analyzing a WSO2 patch workspace.\\n")
	b.WriteString("Inspect the repositories in this workspace and produce a technical analysis.\\n\\n")
	b.WriteString("Analysis run root: ")
	b.WriteString(runRoot)
	b.WriteString("\\n\\n")

	b.WriteString("Repositories in scope:\\n")
	for _, repoName := range repoNames {
		b.WriteString("- ")
		b.WriteString(repoName)
		b.WriteString("\\n")
	}

	if issue != nil {
		b.WriteString("\\nContext: GitHub Issue\\n")
		b.WriteString("URL: ")
		b.WriteString(issueURL)
		b.WriteString("\\n")
		b.WriteString("Title: ")
		b.WriteString(issue.Title)
		b.WriteString("\\n")
		b.WriteString("Body:\\n")
		b.WriteString(issue.Body)
		b.WriteString("\\n")
	}

	if folderPath != "" {
		b.WriteString("\\nContext: Details Folder\\n")
		b.WriteString("Folder: ")
		b.WriteString(folderPath)
		b.WriteString("\\n")
		b.WriteString("Read relevant files from this folder and incorporate them into your analysis.\\n")
	}

	b.WriteString("\\nRequired coverage in your response:\\n")
	b.WriteString("- What is the problem being addressed\\n")
	b.WriteString("- Which components are likely affected\\n")
	b.WriteString("- Potential fixes or implementation approaches\\n")
	b.WriteString("- Steps to reproduce or validate\\n")
	b.WriteString("- Runtime/setup guidance needed to work on and test this patch\\n")

	if userPrompt != "" {
		b.WriteString("\\nAdditional user instructions:\\n")
		b.WriteString(userPrompt)
		b.WriteString("\\n")
	}

	b.WriteString("\\nKeep the answer actionable and grounded in repository evidence.\\n")

	return b.String()
}

func urlParse(rawURL string) (*url.URL, error) {
	return url.ParseRequestURI(rawURL)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
