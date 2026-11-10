package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/alexandrmotologa/hookcraft/internal/cache"
	"github.com/alexandrmotologa/hookcraft/internal/config"
	"github.com/alexandrmotologa/hookcraft/internal/git"
	"github.com/alexandrmotologa/hookcraft/internal/migrate"
	"github.com/alexandrmotologa/hookcraft/internal/runner"
	"github.com/alexandrmotologa/hookcraft/internal/server"
	"github.com/alexandrmotologa/hookcraft/internal/tui"
	"github.com/alexandrmotologa/hookcraft/web"
)

// Version information.
const (
	Version   = "1.0.0"
	BuildDate = "2026-11-10"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]
	args := os.Args[2:]

	out := tui.NewOutput(os.Stdout)

	switch command {
	case "run":
		runHookCommand(out, args)
	case "install":
		installHooksCommand(out, args)
	case "uninstall":
		uninstallHooksCommand(out, args)
	case "scan":
		scanSecretsCommand(out, args)
	case "init":
		initProjectCommand(out, args)
	case "migrate":
		migrateProjectCommand(out, args)
	case "cache":
		cacheCommand(out, args)
	case "status":
		statusCommand(out, args)
	case "serve", "studio":
		serveDashboardCommand(out, args)
	case "version", "-v", "--version":
		fmt.Printf("hookcraft v%s (%s)\n", Version, BuildDate)
	case "help", "-h", "--help":
		printUsage()
	default:
		// If command matches standard git hook name directly, treat as `run <hook>`
		for _, h := range git.SupportedHooks {
			if command == h {
				runHookCommand(out, append([]string{command}, args...))
				return
			}
		}
		fmt.Fprintf(os.Stderr, "Unknown command '%s'. Run 'hookcraft help' for usage.\n", command)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`HookCraft: Ultra-Fast Zero-Dependency Git Hook Orchestrator (< 15MB RAM)

Usage:
  hookcraft <command> [arguments]

Commands:
  run [hook]          Execute a Git hook (pre-commit, commit-msg, pre-push)
  install             Install HookCraft shims into .git/hooks
  uninstall           Remove HookCraft shims and restore previous hooks
  scan                Scan staged diffs or files for sensitive API keys and secrets
  init                Generate a tailored .hookcraft.yaml for the current repository
  migrate             Migrate existing Husky, Python pre-commit, or Lefthook config
  cache [clean]       Manage content-addressable execution cache
  status              Display Git hook installation and repository status
  serve, studio       Launch the embedded Web Studio and Benchmark Waterfall
  version             Print version and build details

Flags:
  --config <path>     Path to custom configuration file
  --non-interactive   Disable interactive prompts (for CI)
  --no-color          Disable colored output

Examples:
  hookcraft run pre-commit
  hookcraft install
  hookcraft scan --staged
  hookcraft studio --port 4173
`)
}

func runHookCommand(out *tui.Output, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to configuration file")
	nonInteractive := fs.Bool("non-interactive", false, "Disable interactive prompts")
	_ = fs.Parse(args)

	hookName := "pre-commit"
	var hookArgs []string

	extraArgs := fs.Args()
	if len(extraArgs) > 0 {
		hookName = extraArgs[0]
		hookArgs = extraArgs[1:]
	}

	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	cfg, _, err := config.LoadConfig(gitRunner.RepoRoot, *configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Config Error: %v\n", err)
		os.Exit(1)
	}

	if *nonInteractive {
		cfg.Settings.InteractiveApproval = false
	}

	stagedFiles, _ := gitRunner.GetStagedFiles()
	out.PrintBanner(hookName, len(stagedFiles))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	engine := runner.NewEngine(cfg, gitRunner)
	summary, runErr := engine.Run(ctx, hookName, hookArgs)

	if summary != nil {
		for _, step := range summary.Steps {
			out.PrintStep(step)
			if len(step.SecretFindings) > 0 {
				out.PrintSecretShieldAlert(step.SecretFindings)
			}
		}

		// Handle interactive approval if configured and formatters modified files
		if summary.Success && hookName == "pre-commit" && cfg.Settings.InteractiveApproval && !cfg.Settings.AutoStageFixes {
			var modified []string
			for _, s := range summary.Steps {
				if len(s.ModifiedFiles) > 0 {
					modified = append(modified, s.ModifiedFiles...)
				}
			}

			if len(modified) > 0 {
				diffFetcher := func() (string, error) {
					return gitRunner.GetStagedDiff(modified...)
				}
				approval, _ := tui.AskApprovalForFixes(out, modified, diffFetcher)
				if approval == tui.ApprovalYes {
					_ = gitRunner.StageFiles(modified...)
					fmt.Println(out.Colorize(tui.Bold+tui.Green, "✓ Formatted files staged."))
				}
			}
		}

		out.PrintSummary(summary)
	}

	if runErr != nil {
		os.Exit(1)
	}
}

func installHooksCommand(out *tui.Output, args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	hooksFlag := fs.String("hooks", "", "Comma-separated list of hooks to install")
	_ = fs.Parse(args)

	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	var targetHooks []string
	if *hooksFlag != "" {
		targetHooks = strings.Split(*hooksFlag, ",")
		for i := range targetHooks {
			targetHooks[i] = strings.TrimSpace(targetHooks[i])
		}
	}

	installed, err := gitRunner.InstallHooks(targetHooks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Install failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Successfully installed HookCraft shims into %s:\n",
		out.Colorize(tui.Bold+tui.Green, "✓"),
		filepath.Join(gitRunner.GitDir, "hooks"),
	)
	for _, h := range installed {
		fmt.Printf("  - %s\n", out.Colorize(tui.Cyan, h))
	}
}

