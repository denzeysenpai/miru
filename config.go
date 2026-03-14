package miru

// FolderBy defines how log files are partitioned by time.
type FolderBy int

const (
	// FolderNone means logs are not partitioned into date folders.
	FolderNone FolderBy = iota
	// FolderMonth partitions logs by month (e.g. 2025-03).
	FolderMonth
	// FolderYear partitions logs by year (e.g. 2025).
	FolderYear
)

// Convenience aliases for FolderBy.
const (
	Month = FolderMonth
	Year  = FolderYear
)

// DebugConfig holds configuration for the debugger.
type DebugConfig struct {
	// OutputPath is the directory for log files. Default: "./Debug Output"
	OutputPath string
	// FolderBy controls partitioning: FolderMonth, FolderYear, or FolderNone (default).
	FolderBy FolderBy
	// Colorful enables colored console output. Default: false (plain).
	Colorful bool
	// WithContext includes function name, file, and line in output. Default: true.
	WithContext bool
	// IncludeTests when true writes debug.Test results to the log file. Default: false.
	IncludeTests bool
}

// DefaultConfig returns a config with all defaults applied.
// OutputPath: "./Debug Output", FolderBy: FolderNone, Colorful: false, WithContext: true, IncludeTests: false.
func DefaultConfig() DebugConfig {
	return DebugConfig{
		OutputPath:   "./Debug Output",
		FolderBy:     FolderNone,
		Colorful:     false,
		WithContext:  true,
		IncludeTests: false,
	}
}

// setDefaults applies default values to unset fields (used when applying user config).
func (c *DebugConfig) setDefaults() {
	if c.OutputPath == "" {
		c.OutputPath = "./Debug Output"
	}
}
