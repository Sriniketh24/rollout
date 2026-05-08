package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

type CLIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Project string `yaml:"project"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".rollout", "config.yaml")
}

func loadConfig() (*CLIConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("not logged in — run `rollout login` first")
	}
	var cfg CLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("corrupt config: %w", err)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}
	return &cfg, nil
}

func saveConfig(cfg *CLIConfig) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o600)
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func apiRequest(cfg *CLIConfig, method, path string, body io.Reader) (*http.Response, error) {
	url := strings.TrimRight(cfg.BaseURL, "/") + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func readJSON(resp *http.Response, v any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// ---------------------------------------------------------------------------
// Shared formatting helpers
// ---------------------------------------------------------------------------

var (
	bold    = color.New(color.Bold).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	hiWhite = color.New(color.FgHiWhite).SprintFunc()
)

func enabledLabel(enabled bool) string {
	if enabled {
		return green("ON")
	}
	return red("OFF")
}

func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
}

func printHeader(w *tabwriter.Writer, headers ...string) {
	fmt.Fprintln(w, bold(strings.Join(headers, "\t")))
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, red("Error: ")+msg+"\n", args...)
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// GitOps types (rollout.yaml)
// ---------------------------------------------------------------------------

type SyncFile struct {
	Project     string     `yaml:"project"`
	Environment string     `yaml:"environment"`
	Flags       []SyncFlag `yaml:"flags"`
}

type SyncFlag struct {
	Key       string       `yaml:"key"`
	Enabled   bool         `yaml:"enabled"`
	Type      string       `yaml:"type"`
	Rollout   *SyncRollout `yaml:"rollout,omitempty"`
	Targeting []SyncTarget `yaml:"targeting,omitempty"`
}

type SyncRollout struct {
	Percentage float64 `yaml:"percentage"`
	BucketBy   string  `yaml:"bucket_by"`
}

type SyncTarget struct {
	Attribute string `yaml:"attribute"`
	Operator  string `yaml:"operator"`
	Values    []any  `yaml:"values"`
	Variation any    `yaml:"variation"`
}

// ---------------------------------------------------------------------------
// API response types
// ---------------------------------------------------------------------------

type APIFlag struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	KillSwitch  bool   `json:"kill_switch"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type APIEnvironment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Key   string `json:"key"`
	Color string `json:"color"`
}

type APIExperiment struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	FlagID         string `json:"flag_id"`
	Status         string `json:"status"`
	TrafficPercent int    `json:"traffic_percent"`
	StartedAt      string `json:"started_at,omitempty"`
}

type APIExperimentResults struct {
	ExperimentID     string               `json:"experiment_id"`
	TotalExposures   int64                `json:"total_exposures"`
	VariationResults []APIVariationResult `json:"variation_results"`
	Recommendation   string               `json:"recommendation"`
}

type APIVariationResult struct {
	VariationKey      string     `json:"variation_key"`
	Exposures         int64      `json:"exposures"`
	Conversions       int64      `json:"conversions"`
	ConversionRate    float64    `json:"conversion_rate"`
	ProbabilityToBeat float64    `json:"probability_to_beat_control"`
	ExpectedLoss      float64    `json:"expected_loss"`
	CredibleInterval  [2]float64 `json:"credible_interval_95"`
	IsControl         bool       `json:"is_control"`
}

