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

	body := d.printStackPretty(string(buf))
	fmt.Print(body)
	d.emit("CheckStack", header+"\n"+body)
}

// printStackPretty formats the stack and returns the string.
func (d *Debugger) printStackPretty(raw string) string {
	var out strings.Builder
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	for i, line := range lines {
		indent := stackIndent
		if i == 0 {
			if d.config.Colorful {
				line = d.yellow(line)
			}
		} else {
			if strings.HasPrefix(line, "\t") {
				indent = stackIndent + "  "
				line = strings.TrimPrefix(line, "\t")
			}
		}
		out.WriteString(indent + line + "\n")
	}
	return out.String()
}
