package miru_test

import (
	"github.com/denzeysenpai/miru"
)

func Example_basic() {
	cfg := miru.DebugConfig{
		OutputPath:   "./Debug Output",
		FolderBy:     miru.FolderNone,
		Colorful:     true,
		WithContext:  true,
		IncludeTests: false,
	}
	debug := miru.NewDebugger()
	debug.Config(cfg)

	debug.Func("Example_basic")
	defer func() {
		if r := recover(); r != nil {
			debug.Catch(r)
		}
	}()

	debug.Out("Hi I'm Mr. Meseeks!", 10)
}

func Example_test() {
	debug := miru.NewDebugger()
	debug.Config(miru.DebugConfig{Colorful: true})

	debug.Test("add", func() int { return 2 + 2 }, 4)
	debug.Test("fail", func() int { return 1 }, 2)
}

func Example_trace() {
	debug := miru.NewDebugger()
	debug.Config(miru.DebugConfig{Colorful: true})

	debug.Func("Example_trace")
	defer debug.Trace("Example_trace")()
	// ... do work ...
}
