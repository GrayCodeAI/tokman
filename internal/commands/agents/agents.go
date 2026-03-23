package agents

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/agents"
	"github.com/GrayCodeAI/tokman/internal/commands/registry"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage AI coding agent integrations",
	Long: `Detect, configure, and manage integrations with various AI coding agents.

Supported agents:
  - Claude Code (claude-code)
  - Cursor (cursor)
  - Cline (cline)
  - Continue (continue)
  - Aider (aider)
  - Codex CLI (codex-cli)

Examples:
  tokman agents list          # Show all detected agents
  tokman agents status        # Show detailed status
  tokman agents setup aider   # Configure Aider for TokMan`,
}

func init() {
	agentsCmd.AddCommand(agentsListCmd())
	agentsCmd.AddCommand(agentsStatusCmd())
	agentsCmd.AddCommand(agentsSetupCmd())
	agentsCmd.AddCommand(agentsInstallCmd())
	registry.Add(func() { registry.Register(agentsCmd) })
}

func agentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all supported AI agents",
		Run: func(cmd *cobra.Command, args []string) {
			statuses := agents.DetectAll()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDISPLAY NAME\tINSTALLED\tCONFIGURED")

			for _, s := range statuses {
				installed := "no"
				if s.Installed {
					installed = "yes"
				}
				configured := "no"
				if s.Configured {
					configured = "yes"
				}

				agent := agents.GetAgent(s.Name)
				displayName := s.Name
				if agent != nil {
					displayName = agent.DisplayName
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, displayName, installed, configured)
			}
			w.Flush()
		},
	}
}

func agentsStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [agent]",
		Short: "Show detailed status of agents",
		Long: `Show detailed status of all agents or a specific agent.
		
If no agent is specified, shows status of all detected agents.`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				agent := agents.GetAgent(args[0])
				if agent == nil {
					fmt.Fprintf(os.Stderr, "Unknown agent: %s\n", args[0])
					os.Exit(1)
				}

				status, err := agent.StatusFunc()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
					os.Exit(1)
				}

				printAgentStatus(status)
			} else {
				statuses := agents.DetectAll()
				for _, s := range statuses {
					printAgentStatus(s)
					fmt.Println()
				}
			}
		},
	}
}

func agentsSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup <agent>",
		Short: "Configure an agent for TokMan integration",
		Long: `Configure an AI agent to work with TokMan for token optimization.

This will:
  - Create necessary configuration directories
  - Write configuration files with TokMan settings
  - Enable token caching and optimization`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentName := args[0]

			agent := agents.GetAgent(agentName)
			if agent == nil {
				fmt.Fprintf(os.Stderr, "Unknown agent: %s\n", agentName)
				fmt.Fprintln(os.Stderr, "\nSupported agents:")
				for _, a := range agents.AllAgents {
					fmt.Fprintf(os.Stderr, "  - %s (%s)\n", a.Name, a.DisplayName)
				}
				os.Exit(1)
			}

			if !agent.DetectFunc() {
				fmt.Fprintf(os.Stderr, "%s is not installed.\n", agent.DisplayName)
				fmt.Fprintln(os.Stderr, "\nInstallation instructions:")
				fmt.Fprintln(os.Stderr, agents.InstallInstructions(agentName))
				os.Exit(1)
			}

			fmt.Printf("Setting up %s for TokMan integration...\n", agent.DisplayName)
			if err := agents.SetupAgent(agentName); err != nil {
				fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully configured %s!\n", agent.DisplayName)
			fmt.Printf("Config file: %s\n", agent.ConfigPath)
		},
	}
}

func agentsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <agent>",
		Short: "Show installation instructions for an agent",
		Long:  `Display installation instructions for supported AI agents.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentName := args[0]
			agent := agents.GetAgent(agentName)

			if agent == nil {
				fmt.Fprintf(os.Stderr, "Unknown agent: %s\n", agentName)
				fmt.Fprintln(os.Stderr, "\nSupported agents:")
				for _, a := range agents.AllAgents {
					fmt.Fprintf(os.Stderr, "  - %s\n", a.Name)
				}
				os.Exit(1)
			}

			fmt.Printf("# Installation: %s\n\n", agent.DisplayName)
			fmt.Println(agents.InstallInstructions(agentName))
		},
	}
}

func printAgentStatus(s *agents.AgentStatus) {
	fmt.Printf("Agent: %s\n", s.Name)
	fmt.Printf("  Installed: %v\n", s.Installed)
	fmt.Printf("  Configured: %v\n", s.Configured)
	if s.Version != "" {
		fmt.Printf("  Version: %s\n", s.Version)
	}
	if s.ConfigPath != "" {
		fmt.Printf("  Config: %s\n", s.ConfigPath)
	}
	if s.ErrorMessage != "" {
		fmt.Printf("  Error: %s\n", s.ErrorMessage)
	}
}
