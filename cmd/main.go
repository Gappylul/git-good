package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gappylul/git-good/internal"
)

var (
	Version   = "development"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	versionFlag := flag.Bool("v", false, "Print version information")
	fullVersionFlag := flag.Bool("version", false, "Print detailed version information")
	flag.Parse()

	if *versionFlag || *fullVersionFlag {
		printVersion(*fullVersionFlag)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "init":
		handleInit()
	case "apply":
		handleApply()
	case "run-hook":
		handleRunHook()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printVersion(full bool) {
	if full {
		fmt.Printf("git-good version:  %s\n", Version)
		fmt.Printf("Git Commit:        %s\n", Commit)
		fmt.Printf("Build Time:        %s\n", BuildTime)
	} else {
		fmt.Println(Version)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  git-good [options] <command>")
	fmt.Println("\nCommands:")
	fmt.Println("  init       - Initialize config and register hook template")
	fmt.Println("  apply      - Validate and apply git-good.yaml changes to Git")
	fmt.Println("  run-hook   - Scan staged changes for secrets (called by git hook)")
	fmt.Println("\nOptions:")
	fmt.Println("  -v         - Print short version string")
	fmt.Println("  -version   - Print detailed build details")
}

func handleInit() {
	configPath, err := internal.DefaultConfigPath()
	if err != nil {
		fmt.Printf("Failed to resolve git root: %v\n", err)
		os.Exit(1)
	}

	if _, err = os.Stat(configPath); err == nil {
		fmt.Println("git-good.yaml already exists in this directory.")
	} else {
		if err = internal.GenerateDefaultConfig(); err != nil {
			fmt.Printf("Failed to generate config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Created a baseline configuration file: git-good.yaml")
	}

	fmt.Println("Edit git-good.yaml, then run: git-good apply")
}

func handleApply() {
	config, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("YAML Validation Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("🍰 git-good.yaml verified successfully! Loaded %d custom rules.\n", len(config.Rules))

	gitRoot, err := internal.GitRoot()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	gitHooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if _, err = os.Stat(gitHooksDir); os.IsNotExist(err) {
		fmt.Println("Error: Directory '.git/hooks' not found. Are you sure you're in a Git repository?")
		os.Exit(1)
	}

	hookPath := filepath.Join(gitHooksDir, "pre-commit")

	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to resolve binary location: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.ToSlash(binaryPath)

	hookContent := fmt.Sprintf("#!/bin/sh\n\"%s\" run-hook\n", binaryPath)

	err = os.WriteFile(hookPath, []byte(hookContent), 0755)
	if err != nil {
		fmt.Printf("Failed to write Git pre-commit hook: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🍰 Hook applied! git-good will now check every commit automatically.")
}

func handleRunHook() {
	config, err := internal.LoadConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[git-good] Warning: could not load config: %v\n", err)
		os.Exit(0)
	}

	matches, err := internal.ScanStagedChanges(config)
	if err != nil {
		fmt.Printf("[git-good] Scanning failed: %v\n", err)
		os.Exit(1)
	}

	if len(matches) > 0 {
		if config.Settings.WebhookURL != "" {
			fmt.Println("[git-good] Dispatching webhook alert...")
			err := internal.SendWebhook(config.Settings.WebhookURL, matches)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[git-good] Webhook dispatch failed: %v\n", err)
			}
		}

		fmt.Println("\n[git-good] COMMIT BLOCKED!")
		fmt.Println("It looks like you're trying to commit credentials or secrets:")
		fmt.Println("------------------------------------------------------------")

		for _, match := range matches {
			fmt.Printf("  File            : %s\n", match.FileName)
			fmt.Printf("  Rule Triggered  : %s\n", match.RuleName)
			fmt.Printf("  Leaked String   : %s\n", match.Content)
			fmt.Println("------------------------------------------------------------")
		}

		fmt.Println("💡 Remove the secrets from your code, stage the changes, and try again.")

		if config.Settings.FailOnMatch {
			os.Exit(1)
		}
	}

	fmt.Println("[git-good] 🍓 Scan clean. No hardcoded secrets detected.")
	os.Exit(0)
}
