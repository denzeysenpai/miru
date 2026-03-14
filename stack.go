package miru

import (
	"fmt"
	"runtime"
	"strings"
)

const stackIndent = "  "

// CheckStack prints the current goroutine's stack trace. Console only.
func (d *Debugger) CheckStack() {
	loc := d.getLocation(2)
	dt := d.dateTime()
	header := d.formatCheckStackHeader(dt, loc)
	fmt.Println(header)

	buf := make([]byte, 4096)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, len(buf)*2)
	}

	d.printStackPretty(string(buf))
}

// printStackPretty prints the stack with consistent indentation and optional color.
func (d *Debugger) printStackPretty(raw string) {
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	for i, line := range lines {
		indent := stackIndent
		if i == 0 {
			// goroutine 1 [running]:
			if d.config.Colorful {
				line = d.yellow(line)
			}
		} else {
			// frame lines (function then file:line); indent file lines a bit more
			if strings.HasPrefix(line, "\t") {
				indent = stackIndent + "  "
				line = strings.TrimPrefix(line, "\t")
			}
		}
		fmt.Println(indent + line)
	}
}
