package version

import (
	"fmt"
	"runtime"
)

// Build information - set via ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf("ldapmerge %s", Version)
}

// Full returns full version information.
func Full() string {
	return fmt.Sprintf(`ldapmerge %s
  Commit:     %s
  Build Date: %s
  Go Version: %s
  OS/Arch:    %s/%s`,
		Version, Commit, BuildDate, GoVersion, runtime.GOOS, runtime.GOARCH)
}

// Short returns short version string.
func Short() string {
	return Version
}
