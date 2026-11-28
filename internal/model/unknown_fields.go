package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// topLevelPath is the path of the warnings about keys of the file's root object.
const topLevelPath = "(top level)"

// unknownFieldWarnings finds the keys of the (comment-free) JSON text that no
// field of the model types has: a misspelt key is otherwise dropped without a
// word and lost on the next rewrite of the file (#725). Free-form maps (`meta`,
// `style`, `metadata`) are not entered, so custom data stays possible there.
func unknownFieldWarnings(clean []byte) []ValidationWarning {
	var raw interface{}
	if err := json.Unmarshal(clean, &raw); err != nil {
		return nil
	}
	var ws []ValidationWarning
	walkUnknownFields(raw, reflect.TypeOf(BausteinsichtModel{}), "", &ws)
	return ws
}

func walkUnknownFields(val interface{}, t reflect.Type, path string, ws *[]ValidationWarning) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		obj, ok := val.(map[string]interface{})
		if !ok {
			return
		}
		fields := jsonFields(t)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ft, known := lookupField(fields, k)
			if !known {
				*ws = append(*ws, unknownFieldWarning(path, k, fields))
				continue
			}
			walkUnknownFields(obj[k], ft, joinPath(path, k), ws)
		}
	case reflect.Map:
		obj, ok := val.(map[string]interface{})
		if !ok {
			return
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walkUnknownFields(obj[k], t.Elem(), joinPath(path, k), ws)
		}
	case reflect.Slice:
		items, ok := val.([]interface{})
		if !ok {
			return
		}
		for i, item := range items {
			walkUnknownFields(item, t.Elem(), fmt.Sprintf("%s[%d]", path, i), ws)
		}
	}
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// jsonFields returns the JSON names of the fields of struct type t.
func jsonFields(t reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Name
		if tag := f.Tag.Get("json"); tag != "" {
			if tag == "-" {
				continue
			}
			if n, _, _ := strings.Cut(tag, ","); n != "" {
				name = n
			}
		}
		fields[name] = f.Type
	}
	return fields
}

// lookupField finds key among the fields; encoding/json matches the names
// without regard to case, so the check does too.
func lookupField(fields map[string]reflect.Type, key string) (reflect.Type, bool) {
	if t, ok := fields[key]; ok {
		return t, true
	}
	for name, t := range fields {
		if strings.EqualFold(name, key) {
			return t, true
		}
	}
	return nil, false
}

func unknownFieldWarning(path, key string, fields map[string]reflect.Type) ValidationWarning {
	kind, where := "field", path
	if path == "" {
		kind, where = "top-level key", topLevelPath
	}
	msg := fmt.Sprintf("unknown %s %q", kind, key)
	if near := nearestName(key, fields); near != "" {
		msg += fmt.Sprintf(" (did you mean %q?)", near)
	}
	return ValidationWarning{Path: where, Message: msg}
}

// nearestName returns the field name closest to key if it is close enough to
// be a typo (at most two edits, fewer than the length of key).
func nearestName(key string, fields map[string]reflect.Type) string {
	best, bestDist := "", 3
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if d := editDistance(strings.ToLower(key), strings.ToLower(n)); d < bestDist && d < len(key) {
			best, bestDist = n, d
		}
	}
	return best
}

// editDistance is the optimal string alignment distance (insert, delete,
// substitute, swap of neighbours).
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := 0; j <= len(rb); j++ {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}