func uninstallHooksCommand(out *tui.Output, args []string) {
	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	removed, err := gitRunner.UninstallHooks(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Uninstall failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Removed %d HookCraft hook shim(s):\n",
		out.Colorize(tui.Bold+tui.Yellow, "✓"),
		len(removed),
	)
	for _, h := range removed {
		fmt.Printf("  - %s\n", h)
	}
}

func scanSecretsCommand(out *tui.Output, args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	threshold := fs.Float64("threshold", 3.8, "Shannon entropy threshold")
	stagedOnly := fs.Bool("staged", true, "Scan staged changes only")
	_ = fs.Parse(args)

	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	scanner := runner.NewEngine(&config.Config{
		Settings: config.Settings{EntropyThreshold: *threshold},
	}, gitRunner).Scanner

	if *stagedOnly {
		hunks, err := gitRunner.GetParsedStagedHunks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[hookcraft] Failed to get staged diff: %v\n", err)
			os.Exit(1)
		}

		findings := scanner.ScanHunks(hunks)
		if len(findings) == 0 {
			fmt.Printf("%s Staged changes clean. No secrets or high-entropy tokens detected.\n",
				out.Colorize(tui.Bold+tui.Green, "✓"),
			)
			return
		}

		out.PrintSecretShieldAlert(findings)
		os.Exit(1)
	}
}

func initProjectCommand(out *tui.Output, args []string) {
	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	targetPath := filepath.Join(gitRunner.RepoRoot, ".hookcraft.yaml")
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("[hookcraft] Configuration already exists at %s\n", targetPath)
		return
	}

	content := config.GenerateStarterYAML(gitRunner.RepoRoot)
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Failed to write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Generated tailored configuration file: %s\n",
		out.Colorize(tui.Bold+tui.Green, "✓"),
		targetPath,
	)
}

func statusCommand(out *tui.Output, args []string) {
	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	branch, _ := gitRunner.Exec("rev-parse", "--abbrev-ref", "HEAD")
	branch = strings.TrimSpace(branch)

	hooks, _ := gitRunner.GetHookStatus()
	staged, _ := gitRunner.GetStagedFiles()
	hasUnstaged, _ := gitRunner.HasUnstagedChanges()

	fmt.Printf("%s Repository Status:\n", out.Colorize(tui.Bold+tui.Cyan, "HookCraft"))
	fmt.Printf("  Root Directory : %s\n", gitRunner.RepoRoot)
	fmt.Printf("  Current Branch : %s\n", out.Colorize(tui.Bold, branch))
	fmt.Printf("  Staged Files   : %d\n", len(staged))
	fmt.Printf("  Unstaged Dirty : %v\n\n", hasUnstaged)

	fmt.Println("Hook Shims:")
	for _, h := range git.SupportedHooks {
		installed := hooks[h]
		if installed {
			fmt.Printf("  %s %-14s (Active shim in .git/hooks/%s)\n",
				out.Colorize(tui.Bold+tui.Green, "✓"), h, h)
		} else {
			fmt.Printf("  %s %-14s (Not installed)\n",
				out.Colorize(tui.Dim+tui.Gray, "○"), h)
		}
	}
}

func serveDashboardCommand(out *tui.Output, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 4173, "HTTP port for Studio dashboard")
	_ = fs.Parse(args)

	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	cfg, configPath, _ := config.LoadConfig(gitRunner.RepoRoot, "")
	srv := server.NewServer(gitRunner.RepoRoot, gitRunner, cfg, configPath, web.DistFS, *port)

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Studio Server error: %v\n", err)
		os.Exit(1)
	}
}

func migrateProjectCommand(out *tui.Output, args []string) {
	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	targetPath := filepath.Join(gitRunner.RepoRoot, ".hookcraft.yaml")
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("[hookcraft] Warning: .hookcraft.yaml already exists at %s\n", targetPath)
		fmt.Print("Overwrite existing configuration? [y/N]: ")
		var resp string
		_, _ = fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "y" {
			fmt.Println("Migration aborted.")
			return
		}
	}

	result, err := migrate.DetectAndMigrate(gitRunner.RepoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Migration failed: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(targetPath, []byte(result.GeneratedYAML), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Failed to write .hookcraft.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Successfully migrated from %s (%s):\n",
		out.Colorize(tui.Bold+tui.Green, "✓"),
		out.Colorize(tui.Bold+tui.Cyan, result.SourceType),
		result.SourcePath,
	)
	for _, h := range result.MigratedHooks {
		fmt.Printf("  - %s\n", h)
	}
	fmt.Printf("\nGenerated clean configuration at: %s\n", targetPath)
}

func cacheCommand(out *tui.Output, args []string) {
	gitRunner, err := git.NewRunner("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookcraft] Error: %v\n", err)
		os.Exit(1)
	}

	mgr := cache.NewManager(gitRunner.RepoRoot)

	if len(args) > 0 && args[0] == "clean" {
		if err := mgr.Clean(); err != nil {
			fmt.Fprintf(os.Stderr, "[hookcraft] Failed to clean cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s Cleaned execution cache.\n", out.Colorize(tui.Bold+tui.Green, "✓"))
		return
	}

	fmt.Printf("%s Cache Status:\n", out.Colorize(tui.Bold+tui.Cyan, "HookCraft"))
	fmt.Printf("  Cache Directory : %s\n", mgr.CacheDir)
	fmt.Printf("  Cached Entries  : %d\n", mgr.Size())
	fmt.Println("\nTo clean the cache, run: hookcraft cache clean")
}
