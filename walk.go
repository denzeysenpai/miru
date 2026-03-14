package miru

import (
	"fmt"
	"reflect"
	"strings"
)

const walkIndent = "  "

// Walk pretty-prints a struct, slice, or map to the console. Depth comes from config (WalkDepth); -1 = no limit.
func (d *Debugger) Walk(v any) {
	loc := d.getLocation(2)
	dt := d.dateTime()
	maxDepth := d.config.WalkDepth
	if maxDepth == 0 {
		maxDepth = 5 // zero value from config
	}

	val := reflect.ValueOf(v)
	if !val.IsValid() {
		line := d.formatWalkHeader(dt, loc, "nil")
		fmt.Println(line)
		d.emit("Walk", line)
		return
	}

	kind := val.Kind()
	if kind == reflect.Ptr {
		val = val.Elem()
		kind = val.Kind()
	}

	var typeSummary string
	switch kind {
	case reflect.Struct:
		typeSummary = fmt.Sprintf("struct (%d fields)", val.NumField())
	case reflect.Slice:
		typeSummary = fmt.Sprintf("slice (len %d)", val.Len())
	case reflect.Array:
		typeSummary = fmt.Sprintf("array (len %d)", val.Len())
	case reflect.Map:
		typeSummary = fmt.Sprintf("map (len %d)", val.Len())
	default:
		line := d.formatWalkHeader(dt, loc, fmt.Sprintf("%v", v))
		fmt.Println(line)
		d.emit("Walk", line)
		return
	}

	line := d.formatWalkHeader(dt, loc, typeSummary)
	fmt.Println(line)
	var body strings.Builder
	body.WriteString(line)
	body.WriteByte('\n')
	d.walkValue(val, 1, maxDepth, &body)
	d.emit("Walk", body.String())
}

func (d *Debugger) walkValue(val reflect.Value, depth, maxDepth int, sb *strings.Builder) {
	if !val.IsValid() {
		d.printIndented(depth, "nil", sb)
		return
	}
	if maxDepth >= 0 && depth > maxDepth {
		d.printIndented(depth, "...", sb)
		return
	}

	kind := val.Kind()
	if kind == reflect.Ptr {
		if val.IsNil() {
			d.printIndented(depth, "nil", sb)
			return
		}
		val = val.Elem()
		kind = val.Kind()
	}

	switch kind {
	case reflect.Interface:
		if val.IsNil() {
			d.printIndented(depth, "nil", sb)
		} else {
			d.walkValue(val.Elem(), depth, maxDepth, sb)
		}
		return
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			fv := val.Field(i)
			if d.walkPrimitive(fv) {
				d.printIndented(depth, fmt.Sprintf("%s: %v", field.Name, fv.Interface()), sb)
				continue
			}
			d.printIndented(depth, field.Name+":", sb)
			d.walkValue(fv, depth+1, maxDepth, sb)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			ev := val.Index(i)
			if d.walkPrimitive(ev) {
				d.printIndented(depth, fmt.Sprintf("[%d]: %v", i, ev.Interface()), sb)
				continue
			}
			d.printIndented(depth, fmt.Sprintf("[%d]:", i), sb)
			d.walkValue(ev, depth+1, maxDepth, sb)
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			mv := val.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface())
			if d.walkPrimitive(mv) {
				d.printIndented(depth, fmt.Sprintf("%s: %v", keyStr, mv.Interface()), sb)
				continue
			}
			d.printIndented(depth, keyStr+":", sb)
			d.walkValue(mv, depth+1, maxDepth, sb)
		}
	default:
		d.printIndented(depth, fmt.Sprintf("%v", val.Interface()), sb)
	}
}

func (d *Debugger) walkPrimitive(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	k := v.Kind()
	if k == reflect.Ptr {
		return v.IsNil() || d.walkPrimitive(v.Elem())
	}
	switch k {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map, reflect.Interface:
		return false
	default:
		return true
	}
}

func (d *Debugger) printIndented(depth int, s string, sb *strings.Builder) {
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += walkIndent
	}
	line := prefix + s
	fmt.Println(line)
	if sb != nil {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
}
