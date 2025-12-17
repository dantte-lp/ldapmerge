package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"ldapmerge/internal/merger"
	"ldapmerge/internal/nsx"
)

var (
	nsxHost     string
	nsxUsername string
	nsxPassword string
	nsxInsecure bool
	nsxTimeout  int
)

// nsxCmd represents the nsx command group
var nsxCmd = &cobra.Command{
	Use:   "nsx",
	Short: "NSX API operations",
	Long: `Commands for interacting with VMware NSX LDAP identity sources.

Available operations:
  pull       - Fetch all LDAP identity sources
  push       - Update LDAP identity sources from file
  get        - Get specific LDAP identity source
  delete     - Delete LDAP identity source
  probe      - Test LDAP server connection
  fetch-cert - Fetch SSL certificate from LDAP server
  search     - Search users/groups in LDAP identity source`,
}

// nsxPullCmd pulls LDAP identity sources from NSX
var nsxPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull LDAP identity sources from NSX",
	Long: `Fetch all LDAP identity sources from NSX Manager.
Outputs JSON that can be used as initial input for merge operation.`,
	RunE: runNSXPull,
}

// nsxPushCmd pushes merged config to NSX
var nsxPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push LDAP identity sources to NSX",
	Long: `Push merged LDAP configuration to NSX Manager.
Takes a JSON file (output from merge command) and updates NSX.`,
	RunE: runNSXPush,
}

// nsxGetCmd gets a specific LDAP identity source
var nsxGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific LDAP identity source",
	Long:  `Fetch a specific LDAP identity source by ID from NSX Manager.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runNSXGet,
}

// nsxDeleteCmd deletes an LDAP identity source
var nsxDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete LDAP identity source",
	Long:  `Delete an LDAP identity source from NSX Manager.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runNSXDelete,
}

// nsxProbeCmd tests LDAP server connection
var nsxProbeCmd = &cobra.Command{
	Use:   "probe <id>",
	Short: "Test LDAP server connection",
	Long: `Test connection to LDAP servers for an existing identity source.
Reports success or failure for each configured LDAP server.`,
	Args: cobra.ExactArgs(1),
	RunE: runNSXProbe,
}

// nsxFetchCertCmd fetches SSL certificate from LDAP server
var nsxFetchCertCmd = &cobra.Command{
	Use:   "fetch-cert <ldap-url>",
	Short: "Fetch SSL certificate from LDAP server",
	Long: `Retrieve the SSL certificate from an LDAP server.
Example: ldapmerge nsx fetch-cert ldaps://ad01.example.com:636`,
	Args: cobra.ExactArgs(1),
	RunE: runNSXFetchCert,
}

// nsxSearchCmd searches users/groups in LDAP identity source
var nsxSearchCmd = &cobra.Command{
	Use:   "search <id> <filter>",
	Short: "Search users/groups in LDAP identity source",
	Long: `Search for users and groups in an LDAP identity source.
Example: ldapmerge nsx search example.lab "john"`,
	Args: cobra.ExactArgs(2),
	RunE: runNSXSearch,
}

func init() {
	rootCmd.AddCommand(nsxCmd)
	nsxCmd.AddCommand(nsxPullCmd)
	nsxCmd.AddCommand(nsxPushCmd)
	nsxCmd.AddCommand(nsxGetCmd)
	nsxCmd.AddCommand(nsxDeleteCmd)
	nsxCmd.AddCommand(nsxProbeCmd)
	nsxCmd.AddCommand(nsxFetchCertCmd)
	nsxCmd.AddCommand(nsxSearchCmd)

	// Common flags for all nsx subcommands
	nsxCmd.PersistentFlags().StringVar(&nsxHost, "host", "", "NSX Manager host URL (e.g., https://nsx.example.com)")
	nsxCmd.PersistentFlags().StringVarP(&nsxUsername, "username", "u", "", "NSX API username")
	nsxCmd.PersistentFlags().StringVarP(&nsxPassword, "password", "P", "", "NSX API password")
	nsxCmd.PersistentFlags().BoolVarP(&nsxInsecure, "insecure", "k", false, "Skip TLS certificate verification")
	nsxCmd.PersistentFlags().IntVar(&nsxTimeout, "timeout", 30, "API request timeout in seconds")

	_ = nsxCmd.MarkPersistentFlagRequired("host")
	_ = nsxCmd.MarkPersistentFlagRequired("username")
	_ = nsxCmd.MarkPersistentFlagRequired("password")

	// Push-specific flags
	nsxPushCmd.Flags().StringVarP(&initialFile, "file", "f", "", "path to merged JSON file (required)")
	_ = nsxPushCmd.MarkFlagRequired("file")
}

