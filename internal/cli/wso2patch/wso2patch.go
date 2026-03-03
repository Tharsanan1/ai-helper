package wso2patch

import "github.com/spf13/cobra"

var WSO2PatchCmd = &cobra.Command{
	Use:   "wso2-patch",
	Short: "Manage coordinated WSO2 patch worktrees",
	Long: `Create coordinated worktrees across multiple repositories for a given APIM product version.

This command resolves per-repository upstream branches from configuration and creates
worktrees under a shared directory structure.`,
}

func init() {
	WSO2PatchCmd.AddCommand(createCmd)
	WSO2PatchCmd.AddCommand(deleteCmd)
}
