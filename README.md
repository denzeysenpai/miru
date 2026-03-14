# miru

A Go debugger toolkit library for developers: structured panic recovery, logging, testing, and tracing with useful context (function, file, line).

## Install

```bash
go get github.com/denzeysenpai/miru
```

## Quick start

```go
package main

import "github.com/denzeysenpai/miru"

func main() {
	cfg := miru.DebugConfig{
		OutputPath:   "./Debug Output",  // default
		FolderBy:     miru.Month,        // or miru.Year, or miru.FolderNone
		Colorful:     true,              // colored console output
		WithContext:  true,              // include function:line in output
		IncludeTests: false,             // when true, Test() also logs to file
	}
	debug := miru.NewDebugger()
	debug.Config(cfg)

	SomeFunction(debug)
}

func SomeFunction(debug *miru.Debugger) {
	debug.Func("SomeFunction")
	defer func() {
		if r := recover(); r != nil {
			debug.Catch(r)
		}
	}()
	// ... your code ...
}
```

## Config

| Field        | Type     | Default              | Description                                                   |
| ------------ | -------- | -------------------- | ------------------------------------------------------------- |
| OutputPath   | string   | `"./Debug Output"` | Directory for log files                                       |
| FolderBy     | FolderBy | FolderNone           | `miru.Month`, `miru.Year`, or `miru.FolderNone`         |
| Colorful     | bool     | false                | Colored console output                                        |
| WithContext  | bool     | true*                | Include function name and line number                         |
| IncludeTests | bool     | false                | When true,`Test()` results are also written to the log file |
| WalkDepth    | int      | 5                    | Max depth for `Walk` pretty-print; -1 = no limit            |

\* Use `miru.DefaultConfig()` to get a config with all defaults (including `WithContext: true`).

## Pretty-print: `Walk`

Inspect structs, slices, and maps with indented output. Depth is limited by `WalkDepth` in config (-1 = no limit).

```go
type User struct{ Name string; Age int }
debug.Walk([]User{{"Alice", 30}, {"Bob", 25}})
debug.Walk(myMap)
```

Output (first line uses same style as other Miru logs; rest is indented):

```
[Miru Walk]:	<dateTime>	main:42	->	slice (len 2)
  [0]:
    Name: Alice
    Age: 30
  [1]:
    Name: Bob
    Age: 25
```

## Panic recovery: `Catch`

`debug.Catch(r)` logs the recovered panic to the **console** and to the **log file**:

```
[Miru Catch]:	<dateTime>	SomeFunction:42	->	Caught: runtime error: ...
```

- Red: `[Miru Catch]` and the caught message (when Colorful is true)
- Yellow: dateTime

## Console logging: `Out`

Like `console.log` in JavaScript: any number of arguments, any types. **Never** writes to log files.

```go
debug.Out("Hi I'm Mr. Meseeks!", 10, jsonData)
```

Output (one line per argument):

```
[Miru Out]:	<dateTime>	SomeFunction:line	->	Hi I'm Mr. Meseeks!
[Miru Out]:	<dateTime>	SomeFunction:line	->	10
[Miru Out]:	<dateTime>	SomeFunction:line	->	{"key":"value"}
```

- Red: `[Miru Out]`
- Yellow: dateTime

Structs, maps, and slices are serialized as JSON.

## Tap: `Tap`

Pass a value through a function (e.g. to log it) and get the same value back. Like Ruby’s `tap`. This way, you can log and get the value in the same line.

```go
x := debug.Tap(compute(), func(v interface{}) { debug.Out(v) })
// x is the result of compute(); you also logged it
```

## Testing: `Test`

Run a function and compare its return value to the expected value. Works with or without arguments:

```go
// no args
debug.Test("add", func() int { return 2 + 2 }, 4)

// with args: funcName, fn, expected, then args to pass to fn
debug.Test("add", func(a, b int) int { return a + b }, 7, 3, 4)
debug.Test("fail", func() int { return 1 }, 2)
```

Output:

```
[Miru Test]:	<dateTime>	add	->	PASSED	(0.20ms)
[Miru Test]:	<dateTime>	fail	->	FAILED	(0.22ms)
```

- Green: `[Miru Test]`; PASSED is green, FAILED is red
- Yellow: dateTime and duration

## Stack trace: `CheckStack`

Print the current goroutine's stack trace (console only, no file):

```go
debug.CheckStack()
```

Output: `[Miru CheckStack]` header plus indented stack lines. With `Colorful`, the goroutine line is yellow.

## Tracing: `Trace`

Measure execution time with a deferred call:

```go
defer debug.Trace("someFunc")()
```

Output:

```
[Miru Trace]:	<dateTime>	someFunc	->	0.25ms
```

- Green: `[Miru Trace]`
- Yellow: dateTime and duration

## License

MIT
