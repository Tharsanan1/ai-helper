package setupcopilot

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	// Flags for setup-copilot command
	noCommit bool
)

// SetupCopilotCmd represents the setup-copilot command
var SetupCopilotCmd = &cobra.Command{
	Use:   "setup-copilot",
	Short: "Setup GitHub Copilot configuration files",
	Long: `Setup GitHub Copilot by copying configuration files to the current repository.

This command will:
1. Copy copilot-instructions.md to .github/ folder
2. Copy copilot-setup-steps.yml to .github/workflows/ folder
3. Commit the changes (unless --no-commit is specified)

The source paths for these files are configured via:
  - copilot_setup.instructions_md_path
  - copilot_setup.workflow_yml_path

Example:
  aihelper setup-copilot
  aihelper setup-copilot --no-commit`,
	RunE: runSetupCopilot,
}

func init() {
	SetupCopilotCmd.Flags().BoolVar(&noCommit, "no-commit", false, "Don't commit the changes after copying files")
}

func runSetupCopilot(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Validate configuration
	if cfg.CopilotSetup.InstructionsMdPath == "" {
		return fmt.Errorf("copilot_setup.instructions_md_path is not configured. Set it with: aihelper config set copilot_setup.instructions_md_path <path>")
	}
	if cfg.CopilotSetup.WorkflowYmlPath == "" {
		return fmt.Errorf("copilot_setup.workflow_yml_path is not configured. Set it with: aihelper config set copilot_setup.workflow_yml_path <path>")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .github directory if it doesn't exist
	githubDir := filepath.Join(cwd, ".github")
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github directory: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	workflowsDir := filepath.Join(githubDir, "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Copy copilot-instructions.md to .github/
	instructionsSrc := cfg.CopilotSetup.InstructionsMdPath
	instructionsDst := filepath.Join(githubDir, "copilot-instructions.md")

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Copying %s to %s\n", instructionsSrc, instructionsDst)
	}

	if !util.GlobalContext.IsDryRun() {
		if err := copyFile(instructionsSrc, instructionsDst); err != nil {
			return fmt.Errorf("failed to copy copilot-instructions.md: %w", err)
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Copied copilot-instructions.md to .github/\n")
	} else {
		fmt.Println("✓ Copied copilot-instructions.md to .github/")
	}

	// Copy copilot-setup-steps.yml to .github/workflows/
	workflowSrc := cfg.CopilotSetup.WorkflowYmlPath
	workflowDst := filepath.Join(workflowsDir, "copilot-setup-steps.yml")

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Copying %s to %s\n", workflowSrc, workflowDst)
	}

	if !util.GlobalContext.IsDryRun() {
		if err := copyFile(workflowSrc, workflowDst); err != nil {
			return fmt.Errorf("failed to copy copilot-setup-steps.yml: %w", err)
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Copied copilot-setup-steps.yml to .github/workflows/\n")
	} else {
		fmt.Println("✓ Copied copilot-setup-steps.yml to .github/workflows/")
	}

	// Commit changes unless --no-commit is specified
	if !noCommit && !util.GlobalContext.IsDryRun() {
		if err := commitChanges(instructionsDst, workflowDst); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}

		if util.GlobalContext.IsColorEnabled() {
			color.Green("✓ Committed changes with message: \"Add copilot setup workflow and instruction md\"\n")
		} else {
			fmt.Println("✓ Committed changes with message: \"Add copilot setup workflow and instruction md\"")
		}
	} else if noCommit {
		fmt.Println("Skipping commit (--no-commit flag specified)")
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sync to ensure all data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// commitChanges stages and commits the copied files
func commitChanges(files ...string) error {
	// Stage files
	addArgs := append([]string{"add"}, files...)
	addCmd := exec.Command("git", addArgs...)
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s - %w", string(output), err)
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", "Add copilot setup workflow and instruction md")
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s - %w", string(output), err)
	}

	return nil
}
