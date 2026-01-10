package pr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var PrCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create a pull request",
	Long: `Create a pull request for the current worktree.

Checks if there are unpushed commits and pushes them to origin.
If an upstream remote is configured, it creates the PR against the upstream main branch.
Otherwise, it uses the default behavior of 'gh pr create'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Initialize git client
		client, err := git.NewClient()
		if err != nil {
			return fmt.Errorf("failed to initialize git client: %w", err)
		}

		// 2. Get current branch
		branch, err := client.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// 3. Check for unpushed commits
		if util.GlobalContext.Verbose {
			fmt.Printf("Checking for unpushed commits on branch %s...\n", branch)
		}

		s := util.NewSpinner("Checking for unpushed commits...")
		s.Start()
		hasUnpushed, err := client.HasUnpushedCommits(branch)
		s.Stop()
		if err != nil {
			return fmt.Errorf("failed to check unpushed commits: %w", err)
		}

		if hasUnpushed {
			fmt.Printf("Pushing branch %s to origin...\n", branch)
			if !util.GlobalContext.DryRun {
				s = util.NewSpinner("Pushing to origin...")
				s.Start()
				err := client.Push(branch)
				s.Stop()
				if err != nil {
					return fmt.Errorf("failed to push branch: %w", err)
				}
				fmt.Println("Successfully pushed to origin.")
			} else {
				fmt.Println("[Dry Run] Would push to origin")
			}
		} else {
			if util.GlobalContext.Verbose {
				fmt.Println("No unpushed commits found.")
			}
		}

		// 4. Check for upstream remote
		ghArgs := []string{"pr", "create"}

		upstreamURL, err := client.GetRemoteURL("upstream")
		if err == nil && upstreamURL != "" {
			if util.GlobalContext.Verbose {
				fmt.Printf("Upstream remote detected (%s). Targeting upstream main.\n", upstreamURL)
			}
			ghArgs = append(ghArgs, "--repo", upstreamURL, "--base", "main")
		} else if util.GlobalContext.Verbose {
			fmt.Println("No upstream remote found. Using default gh behavior.")
		}

		// 5. Generate PR content using Gemini if not provided
		hasTitle := false
		hasBody := false
		for _, arg := range args {
			if arg == "--title" || arg == "-t" {
				hasTitle = true
			}
			if arg == "--body" || arg == "-b" {
				hasBody = true
			}
		}

		if !hasTitle && !hasBody {
			if _, err := exec.LookPath("gemini"); err == nil {
				if util.GlobalContext.Verbose {
					fmt.Println("Gemini CLI found. Generating PR content from commits...")
				}

				// Determine base ref for diff
				// Priority: upstream/main -> origin/main -> main
				baseRef := "main"
				if exists, _ := client.RefExists("upstream/main"); exists {
					baseRef = "upstream/main"
				} else if exists, _ := client.RefExists("origin/main"); exists {
					baseRef = "origin/main"
				}

				if util.GlobalContext.Verbose {
					fmt.Printf("Using base ref: %s\n", baseRef)
				}
				
				logs, err := client.GetCommitLogs(baseRef, "HEAD")
				if err == nil && logs != "" {
					prompt := fmt.Sprintf("Generate a concise Pull Request Title and a detailed Markdown Description based on these commits. Format the output exactly as:\nTITLE: <title>\nBODY:\n<description>\n\nCommits:\n%s", logs)
					
					if util.GlobalContext.Verbose {
						fmt.Printf("Sending prompt to Gemini:\n---\n%s\n---\n", prompt)
					}

					s = util.NewSpinner("Generating PR content with Gemini...")
					s.Start()
					geminiCmd := exec.Command("gemini", prompt)
					var geminiOut bytes.Buffer
					geminiCmd.Stdout = &geminiOut
					
					err := geminiCmd.Run()
					s.Stop()

					if err == nil {
						output := geminiOut.String()
						
						// Parse output
						var title, body string
						lines := strings.Split(output, "\n")
						parsingBody := false
						var bodyBuilder strings.Builder

						for _, line := range lines {
							if strings.HasPrefix(line, "TITLE:") {
								title = strings.TrimSpace(strings.TrimPrefix(line, "TITLE:"))
							} else if strings.HasPrefix(line, "BODY:") {
								parsingBody = true
								continue
							} else if parsingBody {
								bodyBuilder.WriteString(line + "\n")
							}
						}
						body = strings.TrimSpace(bodyBuilder.String())

						if title != "" {
							fmt.Printf("Generated Title: %s\n", title)
							ghArgs = append(ghArgs, "--title", title)
						}
						if body != "" {
							ghArgs = append(ghArgs, "--body", body)
						}
					} else if util.GlobalContext.Verbose {
						fmt.Printf("Failed to run gemini: %v\n", err)
					}
				} else {
					// Check for dirty state first
					isDirty, dirtyErr := client.IsDirty()
					if dirtyErr == nil && isDirty {
						return fmt.Errorf("uncommitted changes detected. Please commit your changes to generate a PR description")
					}

					// Check for empty logs (clean state but no commits)
					if err == nil && logs == "" {
						return fmt.Errorf("no new commits found relative to %s. Cannot generate PR description", baseRef)
					}

					// If git log failed, log it in verbose but allow fallback (or error?)
					// For now, if we can't get logs, we probably shouldn't guess.
					if err != nil {
						return fmt.Errorf("failed to get commit logs: %w", err)
					}
				}
			} else if util.GlobalContext.Verbose {
				fmt.Println("Gemini CLI not found. Skipping PR content generation.")
			}
		}

		// Pass through any extra args to gh
		if len(args) > 0 {
			ghArgs = append(ghArgs, args...)
		}

		fmt.Println("Creating pull request...")
		if util.GlobalContext.DryRun {
			fmt.Printf("[Dry Run] Would run: gh %s\n", strings.Join(ghArgs, " "))
			return nil
		}

		// Check if gh is installed
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("gh cli not found. Please install GitHub CLI to use this command")
		}

		ghCmd := exec.Command("gh", ghArgs...)
		ghCmd.Stdin = os.Stdin
		ghCmd.Stdout = os.Stdout
		ghCmd.Stderr = os.Stderr

		if err := ghCmd.Run(); err != nil {
			// Don't wrap the error as gh likely printed the error message already
			return fmt.Errorf("failed to create PR (exit code %d)", ghCmd.ProcessState.ExitCode())
		}

		return nil
	},
}