type APIAuditEntry struct {
	ID           string `json:"id"`
	Action       string `json:"action"`
	ActorEmail   string `json:"actor_email"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Timestamp    string `json:"timestamp"`
}

type APIHealth struct {
	Service string            `json:"service"`
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Uptime  string            `json:"uptime"`
	Checks  map[string]string `json:"checks"`
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func main() {
	root := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout CLI — GitOps feature flag management",
		Long:  "A command-line tool for managing feature flags, experiments, and GitOps sync with the Rollout platform.",
	}
	root.SilenceUsage = true
	root.SilenceErrors = true

	root.AddCommand(loginCmd())
	root.AddCommand(flagsCmd())
	root.AddCommand(envsCmd())
	root.AddCommand(experimentsCmd())
	root.AddCommand(syncCmd())
	root.AddCommand(diffCmd())
	root.AddCommand(auditCmd())
	root.AddCommand(healthCmd())

	if err := root.Execute(); err != nil {
		fatal("%v", err)
	}
}

func loginCmd() *cobra.Command {
	var apiKey, baseURL, project string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Rollout API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return fmt.Errorf("--api-key is required")
			}
			cfg := &CLIConfig{APIKey: apiKey, BaseURL: baseURL, Project: project}
			if cfg.BaseURL == "" {
				cfg.BaseURL = "http://localhost:8080"
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/health", nil)
			if err != nil {
				return fmt.Errorf("cannot reach server at %s: %w", cfg.BaseURL, err)
			}
			resp.Body.Close()
			if err := saveConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("%s Logged in successfully. Config saved to %s\n", green("✓"), configPath())
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (required)")
	cmd.Flags().StringVar(&baseURL, "url", "http://localhost:8080", "Rollout server URL")
	cmd.Flags().StringVar(&project, "project", "", "Default project key")
	return cmd
}

func flagsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "flags", Short: "Manage feature flags"}
	cmd.AddCommand(flagsListCmd(), flagsGetCmd(), flagsToggleCmd(), flagsCreateCmd(), flagsKillCmd())
	return cmd
}

func flagsListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List all flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/api/v1/flags", nil)
			if err != nil {
				return err
			}
			var flags []APIFlag
			if err := readJSON(resp, &flags); err != nil {
				return err
			}
			if len(flags) == 0 {
				fmt.Println(yellow("No flags found."))
				return nil
			}
			w := newTable()
			printHeader(w, "KEY", "NAME", "TYPE", "STATUS", "KILL SWITCH")
			for _, f := range flags {
				ks := ""
				if f.KillSwitch {
					ks = red("ACTIVE")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", cyan(f.Key), f.Name, f.Type, enabledLabel(f.Enabled), ks)
			}
			w.Flush()
			fmt.Printf("\n%s %d flag(s)\n", bold("Total:"), len(flags))
			return nil
		},
	}
}

func flagsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <key>", Short: "Get flag details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/api/v1/flags/"+args[0], nil)
			if err != nil {
				return err
			}
			var flag APIFlag
			if err := readJSON(resp, &flag); err != nil {
				return err
			}
			fmt.Printf("%s %s\n", bold("Key:"), cyan(flag.Key))
			fmt.Printf("%s %s\n", bold("Name:"), flag.Name)
			fmt.Printf("%s %s\n", bold("Type:"), flag.Type)
			fmt.Printf("%s %s\n", bold("Status:"), enabledLabel(flag.Enabled))
			if flag.KillSwitch {
				fmt.Printf("%s %s\n", bold("Kill Switch:"), red("ACTIVE"))
			}
			fmt.Printf("%s %s\n", bold("Description:"), flag.Description)
			return nil
		},
	}
}

func flagsToggleCmd() *cobra.Command {
	var env string
	cmd := &cobra.Command{
		Use: "toggle <key>", Short: "Toggle a flag", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if env == "" {
				return fmt.Errorf("--env is required")
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodPost, fmt.Sprintf("/api/v1/flags/%s/toggle?env=%s", args[0], env), nil)
			if err != nil {
				return err
			}
			var result map[string]any
			if err := readJSON(resp, &result); err != nil {
				return err
			}
			enabled, _ := result["enabled"].(bool)
			fmt.Printf("%s Flag %s is now %s in %s\n", green("✓"), cyan(args[0]), enabledLabel(enabled), bold(env))
			return nil
		},
	}
	cmd.Flags().StringVar(&env, "env", "", "Target environment (required)")
	return cmd
}

func flagsCreateCmd() *cobra.Command {
	var name, flagType string
	cmd := &cobra.Command{
		Use: "create <key>", Short: "Create a new flag", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				name = args[0]
			}
			valid := map[string]bool{"boolean": true, "string": true, "number": true, "json": true}
			if !valid[flagType] {
				return fmt.Errorf("invalid type %q", flagType)
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]string{"key": args[0], "name": name, "type": flagType})
			resp, err := apiRequest(cfg, http.MethodPost, "/api/v1/flags", strings.NewReader(string(payload)))
			if err != nil {
				return err
			}
			var flag APIFlag
			if err := readJSON(resp, &flag); err != nil {
				return err
			}
			fmt.Printf("%s Flag %s created (type: %s)\n", green("✓"), cyan(flag.Key), flag.Type)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Flag name")
	cmd.Flags().StringVar(&flagType, "type", "boolean", "Flag type: boolean|string|number|json")
	return cmd
}

func flagsKillCmd() *cobra.Command {
	return &cobra.Command{
		Use: "kill <key>", Short: "Activate kill switch", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodPost, fmt.Sprintf("/api/v1/flags/%s/kill", args[0]), nil)
			if err != nil {
				return err
			}
			resp.Body.Close()
			fmt.Printf("%s Kill switch activated for %s\n", red("!"), cyan(args[0]))
			return nil
		},
	}
}

func envsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "envs", Aliases: []string{"environments"}, Short: "Manage environments"}
	cmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/api/v1/environments", nil)
			if err != nil {
				return err
			}
			var envs []APIEnvironment
			if err := readJSON(resp, &envs); err != nil {
				return err
			}
			if len(envs) == 0 {
				fmt.Println(yellow("No environments."))
				return nil
			}
			w := newTable()
			printHeader(w, "KEY", "NAME", "COLOR")
			for _, e := range envs {
				fmt.Fprintf(w, "%s\t%s\t%s\n", cyan(e.Key), e.Name, e.Color)
			}
			w.Flush()
			return nil
		},
	})
	return cmd
}

func experimentsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "experiments", Aliases: []string{"exp"}, Short: "Manage experiments"}
	cmd.AddCommand(experimentsListCmd(), experimentsResultsCmd())
	return cmd
}

func experimentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List experiments",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/api/v1/experiments", nil)
			if err != nil {
				return err
			}
			var exps []APIExperiment
			if err := readJSON(resp, &exps); err != nil {
				return err
			}
			if len(exps) == 0 {
				fmt.Println(yellow("No experiments."))
				return nil
			}
			w := newTable()
			printHeader(w, "ID", "NAME", "STATUS", "TRAFFIC", "STARTED")
			for _, e := range exps {
				st := hiWhite(e.Status)
				switch e.Status {
				case "running":
					st = green(e.Status)
				case "paused":
					st = yellow(e.Status)
				case "stopped":
					st = red(e.Status)
				case "complete":
					st = cyan(e.Status)
				}
				started := "-"
				if e.StartedAt != "" {
					if t, err := time.Parse(time.RFC3339, e.StartedAt); err == nil {
						started = t.Format("2006-01-02 15:04")
					}
				}
				id := e.ID
				if len(id) > 8 {
					id = id[:8]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d%%\t%s\n", id, e.Name, st, e.TrafficPercent, started)
			}
			w.Flush()
			return nil
		},
	}
}

func experimentsResultsCmd() *cobra.Command {
	return &cobra.Command{
		Use: "results <id>", Short: "Show experiment results", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, fmt.Sprintf("/api/v1/experiments/%s/results", args[0]), nil)
			if err != nil {
				return err
			}
			var results APIExperimentResults
			if err := readJSON(resp, &results); err != nil {
				return err
			}
			fmt.Printf("%s %s\n", bold("Experiment:"), results.ExperimentID)
			fmt.Printf("%s %d\n\n", bold("Total Exposures:"), results.TotalExposures)
			w := newTable()
			printHeader(w, "VARIATION", "EXPOSURES", "CONVERSIONS", "RATE", "P(BEAT)", "LOSS", "95% CI")
			for _, v := range results.VariationResults {
				label := v.VariationKey
				if v.IsControl {
					label += " (ctrl)"
				}
				fmt.Fprintf(w, "%s\t%d\t%d\t%.2f%%\t%.1f%%\t%.4f\t[%.4f, %.4f]\n",
					label, v.Exposures, v.Conversions, v.ConversionRate*100,
					v.ProbabilityToBeat*100, v.ExpectedLoss,
					v.CredibleInterval[0], v.CredibleInterval[1])
			}
			w.Flush()
			if results.Recommendation != "" {
				fmt.Printf("\n%s %s\n", bold("Recommendation:"), results.Recommendation)
			}
			return nil
		},
	}
}

func loadSyncFile(path string) (*SyncFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var sf SyncFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if sf.Project == "" || sf.Environment == "" {
		return nil, fmt.Errorf("rollout.yaml: project and environment are required")
	}
	return &sf, nil
}

func syncCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use: "sync", Short: "Sync flags from rollout.yaml to server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			sf, err := loadSyncFile(file)
			if err != nil {
				return err
			}
			fmt.Printf("%s Syncing %d flag(s) to %s/%s\n\n", bold("-->"), len(sf.Flags), cyan(sf.Project), cyan(sf.Environment))
			for _, flag := range sf.Flags {
				payload, _ := json.Marshal(map[string]any{
					"key": flag.Key, "enabled": flag.Enabled, "type": flag.Type,
					"rollout": flag.Rollout, "targeting": flag.Targeting, "environment": sf.Environment,
				})
				resp, err := apiRequest(cfg, http.MethodPut,
					fmt.Sprintf("/api/v1/projects/%s/flags/%s/sync", sf.Project, flag.Key),
					strings.NewReader(string(payload)))
				if err != nil {
					fmt.Printf("  %s %s — %v\n", red("x"), flag.Key, err)
					continue
				}
				resp.Body.Close()
				if resp.StatusCode >= 400 {
					fmt.Printf("  %s %s — server error %d\n", red("x"), flag.Key, resp.StatusCode)
				} else {
					fmt.Printf("  %s %s — synced\n", green("✓"), cyan(flag.Key))
				}
			}
			fmt.Printf("\n%s Sync complete.\n", green("✓"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "rollout.yaml", "Path to rollout.yaml")
	return cmd
}

func diffCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use: "diff", Short: "Preview sync changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			sf, err := loadSyncFile(file)
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/api/v1/flags", nil)
			if err != nil {
				return err
			}
			var serverFlags []APIFlag
			if err := readJSON(resp, &serverFlags); err != nil {
				return err
			}
			m := map[string]APIFlag{}
			for _, f := range serverFlags {
				m[f.Key] = f
			}
			changes := 0
			for _, local := range sf.Flags {
				remote, exists := m[local.Key]
				if !exists {
					fmt.Printf("  %s %s — will be created\n", green("+"), cyan(local.Key))
					changes++
					continue
				}
				if remote.Enabled != local.Enabled || remote.Type != local.Type {
					fmt.Printf("  %s %s — will be updated\n", yellow("~"), cyan(local.Key))
					changes++
				}
			}
			if changes == 0 {
				fmt.Printf("%s Everything in sync.\n", green("✓"))
			} else {
				fmt.Printf("\n%s %d change(s). Run %s to apply.\n", bold("Total:"), changes, cyan("rollout sync"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "rollout.yaml", "Path to rollout.yaml")
	return cmd
}

func auditCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "audit", Short: "Show recent audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, fmt.Sprintf("/api/v1/audit?limit=%d", limit), nil)
			if err != nil {
				return err
			}
			var entries []APIAuditEntry
			if err := readJSON(resp, &entries); err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println(yellow("No audit entries."))
				return nil
			}
			w := newTable()
			printHeader(w, "TIMESTAMP", "ACTION", "ACTOR", "RESOURCE", "ID")
			for _, e := range entries {
				ts := e.Timestamp
				if p, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
					ts = p.Format("2006-01-02 15:04:05")
				}
				id := e.ResourceID
				if len(id) > 8 {
					id = id[:8]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", ts, e.Action, e.ActorEmail, e.ResourceType, id)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 25, "Max entries")
	return cmd
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use: "health", Short: "Check server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			resp, err := apiRequest(cfg, http.MethodGet, "/health", nil)
			if err != nil {
				return fmt.Errorf("server unreachable: %w", err)
			}
			var h APIHealth
			if err := readJSON(resp, &h); err != nil {
				return err
			}
			icon := green("✓")
			if h.Status != "healthy" {
				icon = red("!")
			}
			fmt.Printf("%s %s %s (v%s, uptime: %s)\n", icon, bold(h.Service), h.Status, h.Version, h.Uptime)
			if len(h.Checks) > 0 {
				w := newTable()
				printHeader(w, "CHECK", "STATUS")
				for name, status := range h.Checks {
					fmt.Fprintf(w, "%s\t%s\n", name, status)
				}
				w.Flush()
			}
			return nil
		},
	}
}
