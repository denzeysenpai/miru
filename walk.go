package miru

import (
	"fmt"
	"reflect"
)

const walkIndent = "  "

// Walk pretty-prints a struct, slice, or map to the console. Depth comes from config (WalkDepth); -1 = no limit.
func (d *Debugger) Walk(v interface{}) {
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
		return
	}

	line := d.formatWalkHeader(dt, loc, typeSummary)
	fmt.Println(line)
	d.walkValue(val, 1, maxDepth)
}

func (d *Debugger) walkValue(val reflect.Value, depth, maxDepth int) {
	if !val.IsValid() {
		d.printIndented(depth, "nil")
		return
	}
	// -1 means no limit
	if maxDepth >= 0 && depth > maxDepth {
		d.printIndented(depth, "...")
		return
	}

	kind := val.Kind()
	if kind == reflect.Ptr {
		if val.IsNil() {
			d.printIndented(depth, "nil")
			return
		}
		val = val.Elem()
		kind = val.Kind()
	}

	switch kind {
	case reflect.Interface:
		if val.IsNil() {
			d.printIndented(depth, "nil")
		} else {
			d.walkValue(val.Elem(), depth, maxDepth)
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
				d.printIndented(depth, fmt.Sprintf("%s: %v", field.Name, fv.Interface()))
				continue
			}
			d.printIndented(depth, field.Name+":")
			d.walkValue(fv, depth+1, maxDepth)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			ev := val.Index(i)
			if d.walkPrimitive(ev) {
				d.printIndented(depth, fmt.Sprintf("[%d]: %v", i, ev.Interface()))
				continue
			}
			d.printIndented(depth, fmt.Sprintf("[%d]:", i))
			d.walkValue(ev, depth+1, maxDepth)
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			mv := val.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface())
			if d.walkPrimitive(mv) {
				d.printIndented(depth, fmt.Sprintf("%s: %v", keyStr, mv.Interface()))
				continue
			}
			d.printIndented(depth, keyStr+":")
			d.walkValue(mv, depth+1, maxDepth)
		}
	default:
		d.printIndented(depth, fmt.Sprintf("%v", val.Interface()))
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

func (d *Debugger) printIndented(depth int, s string) {
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += walkIndent
	}
	fmt.Println(prefix + s)
}
