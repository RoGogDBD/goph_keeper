package version

import "fmt"

var (
	// BuildVersion is the application version.
	BuildVersion = "dev"
	// BuildDate is the build timestamp.
	BuildDate = "unknown"
	// BuildCommit is the git commit hash.
	BuildCommit = "none"
)

// Info returns build info string.
func Info() string {
	return fmt.Sprintf("version=%s date=%s commit=%s", BuildVersion, BuildDate, BuildCommit)
}
