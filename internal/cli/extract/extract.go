package extract

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/gh"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	prNumber int
)

// ExtractCmd represents the extract-comments command
var ExtractCmd = &cobra.Command{
	Use:     "extract-comments",
	Aliases: []string{"ec"},
	Short:   "Extract CodeRabbit comments and AI prompts from a PR",
	Long: `Extracts instructions starting with 'In @' from CodeRabbit comments in a Pull Request.
It checks 'upstream' remote first, then 'origin'.`,
	RunE: runExtract,
}

func init() {
	ExtractCmd.Flags().IntVar(&prNumber, "pr", 0, "Pull Request number (required)")
	ExtractCmd.MarkFlagRequired("pr")
}

func runExtract(cmd *cobra.Command, args []string) error {
	// 1. Get Git Client
	s := util.NewSpinner("Initializing git client...")
	s.Start()
	gitClient, err := git.NewClient()
	s.Stop()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	// 2. Determine Repo
	s = util.NewSpinner("Determining repository...")
	s.Start()
	repo, err := getRepoName(gitClient)
	s.Stop()
	if err != nil {
		return err
	}
	fmt.Printf("Using repository: %s\n", repo)

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	// 3. Fetch Comments
	ghClient := gh.NewClient()
	
	// Fetch review threads (allows filtering resolved)
	s = util.NewSpinner(fmt.Sprintf("Fetching review threads for PR %d...", prNumber))
	s.Start()
	threads, err := ghClient.GetReviewThreads(owner, repoName, prNumber)
	s.Stop()
	if err != nil {
		fmt.Printf("Warning: failed to fetch review threads: %v\n", err)
		threads = []gh.ReviewThread{}
	}

	// Fetch issue comments (general PR comments)
	s = util.NewSpinner(fmt.Sprintf("Fetching issue comments for PR %d...", prNumber))
	s.Start()
	issueEndpoint := fmt.Sprintf("repos/%s/issues/%d/comments", repo, prNumber)
	issueComments, err := ghClient.GetComments(issueEndpoint)
	s.Stop()
	if err != nil {
		fmt.Printf("Warning: failed to fetch issue comments: %v\n", err)
		issueComments = []gh.Comment{}
	}

	// 4. Extract Instructions
	var allInstructions []string
	// Go regexp doesn't support lookahead (?=...), so we match the terminator in a group
	// Group 1: The instruction (starting with "In @")
	// Group 2: The terminator (\n\n, \r\n\r\n, or end of string)
	pattern := regexp.MustCompile(`(In @[\s\S]+?)(\n\n|\r\n\r\n|$)`)

	// Process threads
	for _, t := range threads {
		if t.IsResolved {
			continue
		}
		for _, c := range t.Comments.Nodes {
			matches := pattern.FindAllStringSubmatch(c.Body, -1)
			for _, m := range matches {
				if len(m) >= 2 {
					cleaned := strings.ReplaceAll(strings.TrimSpace(m[1]), "\n", " ")
					allInstructions = append(allInstructions, cleaned)
				}
			}
		}
	}

	// Process issue comments
	for _, c := range issueComments {
		matches := pattern.FindAllStringSubmatch(c.Body, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				cleaned := strings.ReplaceAll(strings.TrimSpace(m[1]), "\n", " ")
				allInstructions = append(allInstructions, cleaned)
			}
		}
	}

	// 5. Deduplicate
	seen := make(map[string]bool)
	var uniqueInstructions []string
	for _, instr := range allInstructions {
		if !seen[instr] {
			seen[instr] = true
		
uniqueInstructions = append(uniqueInstructions, instr)
		}
	}

	// 6. Save to file
	outputFile := "ai_instructions.txt"
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	for i, instr := range uniqueInstructions {
		_, err := fmt.Fprintf(f, "--- INSTRUCTION %d ---\n%s\n\n", i+1, instr)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	fmt.Printf("✅ Extracted %d AI instruction blocks to %s\n", len(uniqueInstructions), outputFile)
	return nil
}

func getRepoName(gitClient *git.Client) (string, error) {
	// Try upstream first
	remoteName := "upstream"
	url, err := gitClient.GetRemoteURL(remoteName)
	if err != nil {
		// Try origin
		remoteName = "origin"
		url, err = gitClient.GetRemoteURL(remoteName)
		if err != nil {
			return "", fmt.Errorf("could not find 'upstream' or 'origin' remote")
		}
	}

	// Parse URL to get owner/repo
	// Supported formats:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// https://github.com/owner/repo
	
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")
	
	// Handle git@github.com: format
	if strings.Contains(url, ":") && !strings.HasPrefix(url, "http") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	// Handle http format
	if strings.HasPrefix(url, "http") {
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1]), nil
		}
	}

	return "", fmt.Errorf("failed to parse repo name from remote URL: %s", url)
}
