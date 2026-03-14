package miru

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"time"
)

type Debugger struct {
	mu          sync.Mutex
	config      DebugConfig
	writer      *writer
	currentFunc string
	dashboard   *dashboardHub
}

func NewDebugger() *Debugger {
	cfg := DefaultConfig()
	return NewDebuggerWithConfig(cfg)
}

func NewDebuggerWithConfig(cfg DebugConfig) *Debugger {
	cfg.setDefaults()
	d := &Debugger{config: cfg, writer: newWriter(cfg)}
	return d
}

func (d *Debugger) Config(cfg DebugConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	cfg.setDefaults()
	d.config = cfg
	d.writer = newWriter(cfg)
}

// Func sets the name used in log lines (call it at the top of the function).
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

func (d *Debugger) emit(tag, body string) {
	if d.dashboard != nil {
		d.dashboard.Send(LogEntry{Tag: tag, Body: body})
	}
}

// Catch writes the recovered panic to console + log file.
func (d *Debugger) Catch(r any) {
	loc := d.getLocation(2)
	dt := d.dateTime()
	caught := fmt.Sprintf("Caught: %v", r)
	if err, ok := r.(error); ok {
		caught = fmt.Sprintf("Caught: %v", err)
	}
	line := d.formatCatchLine(dt, loc, caught)
	fmt.Println(line)
	d.emit("Catch", line)
	plain := plainLine("[Miru Catch]", dt, loc, caught)
	_ = d.writer.append(plain)
}

// Out is like console.log — console only, no file. Any number of args, one line each.
func (d *Debugger) Out(args ...any) {
	loc := d.getLocation(2)
	dt := d.dateTime()
	for _, a := range args {
		value := formatValue(a)
		line := d.formatOutLine(dt, loc, value)
		fmt.Println(line)
		d.emit("Out", line)
	}
}

// Tap passes v to fn (e.g. to log it) and returns v unchanged. Like Ruby’s tap.
func (d *Debugger) Tap(v interface{}, fn func(interface{})) interface{} {
	fn(v)
	return v
}

func formatValue(v any) string {
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

// Test runs fn (with optional args) and checks its return against expectedOutput.
// No args: debug.Test("add", fn, 4). With args: debug.Test("add", fn, 7, 3, 4).
// IncludeTests in config controls whether we also append to the log file.
func (d *Debugger) Test(funcName string, fn any, expectedOutput any, args ...any) {
	start := time.Now()
	passed := false
	fv := reflect.ValueOf(fn)
	if fv.Kind() != reflect.Func {
		d.Out("Test error: fn is not a function")
		return
	}
	argVals := make([]reflect.Value, len(args))
	for i, a := range args {
		argVals[i] = reflect.ValueOf(a)
	}
	if fv.Type().NumIn() != len(argVals) {
		d.Out(fmt.Sprintf("Test error: fn expects %d args, got %d", fv.Type().NumIn(), len(argVals)))
		return
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				passed = false
			}
		}()
		out := fv.Call(argVals)
		if len(out) == 1 {
			passed = reflect.DeepEqual(out[0].Interface(), expectedOutput)
		} else if len(out) == 0 {
			passed = expectedOutput == nil
		} else {
			var results []any
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
	d.emit("Test", line)
	if d.config.IncludeTests {
		plain := plainLine("[Miru Test]", dt, funcName+"\t->\t"+status, "("+ms+")")
		_ = d.writer.append(plain)
	}
}

// RemoteDashboard starts a web server that streams logs/traces live. Port 0 or negative = 8765.
// Returns the server so you can call srv.Shutdown(ctx) when done.
func (d *Debugger) RemoteDashboard(port int) *http.Server {
	d.mu.Lock()
	if d.dashboard == nil {
		d.dashboard = newDashboardHub()
	}
	hub := d.dashboard
	d.mu.Unlock()
	return hub.RunServer(port)
}

// Trace measures how long until the deferred func runs. Use: defer debug.Trace("name")()
func (d *Debugger) Trace(name string) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		ms := fmt.Sprintf("%.2fms", elapsed.Seconds()*1000)
		dt := d.dateTime()
		line := d.formatTraceLine(dt, name, ms)
		fmt.Println(line)
		d.emit("Trace", line)
	}
}
