# miru

A Go debugger toolkit library for developers: structured panic recovery, logging, testing, and tracing with useful context (function, file, line).

## Contents

- [Install](#install)
- [Quick start](#quick-start)
- [Config](#config)
- [Pretty-print: `Walk`](#pretty-print-walk)
- [Panic recovery: `Catch`](#panic-recovery-catch)
- [Console logging: `Out`](#console-logging-out)
- [Tap: `Tap`](#tap-tap)
- [Testing: `Test`](#testing-test)
- [Test Groups: `TestGroup`](#test-groups-testgroup)
- [Remote dashboard: `RemoteDashboard`](#remote-dashboard-remotedashboard)
- [Stack trace: `CheckStack`](#stack-trace-checkstack)
- [Tracing: `Trace`](#tracing-trace)
- [Memory Statistics: `Mem`](#memory-statistics-mem)
- [Error Flow: `IfErr`](#error-flow-iferr)
- [License](#license)

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
	
	// Console logging
	debug.Out("Processing user data", userID)
	
	// Memory statistics
	debug.Mem()
	
	// Group related tests
	tg := debug.TestGroup("User Operations")
	defer tg.Close()
	tg.Test("user creation", createUser("test@example.com") != nil)
	tg.Test("user validation", validateEmail("test@example.com"))
	
	// Trace execution time
	defer debug.Trace("database operation")()
	// ... your code ...
}
```

## Using Default: `NewDebugger`

You can also skip the config and just stick to the defaults.

```
func TestBasics() {
	debug := miru.NewDebugger()

	debug.Out("Hello There!")
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

## Test Groups: `TestGroup`

Group related tests together and get a summary of passed/failed tests:

```go
tg := debug.TestGroup("User Authentication")
defer tg.Close()

tg.Test("valid login", authenticate("user", "pass") == nil)
tg.Test("invalid password", authenticate("user", "wrong") != nil)
tg.Test("empty username", authenticate("", "pass") != nil)
```

Output:

```
[Miru TGroup Start]:	<dateTime>	User Authentication
[0]	<dateTime>	valid login	->	PASSED
[1]	<dateTime>	invalid password	->	PASSED
[2]	<dateTime>	empty username	->	PASSED
[Miru TGroup Close]:	<dateTime>	(3 / 3)
```

- Green: `[Miru TGroup Start]` and `[Miru TGroup Close]` headers
- PASSED tests are green, FAILED tests are red
- Final summary shows (passed / total) count

## Remote dashboard: `RemoteDashboard`

Serve a small web UI that shows logs and traces live (SSE). Call once to start the server; all Catch, Out, Test, Trace, Walk, CheckStack, and Mem output is streamed to the page.

```go
srv := debug.RemoteDashboard(8765) // port 0 or negative = 8765
// open http://localhost:8765
// when done: srv.Shutdown(ctx)
```

No log file writing from the dashboard; it only streams what's already printed to the console.

### Dashboard Features

The web dashboard provides a live view of your application's debug output with the following features:

**Live Log Streaming**
- Real-time updates via Server-Sent Events (SSE)
- All log types displayed: Catch, Out, Test, Trace, Walk, CheckStack, Mem, Tap, Error
- Timestamps and source context for each entry

**Log Type Filtering**
- Clickable count cards for each log type
- Shows live counts: All, Errors, Logs, Tests, Traces, Walk, Stack, Memory, Tap, IfErr
- Color-coded entries matching the console output
- Active filter highlighted in blue

**Search**
- Real-time search across all log entries
- Searches in timestamps, log types, and message bodies
- Matching text highlighted in yellow
- Press Ctrl+F to focus search, Esc to clear

**Auto-Scroll Toggle**
- Enable/disable automatic scrolling to latest logs
- Visual indicator shows active state
- Press Ctrl+S to toggle

**Export Logs**
- Download filtered logs as timestamped text file
- Exports only currently visible entries
- Press Ctrl+E to export

**Keyboard Shortcuts**
- Ctrl+F - Focus search box
- Ctrl+S - Toggle auto-scroll
- Ctrl+E - Export logs
- Ctrl+C - Clear logs
- Esc - Clear search
- ? - Toggle shortcuts help panel

### Usage Example

```go
func main() {
    debug := miru.NewDebugger()
    
    // Start dashboard on port 8765
    srv := debug.RemoteDashboard(8765)
    defer srv.Shutdown(context.Background())
    
    // Your application code here
    // All debug output will appear in the dashboard
    debug.Out("Application started")
    
    // Keep running or use a wait group
    select {}
}
```

Access the dashboard at `http://localhost:8765` (or your chosen port).

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

## Memory Statistics: `Mem`

Display runtime memory statistics including allocation, heap usage, system memory, goroutine count, and GC cycles:

```go
debug.Mem()
```

Output:

```
[Miru Mem]:	<dateTime>	memory	->	alloc=12MB heap=8MB sys=45MB goroutines=3 gc=15
```

- Green: `[Miru Mem]`
- Yellow: dateTime
- Shows alloc (current allocation), heap (heap allocation), sys (system memory), goroutines (active goroutines), and gc (GC cycles)

## Error Flow: `IfErr`

`IfErr` provides a **fluent error-handling helper** that lets you react to errors without repetitive `if err != nil` blocks.

It supports chaining actions like:

* `Do()` – run code when an error exists
* `Else()` – run code when no error exists
* `Panic()` – panic if an error exists
* `Retry()` – retry an operation multiple times

`IfErr` also **logs the error automatically** when it is not `nil`.

### Usage

```go
err := doSomething()

debug.IfErr(err).
    Do(func() {
        debug.Out("operation failed")
    }).
    Else(func() {
        debug.Out("operation succeeded")
    })
```

### Running Code When an Error Exists: `Do`

`Do` executes a function  **only if an error occurred**.

```go
debug.IfErr(err).Do(func() {
    debug.Out("error occurred")
})
```

### Handling Success: `Else`

`Else` executes a function  **only if no error occurred**.

```go
debug.IfErr(err).
    Do(func() {
        debug.Out("failed")
    }).
    Else(func() {
        debug.Out("success")
    })
```

### Panic on Error: `Panic`

`Panic` panics if an error exists.

```go
debug.IfErr(err).Panic()
```

Equivalent to:

```go
if err != nil {
    panic(err)
}
```

### Retrying Operations: `Retry`

`Retry` repeatedly executes a function until it succeeds or the retry limit is reached.

```go
err = debug.IfErr(err).Retry(3, func() error {
    return reconnect()
})
```

#### Behavior

* Only runs if the original error is **not nil**
* Stops retrying once the function returns `nil`
* Logs each failed retry attempt

#### Output Sample

```
[Miru Err]: <dateTime> main:32 -> connection failed
[Miru Out]: retry 1/3 failed: connection refused
[Miru Out]: retry 2/3 failed: connection refused
```

### Summary

`IfErr` enables concise and readable error handling:

```go
debug.IfErr(err).Do(...)
debug.IfErr(err).Do(...).Else(...)
debug.IfErr(err).Panic()
debug.IfErr(err).Retry(3, reconnect)
```
## Test Groups: `TestGroup`

Allows users to organize their tests when running their application, useful for testing environment or config variables you need to set up before the application runs. `TestGroup` returns an object used for adding tests.

### Implementation

```Go

var debug Debugger

func TestMyVars(dbUser *string, dbPass *string) {
	dtg := debug.TestGroup("Environment Variables Test")

	dtg.Test("Is username imported", dbUser != nil)
	dtg.Test("Is password imported", dbPass != nil)

	dtg.Close() // this will close the test group and will no longer allow other tests

	dtg.Test("Some other test", false) // panic, once it is closed it cannot be used
}

```

### Test Group: `Test`

Handles the provided test, takes an argument string for its label and the condition to test, useful for simple testing.

### Test Group: `Close`

Closes the test group.

## License

MIT
