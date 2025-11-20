package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"wordsmith/internal/ui"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for wordsmith.

To load completions:

Bash:
  $ source <(wordsmith completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ wordsmith completion bash > /etc/bash_completion.d/wordsmith
  # macOS:
  $ wordsmith completion bash > $(brew --prefix)/etc/bash_completion.d/wordsmith

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ wordsmith completion zsh > "${fpath[1]}/_wordsmith"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ wordsmith completion fish | source
  # To load completions for each session, execute once:
  $ wordsmith completion fish > ~/.config/fish/completions/wordsmith.fish

PowerShell:
  PS> wordsmith completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> wordsmith completion powershell > wordsmith.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

var completionInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install shell completion for your current shell",
	Run: func(cmd *cobra.Command, args []string) {
		shell := detectShell()
		if shell == "" {
			ui.PrintError("Could not detect shell. Please use 'wordsmith completion [bash|zsh|fish|powershell]' manually")
			os.Exit(1)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			ui.PrintError("Could not find home directory: %v", err)
			os.Exit(1)
		}

		var completionDir, completionFile, rcFile, sourceLine string

		switch shell {
		case "zsh":
			// Create completion directory
			completionDir = filepath.Join(home, ".zsh", "completions")
			completionFile = filepath.Join(completionDir, "_wordsmith")
			rcFile = filepath.Join(home, ".zshrc")
			sourceLine = fmt.Sprintf("\nfpath=(%s $fpath)\nautoload -Uz compinit && compinit\n", completionDir)

		case "bash":
			completionDir = filepath.Join(home, ".bash_completion.d")
			completionFile = filepath.Join(completionDir, "wordsmith")
			rcFile = filepath.Join(home, ".bashrc")
			sourceLine = fmt.Sprintf("\n[ -f %s ] && source %s\n", completionFile, completionFile)

		case "fish":
			completionDir = filepath.Join(home, ".config", "fish", "completions")
			completionFile = filepath.Join(completionDir, "wordsmith.fish")
			rcFile = "" // Fish auto-loads from completions dir

		default:
			ui.PrintError("Auto-install not supported for %s. Please use 'wordsmith completion %s' manually", shell, shell)
			os.Exit(1)
		}

		// Create completion directory
		if err := os.MkdirAll(completionDir, 0755); err != nil {
			ui.PrintError("Failed to create completion directory: %v", err)
			os.Exit(1)
		}

		// Generate and write completion script
		f, err := os.Create(completionFile)
		if err != nil {
			ui.PrintError("Failed to create completion file: %v", err)
			os.Exit(1)
		}

		switch shell {
		case "zsh":
			rootCmd.GenZshCompletion(f)
		case "bash":
			rootCmd.GenBashCompletion(f)
		case "fish":
			rootCmd.GenFishCompletion(f, true)
		}
		f.Close()

		fmt.Printf("\033[0;32m✓ Installed completion script to %s\033[0m\n", completionFile)

		// Update shell rc file if needed
		if rcFile != "" {
			rcContent, _ := os.ReadFile(rcFile)
			if !strings.Contains(string(rcContent), "wordsmith") {
				f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
				if err != nil {
					ui.PrintWarning("Could not update %s: %v", rcFile, err)
					ui.PrintInfo("Please add manually: %s", sourceLine)
				} else {
					f.WriteString(sourceLine)
					f.Close()
					ui.PrintSuccess("Updated %s", rcFile)
				}
			}
		}

		fmt.Println()
		ui.PrintInfo("Restart your shell or run: source %s", rcFile)
	},
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "bash") {
		return "bash"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return ""
}

func init() {
	completionCmd.AddCommand(completionInstallCmd)
	rootCmd.AddCommand(completionCmd)
}
