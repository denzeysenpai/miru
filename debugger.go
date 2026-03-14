package miru

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// Debugger provides panic recovery, logging, testing, and tracing.
type Debugger struct {
	mu          sync.Mutex
	config      DebugConfig
	writer      *writer
	currentFunc string // set by Func() for Catch location
}

// NewDebugger returns a new Debugger with default config.
func NewDebugger() *Debugger {
	cfg := DefaultConfig()
	return NewDebuggerWithConfig(cfg)
}

// NewDebuggerWithConfig returns a new Debugger with the given config.
func NewDebuggerWithConfig(cfg DebugConfig) *Debugger {
	cfg.setDefaults()
	d := &Debugger{config: cfg, writer: newWriter(cfg)}
	return d
}

// Config applies the given config (with defaults for empty fields).
func (d *Debugger) Config(cfg DebugConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	cfg.setDefaults()
	d.config = cfg
	d.writer = newWriter(cfg)
}

// Func sets the current function name for subsequent Catch/Out/Trace (used for location in output).
// Call this at the start of the function you are debugging.
func (d *Debugger) Func(name string) {
	d.mu.Lock()
	d.currentFunc = name
	d.mu.Unlock()
}

func (d *Debugger) getLocation(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "?:?"
	}
	if d.config.WithContext {
		funcName := d.currentFunc
		if funcName == "" {
			if pc, _, _, ok := runtime.Caller(skip); ok {
				funcName = filepath.Base(runtime.FuncForPC(pc).Name())
			}
		}
		if funcName != "" {
			return fmt.Sprintf("%s:%d", funcName, line)
		}
		return fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func (d *Debugger) dateTime() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}

// Catch logs a recovered panic to the console and to the log file.
// Format: [Miru Catch]: <dateTime> SomeFunction:lineNumber -> Caught: <details>
func (d *Debugger) Catch(r interface{}) {
	loc := d.getLocation(2)
	dt := d.dateTime()
	caught := fmt.Sprintf("Caught: %v", r)
	// More detailed message for errors
	if err, ok := r.(error); ok {
		caught = fmt.Sprintf("Caught: %v", err)
	}
	line := d.formatCatchLine(dt, loc, caught)
	fmt.Println(line)
	plain := plainLine("[Miru Catch]", dt, loc, caught)
	_ = d.writer.append(plain)
}

// Out prints values to the console only (like console.log). Never writes to log files.
// Each argument is printed on its own line with [Miru Out]: dateTime location -> value.
func (d *Debugger) Out(args ...interface{}) {
	loc := d.getLocation(2)
	dt := d.dateTime()
	for _, a := range args {
		value := formatValue(a)
		line := d.formatOutLine(dt, loc, value)
		fmt.Println(line)
	}
}

// formatValue returns a string representation suitable for logging (JSON for structs/maps/slices).
func formatValue(v interface{}) string {
	if v == nil {
		return "nil"
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Test runs fn (a function with no arguments) and compares its return value to expectedOutput.
// Displays [Miru Test]: dateTime funcName -> PASSED/FAILED (duration).
// When IncludeTests is true, also appends the result to the log file.
func (d *Debugger) Test(funcName string, fn interface{}, expectedOutput interface{}) {
	start := time.Now()
	passed := false
	fv := reflect.ValueOf(fn)
	if fv.Kind() != reflect.Func {
		d.Out("Test error: fn is not a function")
		return
	}
	// Recover from panic in fn
	func() {
		defer func() {
			if r := recover(); r != nil {
				passed = false
			}
		}()
		out := fv.Call(nil)
		if len(out) == 1 {
			passed = reflect.DeepEqual(out[0].Interface(), expectedOutput)
		} else if len(out) == 0 {
			passed = expectedOutput == nil
		} else {
			var results []interface{}
			for i := 0; i < len(out); i++ {
				results = append(results, out[i].Interface())
			}
			passed = reflect.DeepEqual(results, expectedOutput)
		}
	}()
	elapsed := time.Since(start)
	ms := fmt.Sprintf("%.2fms", elapsed.Seconds()*1000)
	dt := d.dateTime()
	status := "PASSED"
	if !passed {
		status = "FAILED"
	}
	line := d.formatTestLine(dt, funcName, status, "("+ms+")", passed)
	fmt.Println(line)
	if d.config.IncludeTests {
		plain := plainLine("[Miru Test]", dt, funcName+"\t->\t"+status, "("+ms+")")
		_ = d.writer.append(plain)
	}
}

// Trace returns a function to be deferred to measure execution time.
// Usage: defer debug.Trace("someFunc")()
// Output: [Miru Trace]: dateTime someFunc -> 0.25ms
func (d *Debugger) Trace(name string) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		ms := fmt.Sprintf("%.2fms", elapsed.Seconds()*1000)
		dt := d.dateTime()
		line := d.formatTraceLine(dt, name, ms)
		fmt.Println(line)
	}
}
