package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// This file adds the text-level edits that "update", "rename" and "remove" need
// to keep the comments and the formatting of the model (ADR-011, #685): replace
// a value anywhere (also inside arrays), remove an object entry or an array
// element together with the comment lines above it, and rename an object key.
// A path is a list of segments; "[N]" selects the N-th element of an array.

// SplitChangePath turns the path of a Change ("model.a.children.b.title",
// "relationships[3].from", "views.x.include[1]") into path segments.
func SplitChangePath(p string) []string {
	var segs []string
	for _, part := range strings.Split(p, ".") {
		for part != "" {
			i := strings.Index(part, "[")
			if i < 0 {
				segs = append(segs, part)
				break
			}
			if i > 0 {
				segs = append(segs, part[:i])
			}
			j := strings.Index(part[i:], "]")
			if j < 0 {
				segs = append(segs, part[i:])
				break
			}
			segs = append(segs, part[i:i+j+1])
			part = part[i+j+1:]
		}
	}
	return segs
}

func arrayIndex(seg string) (int, bool) {
	if len(seg) > 2 && seg[0] == '[' && seg[len(seg)-1] == ']' {
		if n, err := strconv.Atoi(seg[1 : len(seg)-1]); err == nil && n >= 0 {
			return n, true
		}
	}
	return 0, false
}

// childRange finds the child seg of the object or array whose value starts at
// start: entry is where the entry begins (the key's quote, or the element),
// valStart and valEnd delimit the child's value.
func childRange(data []byte, start int, seg string) (entry, valStart, valEnd int, err error) {
	n := len(data)
	i := skipWhitespaceAndComments(data, start, n)
	if idx, isIdx := arrayIndex(seg); isIdx {
		if i >= n || data[i] != '[' {
			return 0, 0, 0, fmt.Errorf("segment %s: expected an array", seg)
		}
		i++
		for k := 0; ; k++ {
			i = skipWhitespaceAndComments(data, i, n)
			if i >= n || data[i] == ']' {
				return 0, 0, 0, fmt.Errorf("segment %s: the array has only %d element(s)", seg, k)
			}
			e := skipValue(data, i, n)
			if k == idx {
				return i, i, e, nil
			}
			i = skipWhitespaceAndComments(data, e, n)
			if i < n && data[i] == ',' {
				i++
			}
		}
	}
	if i >= n || data[i] != '{' {
		return 0, 0, 0, fmt.Errorf("segment %q: expected an object", seg)
	}
	i++
	for {
		i = skipWhitespaceAndComments(data, i, n)
		if i >= n || data[i] == '}' {
			return 0, 0, 0, fmt.Errorf("key %q not found", seg)
		}
		if data[i] != '"' {
			return 0, 0, 0, fmt.Errorf("expected '\"' at offset %d", i)
		}
		keyStart := i
		keyEnd := skipString(data, i, n)
		key := string(data[keyStart+1 : keyEnd-1])
		i = skipWhitespaceAndComments(data, keyEnd, n)
		if i >= n || data[i] != ':' {
			return 0, 0, 0, fmt.Errorf("expected ':' after key %q", key)
		}
		i = skipWhitespaceAndComments(data, i+1, n)
		e := skipValue(data, i, n)
		if key == seg {
			return keyStart, i, e, nil
		}
		i = skipWhitespaceAndComments(data, e, n)
		if i < n && data[i] == ',' {
			i++
		}
	}
}

// locateEntry returns the entry range for the last segment of path.
func locateEntry(data []byte, path []string) (entry, valStart, valEnd int, err error) {
	if len(path) == 0 {
		return 0, 0, 0, fmt.Errorf("empty path")
	}
	cur := 0
	for k, seg := range path {
		entry, valStart, valEnd, err = childRange(data, cur, seg)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("path %v: %w", path[:k+1], err)
		}
		cur = valStart
	}
	return entry, valStart, valEnd, nil
}

