package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Issue represents a GitHub issue
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// PR represents a GitHub pull request
type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	HeadRefName string `json:"headRefName"`
	HeadRepositoryOwner struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
	Url string `json:"url"`
}

// Comment represents a GitHub comment (issue or PR review)
type Comment struct {
	Body string `json:"body"`
}

// Client wraps gh CLI operations
type Client struct{}

// NewClient creates a new gh client
func NewClient() *Client {
	return &Client{}
}

// IsAvailable checks if gh CLI is installed
func (c *Client) IsAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// GetIssue fetches an issue from a repository
func (c *Client) GetIssue(repoURL string, issueNumber int) (*Issue, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("gh cli not found")
	}

	args := []string{"issue", "view", fmt.Sprintf("%d", issueNumber), "--repo", repoURL, "--json", "number,title,body"}
	cmd := exec.Command("gh", args...)
	
	// Capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue json: %w", err)
	}

	return &issue, nil
}

// GetPR fetches a pull request from a repository
func (c *Client) GetPR(repoURL string, prNumber int) (*PR, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("gh cli not found")
	}

	args := []string{"pr", "view", fmt.Sprintf("%d", prNumber), "--repo", repoURL, "--json", "number,title,body,headRefName,headRepositoryOwner,url"}
	cmd := exec.Command("gh", args...)
	
	// Capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pr: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse pr json: %w", err)
	}

	return &pr, nil
}

// GetComments fetches comments from a specific GitHub API endpoint
func (c *Client) GetComments(endpoint string) ([]Comment, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("gh cli not found")
	}

	args := []string{"api", "--paginate", endpoint}
	cmd := exec.Command("gh", args...)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch comments: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse comments json: %w", err)
	}

	return comments, nil
}

// ReviewThread represents a thread of comments on a PR
type ReviewThread struct {
	IsResolved bool `json:"isResolved"`
	Comments   struct {
		Nodes []struct {
			Body string `json:"body"`
		} `json:"nodes"`
	} `json:"comments"`
}

// GetReviewThreads fetches review threads for a PR using GraphQL
func (c *Client) GetReviewThreads(owner, repo string, prNumber int) ([]ReviewThread, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("gh cli not found")
	}

	query := fmt.Sprintf(`
query {
  repository(owner: "%s", name: "%s") {
    pullRequest(number: %d) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 50) {
            nodes {
              body
            }
          }
        }
      }
    }
  }
}`, owner, repo, prNumber)

	args := []string{"api", "graphql", "-f", fmt.Sprintf("query=%s", query)}
	cmd := exec.Command("gh", args...)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch review threads: %w", err)
	}

	var response struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []ReviewThread `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse graphql response: %w", err)
	}

	return response.Data.Repository.PullRequest.ReviewThreads.Nodes, nil
}
