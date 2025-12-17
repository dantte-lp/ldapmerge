package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"ldapmerge/internal/logging"
	"ldapmerge/internal/version"
)

var (
	cfgFile    string
	logDir     string
	logLevel   string
	logConsole bool
)

// Color definitions
var (
	titleStyle   = color.New(color.FgHiCyan, color.Bold)
	headerStyle  = color.New(color.FgHiYellow, color.Bold)
	cmdStyle     = color.New(color.FgHiGreen)
	descStyle    = color.New(color.FgWhite)
	iconStyle    = color.New(color.FgHiBlue)
	versionStyle = color.New(color.FgHiWhite, color.Faint)
)

const banner = `
  ██╗     ██████╗  █████╗ ██████╗ ███╗   ███╗███████╗██████╗  ██████╗ ███████╗
  ██║     ██╔══██╗██╔══██╗██╔══██╗████╗ ████║██╔════╝██╔══██╗██╔════╝ ██╔════╝
  ██║     ██║  ██║███████║██████╔╝██╔████╔██║█████╗  ██████╔╝██║  ███╗█████╗
  ██║     ██║  ██║██╔══██║██╔═══╝ ██║╚██╔╝██║██╔══╝  ██╔══██╗██║   ██║██╔══╝
  ███████╗██████╔╝██║  ██║██║     ██║ ╚═╝ ██║███████╗██║  ██║╚██████╔╝███████╗
  ╚══════╝╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝     ╚═╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝
`

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "ldapmerge",
	Short: "🔄 LDAP configuration merger for VMware NSX",
	Long:  getLongDescription(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip logging init for version and help
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}
		return initLogging(cmd, args)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logging.Close()
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "📋 Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		titleStyle.Print(banner)
		fmt.Println(version.Full())
	},
}

func getLongDescription() string {
	var sb strings.Builder

	titleStyle.Fprint(&sb, banner)
	sb.WriteString("\n")

	versionStyle.Fprintf(&sb, "  Version: %s\n\n", version.Short())

	descStyle.Fprint(&sb, "  LDAP configuration merger tool for VMware NSX 4.2.\n")
	descStyle.Fprint(&sb, "  Merges LDAP server configurations with SSL certificates.\n\n")

	headerStyle.Fprint(&sb, "  ⚡ WORKFLOW\n")
	sb.WriteString("\n")
	sb.WriteString("    ")
	iconStyle.Fprint(&sb, "1. ")
	cmdStyle.Fprint(&sb, "sync")
	descStyle.Fprint(&sb, "    → Full pipeline: pull → merge → push\n")
	sb.WriteString("    ")
	iconStyle.Fprint(&sb, "2. ")
	cmdStyle.Fprint(&sb, "merge")
	descStyle.Fprint(&sb, "   → Merge JSON files locally\n")
	sb.WriteString("    ")
	iconStyle.Fprint(&sb, "3. ")
	cmdStyle.Fprint(&sb, "nsx")
	descStyle.Fprint(&sb, "     → Direct NSX API operations\n")
	sb.WriteString("    ")
	iconStyle.Fprint(&sb, "4. ")
	cmdStyle.Fprint(&sb, "server")
	descStyle.Fprint(&sb, "  → Start REST API server\n")
	sb.WriteString("\n")

	headerStyle.Fprint(&sb, "  📡 NSX OPERATIONS\n")
	sb.WriteString("\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx pull")
	descStyle.Fprint(&sb, "        📥 Fetch LDAP identity sources\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx push")
	descStyle.Fprint(&sb, "        📤 Update LDAP identity sources\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx get")
	descStyle.Fprint(&sb, "         🔍 Get specific source\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx delete")
	descStyle.Fprint(&sb, "      🗑️  Delete source\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx probe")
	descStyle.Fprint(&sb, "       🩺 Test LDAP connection\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx fetch-cert")
	descStyle.Fprint(&sb, "  🔐 Fetch SSL certificate\n")
	sb.WriteString("    ")
	cmdStyle.Fprint(&sb, "nsx search")
	descStyle.Fprint(&sb, "      🔎 Search users/groups\n")
	sb.WriteString("\n")

	headerStyle.Fprint(&sb, "  📚 EXAMPLES\n")
	sb.WriteString("\n")
	descStyle.Fprint(&sb, "    # Full sync with NSX\n")
	cmdStyle.Fprint(&sb, "    $ ldapmerge sync --host https://nsx.example.com -u admin -P secret -r certs.json\n")
	sb.WriteString("\n")
	descStyle.Fprint(&sb, "    # Local merge\n")
	cmdStyle.Fprint(&sb, "    $ ldapmerge merge -i initial.json -r response.json -o result.json\n")
	sb.WriteString("\n")
	descStyle.Fprint(&sb, "    # Start API server\n")
	cmdStyle.Fprint(&sb, "    $ ldapmerge server -p 8080\n")
	sb.WriteString("\n")

	headerStyle.Fprint(&sb, "  🔗 LINKS\n")
	sb.WriteString("\n")
	descStyle.Fprint(&sb, "    Documentation: ")
	cmdStyle.Fprint(&sb, "http://localhost:8080/docs\n")
	descStyle.Fprint(&sb, "    NSX API Docs:   ")
	cmdStyle.Fprint(&sb, "https://developer.broadcom.com/xapis/nsx-t-data-center-rest-api/4.2/\n")

	return sb.String()
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("✗ Error: %v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add version command
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.ldapmerge.yaml)")
	rootCmd.PersistentFlags().StringVar(&logDir, "log-dir", "", "log directory (default: executable directory)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")
	rootCmd.PersistentFlags().BoolVar(&logConsole, "log-console", false, "also output logs to console")

	// Bind to viper
	_ = viper.BindPFlag("logging.dir", rootCmd.PersistentFlags().Lookup("log-dir"))
	_ = viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("logging.console", rootCmd.PersistentFlags().Lookup("log-console"))

	// Customize help template
	rootCmd.SetUsageTemplate(getUsageTemplate())
}

