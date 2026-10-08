// Package version holds build metadata injected by the release workflow.
package version

// Version defaults to dev; release builds set it with -ldflags -X.
var Version = "dev"
