// version/version.go
package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"

	GoVersion = runtime.Version()
	OS        = runtime.GOOS
	Arch      = runtime.GOARCH
)

func String() string {
	return fmt.Sprintf("v%s (%s) - commit %s - built %s - %s/%s",
		Version, GoVersion, Commit, Date, OS, Arch)
}
