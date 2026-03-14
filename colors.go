package miru

import "fmt"

const (
	ansiReset   = "\033[0m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
)

func (d *Debugger) red(s string) string {
	if d.config.Colorful {
		return ansiRed + s + ansiReset
	}
	return s
}

func (d *Debugger) green(s string) string {
	if d.config.Colorful {
		return ansiGreen + s + ansiReset
	}
	return s
}

func (d *Debugger) yellow(s string) string {
	if d.config.Colorful {
		return ansiYellow + s + ansiReset
	}
	return s
}

func (d *Debugger) formatCatchLine(dateTime, location, caught string) string {
	tag := d.red("[Miru Catch]")
	dt := d.yellow(dateTime)
	msg := d.red(caught)
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s", tag, dt, location, msg)
}

func (d *Debugger) formatOutLine(dateTime, location, value string) string {
	tag := d.red("[Miru Out]")
	dt := d.yellow(dateTime)
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s", tag, dt, location, value)
}

func (d *Debugger) formatTestLine(dateTime, funcName, status, duration string, passed bool) string {
	tag := d.green("[Miru Test]")
	dt := d.yellow(dateTime)
	ms := d.yellow(duration)
	var statusColored string
	if passed {
		statusColored = d.green(status)
	} else {
		statusColored = d.red(status)
	}
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s\t%s", tag, dt, funcName, statusColored, ms)
}

func (d *Debugger) formatTraceLine(dateTime, name, duration string) string {
	tag := d.green("[Miru Trace]")
	dt := d.yellow(dateTime)
	ms := d.yellow(duration)
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s", tag, dt, name, ms)
}

func (d *Debugger) formatWalkHeader(dateTime, location, typeSummary string) string {
	tag := d.green("[Miru Walk]")
	dt := d.yellow(dateTime)
	return fmt.Sprintf("%s:\t%s\t%s\t->\t%s", tag, dt, location, typeSummary)
}

func (d *Debugger) formatCheckStackHeader(dateTime, location string) string {
	tag := d.green("[Miru CheckStack]")
	dt := d.yellow(dateTime)
	return fmt.Sprintf("%s:\t%s\t%s\t->\tgoroutine stack", tag, dt, location)
}
