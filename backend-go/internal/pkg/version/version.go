// Package version 保存构建信息，由 ldflags 注入。
package version

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)
