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
	"ldapmerge/internal/models"
	"ldapmerge/internal/nsx"
)

var (
	// sync-specific flags
	syncResponseFile string
	syncOutputFile   string
	syncDryRun       bool
)

// syncCmd represents the sync command - full pipeline
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Full sync: pull from NSX, merge with certificates, push back",
	Long: `Execute the complete synchronization pipeline:

1. PULL  - Fetch current LDAP identity sources from NSX Manager
2. MERGE - Combine with certificate response data (from Ansible)
3. PUSH  - Update NSX Manager with merged configuration

This command performs all three steps in sequence with a single invocation.`,
	Example: `  # Basic usage
  ldapmerge sync \
    --host https://nsx.example.com \
    -u admin -P secret \
    -r certificates_response.json

  # With output file and dry-run
  ldapmerge sync \
    --host https://nsx.example.com \
    -u admin -P secret \
    -r certificates_response.json \
    -o merged_result.json \
    --dry-run

  # Skip TLS verification
  ldapmerge sync \
    --host https://nsx.example.com \
    -u admin -P secret -k \
    -r certificates_response.json`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// NSX connection flags (same as nsx command)
	syncCmd.Flags().StringVar(&nsxHost, "host", "", "NSX Manager host URL (required)")
	syncCmd.Flags().StringVarP(&nsxUsername, "username", "u", "", "NSX API username (required)")
	syncCmd.Flags().StringVarP(&nsxPassword, "password", "P", "", "NSX API password (required)")
	syncCmd.Flags().BoolVarP(&nsxInsecure, "insecure", "k", false, "Skip TLS certificate verification")
	syncCmd.Flags().IntVar(&nsxTimeout, "timeout", 30, "API request timeout in seconds")

	// Sync-specific flags
	syncCmd.Flags().StringVarP(&syncResponseFile, "response", "r", "", "Path to certificate response JSON file (required)")
	syncCmd.Flags().StringVarP(&syncOutputFile, "output", "o", "", "Save merged result to file (optional)")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Perform pull and merge, but skip push to NSX")

	_ = syncCmd.MarkFlagRequired("host")
	_ = syncCmd.MarkFlagRequired("username")
	_ = syncCmd.MarkFlagRequired("password")
	_ = syncCmd.MarkFlagRequired("response")
}

func runSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	log := slog.With(
		"command", "sync",
		"nsx_host", nsxHost,
		"dry_run", syncDryRun,
	)

	log.Info("starting sync operation")

	// Step 1: PULL from NSX
	log.Info("step 1/3: pulling LDAP identity sources from NSX")
	fmt.Println("► Step 1/3: Pulling current configuration from NSX...")

	client := nsx.NewClient(nsx.ClientConfig{
		Host:     nsxHost,
		Username: nsxUsername,
		Password: nsxPassword,
		Insecure: nsxInsecure,
		Timeout:  time.Duration(nsxTimeout) * time.Second,
	})

	pullStart := time.Now()
	result, err := client.ListLDAPIdentitySources(ctx)
	if err != nil {
		log.Error("failed to pull from NSX", "error", err, "duration", time.Since(pullStart))
		return fmt.Errorf("pull failed: %w", err)
	}

	initial := nsx.LDAPIdentitySourcesToDomains(result.Results)
	log.Info("pull completed",
		"sources_count", len(initial),
		"duration", time.Since(pullStart),
	)
	fmt.Printf("  ✓ Fetched %d LDAP identity sources\n", len(initial))

	// Step 2: MERGE with certificates
	log.Info("step 2/3: merging with certificate response",
		"response_file", syncResponseFile,
	)
	fmt.Println("► Step 2/3: Merging with certificate data...")

	mergeStart := time.Now()
	m := merger.New()

	response, err := m.LoadResponseFromFile(syncResponseFile)
	if err != nil {
		log.Error("failed to load response file", "error", err, "file", syncResponseFile)
		return fmt.Errorf("failed to load response file: %w", err)
	}

	merged := m.Merge(initial, response)

	// Count certificates added
	certsAdded := countCertificates(merged)
	log.Info("merge completed",
		"domains_count", len(merged),
		"certificates_added", certsAdded,
		"duration", time.Since(mergeStart),
	)
	fmt.Printf("  ✓ Merged %d domains, %d certificates added\n", len(merged), certsAdded)

	// Save output file if requested
	if syncOutputFile != "" {
		if err := saveResultToFile(merged, syncOutputFile); err != nil {
			log.Error("failed to save output file", "error", err, "file", syncOutputFile)
			return fmt.Errorf("failed to save output: %w", err)
		}
		log.Info("saved merged result to file", "file", syncOutputFile)
		fmt.Printf("  ✓ Saved result to %s\n", syncOutputFile)
	}

	// Step 3: PUSH to NSX (unless dry-run)
	if syncDryRun {
		log.Info("dry-run mode, skipping push to NSX")
		fmt.Println("► Step 3/3: Skipped (dry-run mode)")
		fmt.Println("\n✓ Sync completed (dry-run)")
	} else {
		log.Info("step 3/3: pushing merged configuration to NSX")
		fmt.Println("► Step 3/3: Pushing configuration to NSX...")

		pushStart := time.Now()
		sources := nsx.DomainsToLDAPIdentitySources(merged)

		var successCount, errorCount int
		for _, source := range sources {
			sourceLog := log.With("source_id", source.ID)
			sourceLog.Info("updating LDAP identity source")

			_, err := client.PutLDAPIdentitySource(ctx, &source)
			if err != nil {
				sourceLog.Error("failed to update source", "error", err)
				fmt.Printf("  ✗ %s: %v\n", source.ID, err)
				errorCount++
				continue
			}

			sourceLog.Info("source updated successfully")
			fmt.Printf("  ✓ %s\n", source.ID)
			successCount++
		}

		log.Info("push completed",
			"success_count", successCount,
			"error_count", errorCount,
			"duration", time.Since(pushStart),
		)

		if errorCount > 0 {
			fmt.Printf("\n⚠ Sync completed with errors: %d succeeded, %d failed\n", successCount, errorCount)
		} else {
			fmt.Println("\n✓ Sync completed successfully")
		}
	}

	log.Info("sync operation finished",
		"total_duration", time.Since(startTime),
	)

	return nil
}

func countCertificates(domains []models.Domain) int {
	count := 0
	for _, d := range domains {
		for _, s := range d.LDAPServers {
			count += len(s.Certificates)
		}
	}
	return count
}

func saveResultToFile(domains []models.Domain, path string) error {
	data, err := json.MarshalIndent(domains, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
