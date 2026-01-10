package audit

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/launcher"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var AuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Perform a local pre-review of your changes",
	Long: `Gets the diff of the current worktree vs main (or upstream/origin)
and sends it to Gemini/Claude with a "Senior Engineer" system prompt
to provide a structured critique.`, 
	RunE: runAudit,
}

func runAudit(cmd *cobra.Command, args []string) error {
	s := util.NewSpinner("Preparing audit...")
	s.Start()
	defer s.Stop()

	// Initialize git client
	client, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	// Determine base branch
	// We want to compare against the base of the feature branch.
	// Usually this is origin/main.
	baseBranch := "origin/main"
	
	// Check if origin/main exists
	if exists, _ := client.RefExists("origin/main"); !exists {
		if exists, _ := client.RefExists("upstream/main"); exists {
			baseBranch = "upstream/main"
		} else if exists, _ := client.BranchExists("main"); exists {
			baseBranch = "main"
		} else if exists, _ := client.BranchExists("master"); exists {
			baseBranch = "master"
		} else if exists, _ := client.RefExists("origin/master"); exists {
			baseBranch = "origin/master"
		}
	}

	// Get current HEAD
	head := "HEAD"

	s.Update(fmt.Sprintf("Getting diff between %s and %s...", baseBranch, head))

	diff, err := client.GetDiff(baseBranch, head)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		s.Stop()
		fmt.Println("No changes detected to audit (compared to " + baseBranch + ").")
		return nil
	}

	// Construct system prompt
	systemPrompt := `You are a Senior Software Engineer. Review the following git diff.
Focus on:
1. Security vulnerabilities.
2. Performance bottlenecks.
3. Code style and idiomatic usage.
4. Potential bugs or edge cases.

Provide a structured critique with actionable feedback.
If the code looks good, briefly mention that.`

	fullPrompt := fmt.Sprintf("%s\n\nDiff:\n%s", systemPrompt, diff)

	// Check for AI tools
	// Try Gemini first
	gemini := launcher.NewGeminiLauncher("")
	if gemini.IsAvailable() {
		s.Update("Analyzing with Gemini...")
		s.Stop()

		geminiPath := gemini.GetCLIPath()
		
		// Use stdin to pass prompt to avoid argument length limits
		cmd := exec.Command(geminiPath)
		cmd.Stdin = strings.NewReader(fullPrompt)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		fmt.Println("Sending diff to Gemini for review...")
		return cmd.Run()
	}

	// Try Claude
	claude := launcher.NewClaudeLauncher("")
	if claude.IsAvailable() {
		s.Update("Analyzing with Claude...")
		s.Stop()

		claudePath := claude.GetCLIPath()
		
		// Use -p for print mode and pass prompt via stdin
		cmd := exec.Command(claudePath, "-p")
		cmd.Stdin = strings.NewReader(fullPrompt)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		fmt.Println("Sending diff to Claude for review...")
		return cmd.Run()
	}

	return fmt.Errorf("no AI tool (Gemini or Claude) found to perform audit. Please install one of them")
}
