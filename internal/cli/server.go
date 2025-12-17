package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"ldapmerge/internal/api"
	"ldapmerge/internal/repository"
)

var (
	serverHost string
	serverPort int
	dbPath     string
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long: `Start an HTTP API server that exposes endpoints for merge operations.

Endpoints:
  POST /api/merge      - Merge initial and response JSON data
  GET  /api/health     - Health check endpoint
  GET  /api/history    - List merge history
  GET  /api/history/:id - Get specific history entry
  GET  /api/configs    - List NSX configurations
  POST /api/configs    - Create NSX configuration
  GET  /api/configs/:id - Get specific configuration
  DELETE /api/configs/:id - Delete configuration

Documentation:
  GET  /docs           - Scalar API documentation`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "server host address")
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "server port")
	serverCmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite database (default: $HOME/.ldapmerge/data.db)")

	_ = viper.BindPFlag("server.host", serverCmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("server.port", serverCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("server.db", serverCmd.Flags().Lookup("db"))
}

func getDBPath() string {
	if dbPath != "" {
		return dbPath
	}

	if p := viper.GetString("server.db"); p != "" {
		return p
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "ldapmerge.db"
	}

	dataDir := filepath.Join(home, ".ldapmerge")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "ldapmerge.db"
	}

	return filepath.Join(dataDir, "data.db")
}

func runServer(cmd *cobra.Command, args []string) error {
	addr := fmt.Sprintf("%s:%d", serverHost, serverPort)

	dbFile := getDBPath()
	fmt.Printf("Using database: %s\n", dbFile)

	repo, err := repository.New(dbFile)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer repo.Close()

	srv := api.NewServer(addr, repo)

	fmt.Printf("Starting API server on %s\n", addr)
	fmt.Printf("API documentation available at http://%s/docs\n", addr)
	return srv.Start()
}
