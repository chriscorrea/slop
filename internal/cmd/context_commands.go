package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"slop/internal/manifest"

	"github.com/spf13/cobra"
)

// createContextCommand creates the context management command with subcommands
func createContextCommand() *cobra.Command {
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Manage persistent context for the current directory",
		Long:  "Manage persistent file context using .slop/context manifest files. Searches for manifest in current directory and parent directories.",
	}

	// add subcommands
	contextCmd.AddCommand(createContextAddCommand())
	contextCmd.AddCommand(createContextListCommand())
	contextCmd.AddCommand(createContextClearCommand())

	return contextCmd
}

// createContextAddCommand creates the 'context add' subcommand
func createContextAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path...>",
		Short: "Add files to the current directory context",
		Long:  `Add one or more files to the current directory context. Creates context manifest for the current directory if no manifest is found.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := manifest.NewManifestManager("")
			manifestPath := manager.GetManifestPath()

			// expand and validate paths
			var validPaths []string
			for _, arg := range args {
				// convert to absolute path for validation
				absPath, err := filepath.Abs(arg)
				if err != nil {
					return fmt.Errorf("invalid path %q: %w", arg, err)
				}

				// check if path exists
				if _, err := os.Stat(absPath); err != nil {
					return fmt.Errorf("path does not exist: %q", arg)
				}

				// store the original argument (which may be relative)
				validPaths = append(validPaths, arg)
			}

			// add paths to manifest
			err := manager.AddPaths(manifestPath, validPaths)
			if err != nil {
				return fmt.Errorf("failed to add paths to context: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added %d path(s) to project context:\n", len(validPaths))
			for _, path := range validPaths {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", path)
			}

			return nil
		},
	}
}

// createContextListCommand creates the 'context list' subcommand
func createContextListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all files in the current directory's context",
		Long:  "Display all files in the context manifest found in current directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := manifest.NewManifestManager("")

			manifestPath, projectRoot, err := manager.FindManifest()
			if err != nil {
				return fmt.Errorf("failed to find manifest: %w", err)
			}

			if manifestPath == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No context manifest found. Use 'slop context add' to create one in current directory.")
				return nil
			}

			paths, err := manager.LoadManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to load manifest: %w", err)
			}

			if len(paths) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Context manifest is empty. Use 'slop context add' to add files.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Context files (from %s):\n", projectRoot)
			for i, path := range paths {
				fmt.Fprintf(cmd.OutOrStdout(), "%3d. %s\n", i+1, path)
			}

			return nil
		},
	}
}

// createContextClearCommand creates the 'context clear' subcommand
func createContextClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all files from the current directory's context",
		Long:  "Clear all files from the context manifest found in current directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := manifest.NewManifestManager("")

			manifestPath, projectRoot, err := manager.FindManifest()
			if err != nil {
				return fmt.Errorf("failed to find manifest: %w", err)
			}

			if manifestPath == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No context manifest found.")
				return nil
			}

			// load existing paths to show what's being cleared
			paths, err := manager.LoadManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to load manifest: %w", err)
			}

			if len(paths) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Context manifest is already empty.")
				return nil
			}

			// clear the manifest
			err = manager.ClearManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to clear context: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Cleared %d path(s) from context manifest in %s\n", len(paths), projectRoot)

			return nil
		},
	}
}