func getNSXClient() *nsx.Client {
	return nsx.NewClient(nsx.ClientConfig{
		Host:     nsxHost,
		Username: nsxUsername,
		Password: nsxPassword,
		Insecure: nsxInsecure,
		Timeout:  time.Duration(nsxTimeout) * time.Second,
	})
}

func runNSXPull(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	log := slog.With(
		"command", "nsx.pull",
		"nsx_host", nsxHost,
	)

	log.Info("starting pull operation")

	client := getNSXClient()

	result, err := client.ListLDAPIdentitySources(ctx)
	if err != nil {
		log.Error("failed to fetch LDAP identity sources", "error", err)
		return fmt.Errorf("failed to fetch LDAP identity sources: %w", err)
	}

	domains := nsx.LDAPIdentitySourcesToDomains(result.Results)

	log.Info("pull completed",
		"sources_count", len(domains),
		"duration", time.Since(startTime),
	)

	jsonData, err := json.MarshalIndent(domains, "", "    ")
	if err != nil {
		log.Error("failed to encode JSON", "error", err)
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func runNSXPush(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	log := slog.With(
		"command", "nsx.push",
		"nsx_host", nsxHost,
		"file", initialFile,
	)

	log.Info("starting push operation")

	m := merger.New()

	domains, err := m.LoadInitialFromFile(initialFile)
	if err != nil {
		log.Error("failed to load file", "error", err)
		return fmt.Errorf("failed to load file: %w", err)
	}

	client := getNSXClient()
	sources := nsx.DomainsToLDAPIdentitySources(domains)

	var successCount, errorCount int
	for _, source := range sources {
		sourceLog := log.With("source_id", source.ID)
		sourceLog.Info("updating LDAP identity source")

		fmt.Printf("Updating LDAP identity source: %s\n", source.ID)
		_, err := client.PutLDAPIdentitySource(ctx, &source)
		if err != nil {
			sourceLog.Error("failed to update source", "error", err)
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
			errorCount++
			continue
		}

		sourceLog.Info("source updated successfully")
		fmt.Printf("  OK\n")
		successCount++
	}

	log.Info("push completed",
		"success_count", successCount,
		"error_count", errorCount,
		"duration", time.Since(startTime),
	)

	return nil
}

func runNSXGet(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	id := args[0]

	log := slog.With(
		"command", "nsx.get",
		"nsx_host", nsxHost,
		"source_id", id,
	)

	log.Info("fetching LDAP identity source")

	client := getNSXClient()

	source, err := client.GetLDAPIdentitySource(ctx, id)
	if err != nil {
		log.Error("failed to fetch LDAP identity source", "error", err)
		return fmt.Errorf("failed to fetch LDAP identity source: %w", err)
	}

	domain := nsx.LDAPIdentitySourceToDomain(*source)

	log.Info("fetch completed", "duration", time.Since(startTime))

	jsonData, err := json.MarshalIndent(domain, "", "    ")
	if err != nil {
		log.Error("failed to encode JSON", "error", err)
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func runNSXDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	id := args[0]

	log := slog.With(
		"command", "nsx.delete",
		"nsx_host", nsxHost,
		"source_id", id,
	)

	log.Info("deleting LDAP identity source")

	client := getNSXClient()

	if err := client.DeleteLDAPIdentitySource(ctx, id); err != nil {
		log.Error("failed to delete LDAP identity source", "error", err)
		return fmt.Errorf("failed to delete: %w", err)
	}

	log.Info("LDAP identity source deleted successfully")
	fmt.Printf("✓ Deleted LDAP identity source: %s\n", id)
	return nil
}

func runNSXProbe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	id := args[0]

	log := slog.With(
		"command", "nsx.probe",
		"nsx_host", nsxHost,
		"source_id", id,
	)

	log.Info("probing LDAP identity source")

	client := getNSXClient()

	result, err := client.ProbeConfiguredSource(ctx, id)
	if err != nil {
		log.Error("probe failed", "error", err)
		return fmt.Errorf("probe failed: %w", err)
	}

	fmt.Printf("Probe results for %s:\n", id)
	for _, item := range result.Results {
		status := "✓"
		if !item.Success {
			status = "✗"
		}
		fmt.Printf("  %s %s", status, item.LDAPServerURL)
		if item.ErrorMessage != "" {
			fmt.Printf(" - %s", item.ErrorMessage)
		}
		fmt.Println()

		log.Info("probe result",
			"url", item.LDAPServerURL,
			"success", item.Success,
			"error", item.ErrorMessage,
		)
	}

	return nil
}

func runNSXFetchCert(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ldapURL := args[0]

	log := slog.With(
		"command", "nsx.fetch-cert",
		"nsx_host", nsxHost,
		"ldap_url", ldapURL,
	)

	log.Info("fetching certificate from LDAP server")

	client := getNSXClient()

	result, err := client.FetchCertificate(ctx, ldapURL)
	if err != nil {
		log.Error("failed to fetch certificate", "error", err)
		return fmt.Errorf("failed to fetch certificate: %w", err)
	}

	log.Info("certificate fetched successfully")

	// Print certificate details
	fmt.Printf("Certificate from %s:\n\n", ldapURL)
	if len(result.Details) > 0 {
		d := result.Details[0]
		fmt.Printf("  Subject CN:  %s\n", d.SubjectCN)
		fmt.Printf("  Subject DN:  %s\n", d.SubjectDN)
		fmt.Printf("  Issuer CN:   %s\n", d.IssuerCN)
		fmt.Printf("  Not Before:  %s\n", d.NotBefore)
		fmt.Printf("  Not After:   %s\n", d.NotAfter)
		fmt.Printf("  Algorithm:   %s\n", d.SignatureAlgorithm)
		fmt.Println()
	}

	fmt.Println("PEM Certificate:")
	fmt.Println(result.PEMEncoded)

	return nil
}

func runNSXSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	id := args[0]
	filter := args[1]

	log := slog.With(
		"command", "nsx.search",
		"nsx_host", nsxHost,
		"source_id", id,
		"filter", filter,
	)

	log.Info("searching LDAP identity source")

	client := getNSXClient()

	result, err := client.Search(ctx, id, filter)
	if err != nil {
		log.Error("search failed", "error", err)
		return fmt.Errorf("search failed: %w", err)
	}

	log.Info("search completed", "result_count", result.ResultCount)

	fmt.Printf("Search results for '%s' in %s (%d found):\n\n", filter, id, result.ResultCount)

	for _, item := range result.Results {
		typeIcon := "👤"
		if item.Type == "group" {
			typeIcon = "👥"
		}
		fmt.Printf("%s %s\n", typeIcon, item.Name)
		fmt.Printf("   DN: %s\n", item.DN)
		if item.DisplayName != "" {
			fmt.Printf("   Display Name: %s\n", item.DisplayName)
		}
		if item.Email != "" {
			fmt.Printf("   Email: %s\n", item.Email)
		}
		fmt.Println()
	}

	return nil
}
