package miru

// FolderBy is how we split up log dirs (by month, year, or not at all).
type FolderBy int

const (
	FolderNone  FolderBy = iota // no subfolders
	FolderMonth                 // 2025-03
	FolderYear                  // 2025
)

const (
	Month = FolderMonth
	Year  = FolderYear
)

type DebugConfig struct {
	OutputPath   string   // where to write logs (default "./Debug Output")
	FolderBy     FolderBy // Month, Year, or FolderNone
	Colorful     bool     // ANSI colors in terminal
	WithContext  bool     // add func:line to each line
	IncludeTests bool     // also write Test results to file
	WalkDepth    int      // max depth for Walk (-1 = no limit, default 5)
}

// DefaultConfig gives you the usual defaults.
func DefaultConfig() DebugConfig {
	return DebugConfig{
		OutputPath:   "./Debug Output",
		FolderBy:     FolderNone,
		Colorful:     false,
		WithContext:  true,
		IncludeTests: false,
		WalkDepth:    5,
	}
}

func (c *DebugConfig) setDefaults() {
	if c.OutputPath == "" {
		c.OutputPath = "./Debug Output"
	}
	// WalkDepth 0 is zero value; treat as "use default"
	if c.WalkDepth == 0 {
		c.WalkDepth = 5
	}
}