func getUsageTemplate() string {
	return `
` + color.HiYellowString("📖 USAGE") + `
  {{.UseLine}}

{{if .HasAvailableSubCommands}}` + color.HiYellowString("📦 COMMANDS") + `
{{range .Commands}}{{if .IsAvailableCommand}}  ` + color.HiGreenString("{{rpad .Name .NamePadding}}") + ` {{.Short}}
{{end}}{{end}}{{end}}
{{if .HasAvailableLocalFlags}}` + color.HiYellowString("🚩 FLAGS") + `
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
{{if .HasAvailableInheritedFlags}}` + color.HiYellowString("🌐 GLOBAL FLAGS") + `
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
{{if .HasExample}}` + color.HiYellowString("💡 EXAMPLES") + `
{{.Example}}
{{end}}
` + color.HiWhiteString("Use \"{{.CommandPath}} [command] --help\" for more information about a command.") + `
`
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ldapmerge")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("LDAPMERGE")

	_ = viper.ReadInConfig()
}

func initLogging(cmd *cobra.Command, _ []string) error {
	// Determine log directory
	dir := viper.GetString("logging.dir")
	if dir == "" {
		exe, err := os.Executable()
		if err != nil {
			dir = "."
		} else {
			dir = filepath.Dir(exe)
		}
	}

	// Parse log level
	level := parseLogLevel(viper.GetString("logging.level"))

	cfg := logging.Config{
		LogDir:     dir,
		LogFile:    "ldapmerge.log",
		MaxSize:    100, // 100 MB
		MaxBackups: 5,
		MaxAge:     30, // 30 days
		Compress:   true,
		Level:      level,
		JSONFormat: true,
		Console:    viper.GetBool("logging.console"),
	}

	if err := logging.Init(cfg); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	slog.Info("application started",
		"command", cmd.Name(),
		"version", version.Short(),
		"log_dir", dir,
		"log_level", level.String(),
	)

	return nil
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