// ReplaceValueAt replaces the value at path (which may run through arrays) with
// valueJSON and leaves everything else as it is.
func ReplaceValueAt(data []byte, path []string, valueJSON string) ([]byte, error) {
	_, vs, ve, err := locateEntry(data, path)
	if err != nil {
		return nil, err
	}
	out := append([]byte{}, data[:vs]...)
	out = append(out, valueJSON...)
	return append(out, data[ve:]...), nil
}

// RenameObjectKey renames the key at path (its last segment) to newKey.
func RenameObjectKey(data []byte, path []string, newKey string) ([]byte, error) {
	entry, _, _, err := locateEntry(data, path)
	if err != nil {
		return nil, err
	}
	if _, isIdx := arrayIndex(path[len(path)-1]); isIdx || data[entry] != '"' {
		return nil, fmt.Errorf("path %v does not end at an object key", path)
	}
	keyEnd := skipString(data, entry, len(data))
	quoted, err := json.Marshal(newKey)
	if err != nil {
		return nil, err
	}
	out := append([]byte{}, data[:entry]...)
	out = append(out, quoted...)
	return append(out, data[keyEnd:]...), nil
}

var trailingComma = regexp.MustCompile(`,(\s*(//.*)?)$`)

// RemoveEntry removes the object entry or array element at path. An entry that
// has its own lines is removed with them, together with the comment lines
// directly above it (they describe it); a comment after its comma goes with it.
// The comma of the previous entry is removed if the removed one was the last.
func RemoveEntry(data []byte, path []string) ([]byte, error) {
	entry, _, valEnd, err := locateEntry(data, path)
	if err != nil {
		return nil, err
	}
	n := len(data)

	// After the value: an optional comma, then an optional comment up to the end of the line.
	j := valEnd
	for j < n && (data[j] == ' ' || data[j] == '\t') {
		j++
	}
	hadComma := false
	if j < n && data[j] == ',' {
		hadComma = true
		j++
	}
	k := j
	for k < n && (data[k] == ' ' || data[k] == '\t') {
		k++
	}
	endOfLine := k >= n || data[k] == '\n' || data[k] == '\r' ||
		(k+1 < n && data[k] == '/' && data[k+1] == '/')
	if k+1 < n && data[k] == '/' && data[k+1] == '/' {
		for k < n && data[k] != '\n' {
			k++
		}
	}
	if k < n && data[k] == '\r' {
		k++
	}
	if k < n && data[k] == '\n' {
		k++
	}

	ls := entry
	for ls > 0 && (data[ls-1] == ' ' || data[ls-1] == '\t') {
		ls--
	}
	ownLines := (ls == 0 || data[ls-1] == '\n') && endOfLine

	var start, end int
	if ownLines {
		start, end = ls, k
		// The comment lines directly above describe the entry.
		for start > 0 {
			prev := start - 1 // the '\n' that ends the previous line
			pl := prev
			for pl > 0 && data[pl-1] != '\n' {
				pl--
			}
			if !strings.HasPrefix(strings.TrimSpace(string(data[pl:prev])), "//") {
				break
			}
			start = pl
		}
	} else {
		// On a line with other content: remove the entry and its comma only.
		start, end = entry, j
		for end < n && (data[end] == ' ' || data[end] == '\t') {
			end++
		}
	}

	out := append([]byte{}, data[:start]...)
	out = append(out, data[end:]...)
	if hadComma {
		return out, nil
	}
	// The removed entry was the last: the entry before it must lose its comma.
	return dropPreviousComma(out, start)
}

// dropPreviousComma removes the comma that ends the last entry before offset
// pos (it is followed by nothing but whitespace and a comment).
func dropPreviousComma(data []byte, pos int) ([]byte, error) {
	i := pos
	for i > 0 {
		ls := i
		for ls > 0 && data[ls-1] != '\n' {
			ls--
		}
		line := string(data[ls:i])
		if strings.TrimSpace(line) != "" {
			if loc := trailingComma.FindStringSubmatchIndex(line); loc != nil {
				at := ls + loc[0]
				return append(append([]byte{}, data[:at]...), data[at+1:]...), nil
			}
			return data, nil // the previous line is the opening bracket, or has no comma
		}
		if ls == 0 {
			break
		}
		i = ls - 1
	}
	return data, nil
}
