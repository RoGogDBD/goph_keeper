package version

import "fmt"

var (
	BuildVersion = "dev"
	BuildDate    = "unknown"
	BuildCommit  = "none"
)

func Info() string {
	return fmt.Sprintf("version=%s date=%s commit=%s", BuildVersion, BuildDate, BuildCommit)
}
