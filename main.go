package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/obiMadu/wtmag/internal/agent"
	createinput "github.com/obiMadu/wtmag/internal/cli"
	"github.com/obiMadu/wtmag/internal/config"
	promptbuilder "github.com/obiMadu/wtmag/internal/prompt"
	"github.com/obiMadu/wtmag/internal/repository"
	"github.com/obiMadu/wtmag/internal/runtime"
	"github.com/obiMadu/wtmag/internal/session"
	"github.com/obiMadu/wtmag/internal/source"
	_ "github.com/obiMadu/wtmag/internal/source/providers"
	"github.com/obiMadu/wtmag/internal/workitem"
	"github.com/spf13/cobra"
)

var (
	githubFlag        int
	jiraFlag          string
	promptFlag        string
	projectFlag       string
	typeFlag          string
	launchFlag        string
	agentFlag         string
	cleanupIDFlag     string
	promoteIDFlag     string
	sessionLaunchFlag bool
	windowLaunchFlag  bool
	allFlag           bool
	forceFlag         bool
)

var rootCmd = &cobra.Command{
	Use:   "wtmag",
	Short: "Local orchestrator for task descriptions and PR reviews",
	Long: `wtmag creates isolated local worktrees and tmux-backed agent workers 
from task descriptions (GitHub issues, Jira tickets, custom prompts) and for PR reviews.`,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new worker from a task description or PR review target",
	RunE:  createCmdRun,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent workers",
	RunE:  listCmdRun,
}

var attachCmd = &cobra.Command{
	Use:   "attach [worker-id]",
	Short: "Attach to an agent worker",
	Args:  cobra.ExactArgs(1),
	RunE:  attachCmdRun,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove worktree and cleanup worker",
	RunE:  cleanupCmdRun,
}

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a window worker into a dedicated session",
	RunE:  promoteCmdRun,
}

func init() {
	rootCmd.AddCommand(createCmd, listCmd, attachCmd, cleanupCmd, promoteCmd)

	createCmd.Flags().IntVar(&githubFlag, "github", 0, "GitHub issue/PR number")
	createCmd.Flags().StringVar(&jiraFlag, "jira", "", "Jira ticket ID (e.g., PROJ-123)")
	createCmd.Flags().StringVar(&promptFlag, "prompt", "", "Raw prompt text")
	createCmd.Flags().StringVar(&projectFlag, "project", "", "Project name (auto-detected if not set)")
	createCmd.Flags().StringVarP(&typeFlag, "type", "t", "", "Type (required for --github: issue, pr)")
	createCmd.Flags().StringVar(&launchFlag, "launch", "", "Launch worker in `session` or `window` mode")
	createCmd.Flags().StringVar(&agentFlag, "agent", "", "AI agent to spawn (defaults to agents.default.name from wtmag.toml or ~/.config/wtmag/config.toml)")
	createCmd.Flags().BoolVarP(&sessionLaunchFlag, "session", "s", false, "Launch worker in a dedicated tmux session")
	createCmd.Flags().BoolVarP(&windowLaunchFlag, "window", "w", false, "Launch worker in the current tmux session as a window")
	listCmd.Flags().BoolVar(&allFlag, "all", false, "List workers across all projects")

	cleanupCmd.Flags().StringVar(&cleanupIDFlag, "id", "", "Worker ID to cleanup")
	cleanupCmd.Flags().BoolVar(&forceFlag, "force", false, "Force worktree removal with `wt remove --force`")
	cleanupCmd.MarkFlagRequired("id")

	promoteCmd.Flags().StringVar(&promoteIDFlag, "id", "", "Window worker ID to promote into a dedicated session")
	promoteCmd.MarkFlagRequired("id")
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

	placementKind, err := resolveLaunchPlacement(workItem)
	if err != nil {
		return err
	}

	selectedAgent := strings.TrimSpace(agentFlag)
	if selectedAgent == "" {
		selectedAgent, err = agent.DefaultName()
		if err != nil {
			return err
		}
	}

	localRuntime := runtime.LocalRuntime{}
	workerSession, err := localRuntime.Create(project, workItem, workerPrompt, selectedAgent, placementKind)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Created worker: %s (%s)\n", workerSession.ID, workerSession.Placement)
	fmt.Printf("  Attach: wtmag attach %s\n", workerSession.ID)
	return nil
}

func listCmdRun(cmd *cobra.Command, args []string) error {
	localRuntime := runtime.LocalRuntime{}
	var (
		sessions []runtime.Session
		err      error
	)

	if allFlag {
		sessions, err = localRuntime.ListAll()
	} else {
		project, err := resolveProjectName("")
		if err != nil {
			return err
		}

		sessions, err = localRuntime.List(project)
	}
	if err != nil {
		return err
	}

	for _, s := range sessions {
		fmt.Printf("%s - %s (%s)\n", s.ID, s.Status, s.Placement)
	}
	return nil
}

func attachCmdRun(cmd *cobra.Command, args []string) error {
	localRuntime := runtime.LocalRuntime{}
	return localRuntime.Attach(args[0])
}

func cleanupCmdRun(cmd *cobra.Command, args []string) error {
	localRuntime := runtime.LocalRuntime{}
	return localRuntime.Cleanup(cleanupIDFlag, forceFlag)
}

func promoteCmdRun(cmd *cobra.Command, args []string) error {
	localRuntime := runtime.LocalRuntime{}
	workerSession, err := localRuntime.Promote(promoteIDFlag)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Promoted worker: %s (%s)\n", workerSession.ID, workerSession.Placement)
	fmt.Printf("  Attach: wtmag attach %s\n", workerSession.ID)
	return nil
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

func resolveLaunchPlacement(workItem workitem.WorkItem) (session.PlacementKind, error) {
	trimmedLaunch := strings.ToLower(strings.TrimSpace(launchFlag))

	if sessionLaunchFlag && windowLaunchFlag {
		return "", fmt.Errorf("-s and -w are mutually exclusive")
	}

	if trimmedLaunch != "" && trimmedLaunch != string(session.PlacementKindSession) && trimmedLaunch != string(session.PlacementKindWindow) {
		return "", fmt.Errorf("unknown launch mode %q (use 'session' or 'window')", trimmedLaunch)
	}

	if sessionLaunchFlag {
		if trimmedLaunch != "" && trimmedLaunch != string(session.PlacementKindSession) {
			return "", fmt.Errorf("-s conflicts with --launch %s", trimmedLaunch)
		}

		return session.PlacementKindSession, nil
	}

	if windowLaunchFlag {
		if trimmedLaunch != "" && trimmedLaunch != string(session.PlacementKindWindow) {
			return "", fmt.Errorf("-w conflicts with --launch %s", trimmedLaunch)
		}

		return session.PlacementKindWindow, nil
	}

	switch trimmedLaunch {
	case string(session.PlacementKindSession):
		return session.PlacementKindSession, nil
	case string(session.PlacementKindWindow):
		return session.PlacementKindWindow, nil
	}

	configuredValues, err := config.Load()
	if err != nil {
		return "", err
	}

	configuredLaunch := configuredValues.Launch.Implementation
	if workItem.Mode == workitem.ModeReview {
		configuredLaunch = configuredValues.Launch.Review
	}

	switch strings.ToLower(strings.TrimSpace(configuredLaunch)) {
	case string(session.PlacementKindSession):
		return session.PlacementKindSession, nil
	case string(session.PlacementKindWindow):
		return session.PlacementKindWindow, nil
	default:
		return "", fmt.Errorf("invalid launch default %q for %s work in %s (use 'session' or 'window')", configuredLaunch, workItem.Mode, config.LookupDescription())
	}
}
