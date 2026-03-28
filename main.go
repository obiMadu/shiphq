package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/obiMadu/shiphq/internal/adapters"
	"github.com/obiMadu/shiphq/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	githubFlag  int
	jiraFlag    string
	promptFlag  string
	runtimeFlag string
	projectFlag string
	typeFlag    string
	idFlag      string
)

var rootCmd = &cobra.Command{
	Use:   "shiphq",
	Short: "Agent workflow orchestrator for Git worktrees and cloud sandboxes",
	Long: `shiphq creates isolated worktrees and agent sessions from GitHub issues, 
Jira tickets, or custom prompts. Supports local (tmux + worktrunk) and 
cloud (Daytona) runtimes.`,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent session from an issue or prompt",
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
	createCmd.Flags().StringVar(&runtimeFlag, "runtime", "local", "Runtime backend (local, daytona)")
	createCmd.Flags().StringVar(&projectFlag, "project", "", "Project name (auto-detected if not set)")
	createCmd.Flags().StringVarP(&typeFlag, "type", "t", "", "Type (required for --github: issue, pr)")

	cleanupCmd.Flags().StringVar(&idFlag, "id", "", "Session ID to cleanup")
	cleanupCmd.MarkFlagRequired("id")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createCmdRun(cmd *cobra.Command, args []string) error {
	project := projectFlag
	if project == "" {
		project = detectProject()
	}

	var task adapters.Task
	var err error

	switch {
	case githubFlag != 0:
		task, err = adapters.FetchGitHub(githubFlag, typeFlag)
		if promptFlag != "" {
			task.Prompt = promptFlag
		}
	case jiraFlag != "":
		task, err = adapters.FetchJira(jiraFlag)
		if promptFlag != "" {
			task.Prompt = promptFlag
		}
	case promptFlag != "":
		task = adapters.TaskFromPrompt(promptFlag)
	default:
		return fmt.Errorf("must specify --github, --jira, or --prompt")
	}

	if err != nil {
		return err
	}

	rt, err := runtime.Get(runtimeFlag)
	if err != nil {
		return err
	}

	session, err := rt.Create(project, task)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Created session: %s\n", session.ID)
	fmt.Printf("  Runtime: %s\n", session.Runtime)
	fmt.Printf("  Attach: shiphq attach %s\n", session.ID)
	return nil
}

func listCmdRun(cmd *cobra.Command, args []string) error {
	rt, _ := runtime.Get("local")
	sessions, err := rt.List(detectProject())
	if err != nil {
		return err
	}

	for _, s := range sessions {
		fmt.Printf("%s (%s) - %s\n", s.ID, s.Runtime, s.Status)
	}
	return nil
}

func attachCmdRun(cmd *cobra.Command, args []string) error {
	rt, _ := runtime.Get("local")
	return rt.Attach(args[0])
}

func cleanupCmdRun(cmd *cobra.Command, args []string) error {
	rt, _ := runtime.Get("local")
	return rt.Cleanup(idFlag)
}

func detectProject() string {
	wd, _ := os.Getwd()
	base := filepath.Base(wd)
	if base == "." || base == "/" || base == "" {
		return "project"
	}
	return base
}
