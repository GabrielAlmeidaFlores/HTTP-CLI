package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/exporter"
	"github.com/user/http-cli/internal/parser"
	"github.com/user/http-cli/internal/storage"
	"github.com/user/http-cli/internal/transport"
	"github.com/user/http-cli/internal/ui"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "http-cli",
	Short: "A terminal-based HTTP client",
	Long:  "http-cli is a terminal-based HTTP client with a TUI interface, vim-style keybindings, and collection management.",
	RunE:  runTUI,
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import requests from various formats",
}

var importCurlCmd = &cobra.Command{
	Use:   "curl <command>",
	Short: "Import a cURL command",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportCurl,
}

var importPostmanCmd = &cobra.Command{
	Use:   "postman <file>",
	Short: "Import a Postman collection",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportPostman,
}

var importHTTPFileCmd = &cobra.Command{
	Use:   "http-file <file>",
	Short: "Import a .http or .rest file",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportHTTPFile,
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export requests to various formats",
}

var exportCurlCmd = &cobra.Command{
	Use:   "curl <request-name>",
	Short: "Export a request as a cURL command",
	Args:  cobra.ExactArgs(1),
	RunE:  runExportCurl,
}

var exportPostmanCmd = &cobra.Command{
	Use:   "postman <output-file>",
	Short: "Export all requests as a Postman collection",
	Args:  cobra.ExactArgs(1),
	RunE:  runExportPostman,
}

func init() {
	importCmd.AddCommand(importCurlCmd, importPostmanCmd, importHTTPFileCmd)
	exportCmd.AddCommand(exportCurlCmd, exportPostmanCmd)
	rootCmd.AddCommand(importCmd, exportCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfgManager := config.NewManager()
	if err := cfgManager.Load(findProjectConfig(), defaultUserConfig()); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
	}

	store, err := storage.NewStore(dataDir())
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	cfg := cfgManager.Get()
	httpClient := transport.NewClient(
		cfg.RequestDefaults.TimeoutSeconds,
		cfg.RequestDefaults.FollowRedirects,
		cfg.RequestDefaults.VerifySSL,
	)

	app := ui.NewApp(cfg, store, httpClient, transport.ParseCurlCommand)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func runImportCurl(cmd *cobra.Command, args []string) error {
	req, err := transport.ParseCurlCommand(args[0])
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	store, err := storage.NewStore(dataDir())
	if err != nil {
		return err
	}
	if err := store.SaveRequest(context.Background(), req); err != nil {
		return err
	}
	fmt.Printf("Saved request: %s\n", req.Name)
	return nil
}

func runImportPostman(cmd *cobra.Command, args []string) error {
	reqs, col, err := parser.ParsePostmanCollection(args[0])
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	store, err := storage.NewStore(dataDir())
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, r := range reqs {
		if err := store.SaveRequest(ctx, r); err != nil {
			return err
		}
	}
	if col != nil {
		if err := store.SaveCollection(ctx, col); err != nil {
			return err
		}
	}
	fmt.Printf("Imported %d requests\n", len(reqs))
	return nil
}

func runImportHTTPFile(cmd *cobra.Command, args []string) error {
	reqs, err := parser.ParseHTTPFile(args[0])
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	store, err := storage.NewStore(dataDir())
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, r := range reqs {
		if err := store.SaveRequest(ctx, r); err != nil {
			return err
		}
	}
	fmt.Printf("Imported %d requests\n", len(reqs))
	return nil
}

func runExportCurl(cmd *cobra.Command, args []string) error {
	store, err := storage.NewStore(dataDir())
	if err != nil {
		return err
	}
	reqs, err := store.ListRequests(context.Background())
	if err != nil {
		return err
	}
	for _, r := range reqs {
		if r.Name == args[0] {
			fmt.Println(exporter.ToCurl(r))
			return nil
		}
	}
	return fmt.Errorf("request %q not found", args[0])
}

func runExportPostman(cmd *cobra.Command, args []string) error {
	store, err := storage.NewStore(dataDir())
	if err != nil {
		return err
	}
	reqs, err := store.ListRequests(context.Background())
	if err != nil {
		return err
	}
	data, err := exporter.ToPostmanCollection("http-cli", reqs)
	if err != nil {
		return err
	}
	if err := os.WriteFile(args[0], data, 0o644); err != nil {
		return err
	}
	fmt.Printf("Exported %d requests to %s\n", len(reqs), args[0])
	return nil
}

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".data"
	}
	return filepath.Join(home, ".local", "share", "http-cli")
}

func defaultUserConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "http-cli", "config.json")
}

func findProjectConfig() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "configs", "config.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	candidate := filepath.Join("configs", "config.json")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}
