package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func jsonString(s string) (string, error) {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return "", err
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// PatchUpdateChanges writes the field changes of UpdateElement to the raw model
// text: a changed value is replaced in place, a new field is inserted, a field
// set to "" is removed. Comments and formatting stay (#685).
func PatchUpdateChanges(data []byte, changes []Change) ([]byte, error) {
	var err error
	for _, c := range changes {
		segs := SplitChangePath(c.Path)
		if len(segs) < 2 {
			return nil, fmt.Errorf("unexpected path %q", c.Path)
		}
		object, key := segs[:len(segs)-1], segs[len(segs)-1]
		if c.To == "" {
			if HasPath(data, segs) {
				if data, err = RemoveEntry(data, segs); err != nil {
					return nil, err
				}
			}
			continue
		}
		value, err := jsonString(c.To)
		if err != nil {
			return nil, err
		}
		if data, err = UpsertObjectField(data, object, key, value); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// PatchRenameChanges writes the changes of RenameElement to the raw model text:
// every reference gets its new value in place and the key of the element is
// renamed. The first change is the element itself.
func PatchRenameChanges(data []byte, changes []Change) ([]byte, error) {
	if len(changes) == 0 {
		return data, nil
	}
	var err error
	for _, c := range changes[1:] { // references first: their paths do not move
		value, jerr := jsonString(c.To)
		if jerr != nil {
			return nil, jerr
		}
		if data, err = ReplaceValueAt(data, SplitChangePath(c.Path), value); err != nil {
			return nil, err
		}
	}
	newKey := changes[0].To
	if dot := strings.LastIndex(newKey, "."); dot >= 0 {
		newKey = newKey[dot+1:]
	}
	return RenameObjectKey(data, SplitChangePath(changes[0].Path), newKey)
}

var (
	reRelRef   = regexp.MustCompile(`^relationships\[(\d+)\]\.(from|to)$`)
	reViewRef  = regexp.MustCompile(`^views\.([^.\[]+)\.(include|exclude)\[(\d+)\]$`)
	reStepRef  = regexp.MustCompile(`^dynamicViews\[(\d+)\]\.steps\[(\d+)\]\.(from|to)$`)
	reDynView  = regexp.MustCompile(`^dynamicViews\[(\d+)\]$`)
	reRelEntry = regexp.MustCompile(`^relationships\[(\d+)\]$`)
)

// PatchRemoveChanges writes the changes of RemoveElement, RemoveRelationship
// and their cascade to the raw model text: the element goes, and so do the
// relationships, view entries, dynamic-view steps and dynamic views that
// referred to it. Entries are removed from the highest index down, so the
// indexes in the paths of the changes stay valid.
func PatchRemoveChanges(data []byte, changes []Change) ([]byte, error) {
	removals := map[string]map[int]bool{} // array path -> indexes
	segsOf := map[string][]string{}
	add := func(array []string, index int) {
		key := strings.Join(array, "/")
		if removals[key] == nil {
			removals[key] = map[int]bool{}
			segsOf[key] = array
		}
		removals[key][index] = true
	}
	var elementPaths [][]string
	for _, c := range changes {
		switch {
		case reRelRef.MatchString(c.Path):
			m := reRelRef.FindStringSubmatch(c.Path)
			i, _ := strconv.Atoi(m[1])
			add([]string{"relationships"}, i)
		case reRelEntry.MatchString(c.Path):
			i, _ := strconv.Atoi(reRelEntry.FindStringSubmatch(c.Path)[1])
			add([]string{"relationships"}, i)
		case reViewRef.MatchString(c.Path):
			m := reViewRef.FindStringSubmatch(c.Path)
			i, _ := strconv.Atoi(m[3])
			add([]string{"views", m[1], m[2]}, i)
		case reStepRef.MatchString(c.Path):
			m := reStepRef.FindStringSubmatch(c.Path)
			s, _ := strconv.Atoi(m[2])
			add([]string{"dynamicViews", "[" + m[1] + "]", "steps"}, s)
		case reDynView.MatchString(c.Path):
			i, _ := strconv.Atoi(reDynView.FindStringSubmatch(c.Path)[1])
			add([]string{"dynamicViews"}, i)
		case strings.HasPrefix(c.Path, "model."):
			elementPaths = append(elementPaths, SplitChangePath(c.Path))
		default:
			return nil, fmt.Errorf("unexpected path %q", c.Path)
		}
	}

	// Steps and view lists first, whole dynamic views and relationships after
	// them (their own indexes are not affected by the removals inside them).
	keys := make([]string, 0, len(removals))
	for k := range removals {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(segsOf[keys[i]]) > len(segsOf[keys[j]]) })
	var err error
	for _, k := range keys {
		var idx []int
		for i := range removals[k] {
			idx = append(idx, i)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(idx)))
		for _, i := range idx {
			path := append(append([]string{}, segsOf[k]...), "["+strconv.Itoa(i)+"]")
			// A step of a dynamic view that is removed as a whole is gone with it.
			if data, err = RemoveEntry(data, path); err != nil {
				if strings.HasPrefix(k, "dynamicViews/[") && !HasPath(data, segsOf[k]) {
					continue
				}
				return nil, err
			}
		}
	}
	for _, p := range elementPaths {
		if data, err = RemoveEntry(data, p); err != nil {
			return nil, err
		}
	}
	return data, nil
}
