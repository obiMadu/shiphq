package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/obiMadu/shiphq/internal/agent"
	createinput "github.com/obiMadu/shiphq/internal/cli"
	"github.com/obiMadu/shiphq/internal/config"
	promptbuilder "github.com/obiMadu/shiphq/internal/prompt"
	"github.com/obiMadu/shiphq/internal/repository"
	"github.com/obiMadu/shiphq/internal/runtime"
	"github.com/obiMadu/shiphq/internal/source"
	_ "github.com/obiMadu/shiphq/internal/source/providers"
	"github.com/obiMadu/shiphq/internal/workitem"
	"github.com/spf13/cobra"
)

var (
	githubFlag  int
	jiraFlag    string
	promptFlag  string
	projectFlag string
	typeFlag    string
	idFlag      string
	agentFlag   string
	forceFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "shiphq",
	Short: "Local orchestrator for task descriptions and PR reviews",
	Long: `shiphq creates isolated local worktrees and tmux-backed agent sessions 
from task descriptions (GitHub issues, Jira tickets, custom prompts) and for PR reviews.`,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new worker session from a task description or PR review target",
	RunE:  createCmdRun,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active agent sessions",
	RunE:  listCmdRun,
}

var attachCmd = &cobra.Command{
	Use:   "attach [session-id]",
	Short: "Attach to an agent session",
	Args:  cobra.ExactArgs(1),
	RunE:  attachCmdRun,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove worktree and cleanup session",
	RunE:  cleanupCmdRun,
}

func init() {
	rootCmd.AddCommand(createCmd, listCmd, attachCmd, cleanupCmd)

	createCmd.Flags().IntVar(&githubFlag, "github", 0, "GitHub issue/PR number")
	createCmd.Flags().StringVar(&jiraFlag, "jira", "", "Jira ticket ID (e.g., PROJ-123)")
	createCmd.Flags().StringVar(&promptFlag, "prompt", "", "Raw prompt text")
	createCmd.Flags().StringVar(&projectFlag, "project", "", "Project name (auto-detected if not set)")
	createCmd.Flags().StringVarP(&typeFlag, "type", "t", "", "Type (required for --github: issue, pr)")
	createCmd.Flags().StringVar(&agentFlag, "agent", "", "AI agent to spawn (defaults to config agents.default.name)")

	cleanupCmd.Flags().StringVar(&idFlag, "id", "", "Session ID to cleanup")
	cleanupCmd.Flags().BoolVar(&forceFlag, "force", false, "Force worktree removal with `wt remove --force`")
	cleanupCmd.MarkFlagRequired("id")
}

func main() {
	if err := config.EnsureConfigFile(defaultConfigFileContents); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createCmdRun(cmd *cobra.Command, args []string) error {
	project, err := resolveProjectName(projectFlag)
	if err != nil {
		return err
	}

	createInput, err := createinput.ResolveCreateInput(githubFlag, jiraFlag, promptFlag, typeFlag)
	if err != nil {
		return err
	}

	workItem, err := source.Fetch(createInput.SourceRef)
	if err != nil {
		return err
	}

	repositoryTarget, err := repository.DetectTarget()
	if err != nil {
		return err
	}

	workerPrompt := promptbuilder.BuildDefault(workItem, repositoryTarget)
	if createInput.PromptOverride != "" {
		workerPrompt = promptbuilder.BuildOverride(workItem, repositoryTarget, createInput.PromptOverride)
	}

	selectedAgent := strings.TrimSpace(agentFlag)
	if selectedAgent == "" {
		selectedAgent, err = agent.DefaultName()
		if err != nil {
			return err
		}
	}

	localRuntime := runtime.LocalRuntime{}
	session, err := localRuntime.Create(project, workItem, workerPrompt, selectedAgent)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Created session: %s\n", session.ID)
	fmt.Printf("  Attach: shiphq attach %s\n", session.ID)
	return nil
}

func listCmdRun(cmd *cobra.Command, args []string) error {
	project, err := resolveProjectName("")
	if err != nil {
		return err
	}

	localRuntime := runtime.LocalRuntime{}
	sessions, err := localRuntime.List(project)
	if err != nil {
		return err
	}

	for _, s := range sessions {
		fmt.Printf("%s - %s\n", s.ID, s.Status)
	}
	return nil
}

func attachCmdRun(cmd *cobra.Command, args []string) error {
	localRuntime := runtime.LocalRuntime{}
	return localRuntime.Attach(args[0])
}

func cleanupCmdRun(cmd *cobra.Command, args []string) error {
	localRuntime := runtime.LocalRuntime{}
	return localRuntime.Cleanup(idFlag, forceFlag)
}

func resolveProjectName(project string) (string, error) {
	projectName := project
	if projectName == "" {
		projectName = detectProject()
	}

	normalizedProjectName := workitem.NormalizeIdentifier(projectName)
	if normalizedProjectName == "" {
		return "", fmt.Errorf("project name cannot be empty")
	}

	return normalizedProjectName, nil
}

func detectProject() string {
	if project := detectProjectFromGit(); project != "" {
		return project
	}

	wd, _ := os.Getwd()
	base := filepath.Base(wd)
	if base == "." || base == "/" || base == "" {
		return "project"
	}
	return base
}

func detectProjectFromGit() string {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	commonDir := strings.TrimSpace(string(out))
	if commonDir == "" {
		return ""
	}

	if !filepath.IsAbs(commonDir) {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		commonDir = filepath.Join(wd, commonDir)
	}

	commonDir = filepath.Clean(commonDir)
	if filepath.Base(commonDir) == ".git" {
		return filepath.Base(filepath.Dir(commonDir))
	}

	return filepath.Base(commonDir)
}
