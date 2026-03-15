package miru

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

type tracker struct {
	name     string
	target   any
	last     map[string]string
	mu       sync.Mutex
	debugger *Debugger
}

func (t *tracker) snapshot() {

	v := reflect.ValueOf(t.target)

	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {

		field := typ.Field(i)
		value := v.Field(i).Interface()

		t.last[field.Name] = formatValue(value)
	}
}

func (t *tracker) compare() {

	v := reflect.ValueOf(t.target)

	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {

		field := typ.Field(i)
		val := v.Field(i).Interface()

		current := formatValue(val)

		prev, exists := t.last[field.Name]

		if !exists {
			t.last[field.Name] = current
			continue
		}

		if current != prev {

			t.last[field.Name] = current

			d := t.debugger
			dt := d.dateTime()

			tag := d.green("[Miru Track]")
			date := d.yellow(dt)

			name := fmt.Sprintf("%s.%s", t.name, field.Name)

			body := fmt.Sprintf("%s\t->\t%s (was %s)", name, current, prev)

			line := fmt.Sprintf("%s:\t%s\t%s", tag, date, body)

			fmt.Println(line)

			d.emit("Trace", line)

			if d.config.IncludeTests {

				plain := plainLine("[Miru Track]", dt, name, current)
				_ = d.writer.append(plain)

			}
		}
	}
}
func (t *tracker) watchLoop() {

	for {
		time.Sleep(10 * time.Millisecond)

		t.mu.Lock()
		t.compare()
		t.mu.Unlock()
	}
}

// Logs whenever the given 'v' struct changes its values
func (d *Debugger) Track(name string, v any) {

	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		panic("debug.Track requires a struct or a pointer to a struct")
	}

	t := &tracker{
		name:     name,
		target:   v,
		last:     map[string]string{},
		debugger: d,
	}

	t.snapshot()

	go t.watchLoop()
}
