package cli

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"ldapmerge/internal/merger"
)

var (
	initialFile  string
	responseFile string
	outputFile   string
	compact      bool
)

// mergeCmd represents the merge command
var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge initial config with certificate response",
	Long: `Merge LDAP server configurations with certificate data.

Takes an initial JSON file containing domain and LDAP server configurations,
and a response JSON file containing certificate information.
Outputs merged JSON with certificates added to matching LDAP servers.`,
	RunE: runMerge,
}

func init() {
	rootCmd.AddCommand(mergeCmd)

	mergeCmd.Flags().StringVarP(&initialFile, "initial", "i", "", "path to initial JSON file (required)")
	mergeCmd.Flags().StringVarP(&responseFile, "response", "r", "", "path to response JSON file (required)")
	mergeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "path to output file (default: stdout)")
	mergeCmd.Flags().BoolVarP(&compact, "compact", "c", false, "output compact JSON (no indentation)")

	mergeCmd.MarkFlagRequired("initial")
	mergeCmd.MarkFlagRequired("response")
}

func runMerge(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	log := slog.With(
		"command", "merge",
		"initial_file", initialFile,
		"response_file", responseFile,
	)

	log.Info("starting merge operation")

	m := merger.New()

	result, err := m.MergeFromFiles(initialFile, responseFile)
	if err != nil {
		log.Error("merge failed", "error", err)
		return fmt.Errorf("merge failed: %w", err)
	}

	log.Info("merge completed",
		"domains_count", len(result),
		"duration", time.Since(startTime),
	)

	jsonData, err := m.ToJSON(result, !compact)
	if err != nil {
		log.Error("failed to encode JSON", "error", err)
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			log.Error("failed to write output file", "error", err, "file", outputFile)
			return fmt.Errorf("failed to write output file: %w", err)
		}
		log.Info("output written to file", "file", outputFile, "size_bytes", len(jsonData))
		fmt.Fprintf(os.Stderr, "Output written to %s\n", outputFile)
	} else {
		fmt.Println(string(jsonData))
	}

	log.Info("merge operation finished", "total_duration", time.Since(startTime))

	return nil
}
