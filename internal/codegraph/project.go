package codegraph

import (
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// ProjectionResult is the outcome of collapsing a CodeGraph onto model
// elements via their codeMapping.
type ProjectionResult struct {
	// Snapshot is the resulting asIs snapshot: elements resolved from the
	// graph's nodes, and inter-element relationships resolved from its edges.
	// Intra-element edges (both endpoints resolving to the same element) are
	// dropped — they are implementation detail, not architecture.
	Snapshot *model.ModelSnapshot

	// UnmappedPackages lists graph packages (nodes and edge endpoints) that
	// no element's codeMapping.packages covers, deduplicated and sorted.
	// Collected rather than silently dropped, so the caller can report them.
	UnmappedPackages []string
}

// Project collapses cg onto m's codeMapping-anchored elements. Matching is a
// longest-prefix, dot-boundary match mirroring ArchUnit's own package
// matcher: a codeMapping package "com.acme.api" covers "com.acme.api" and
// "com.acme.api.internal", but not "com.acme.apiextra". See #562 and
// ADR-013.
func Project(cg *CodeGraph, m *model.BausteinsichtModel) (*ProjectionResult, error) {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return nil, err
	}

	mappings := buildPackageIndex(flat)
	elemIDs := make(map[string]bool)
	unmapped := make(map[string]bool)

	resolve := func(pkg string) (string, bool) {
		if owner := resolvePackage(pkg, mappings); owner != "" {
			elemIDs[owner] = true
			return owner, true
		}
		unmapped[pkg] = true
		return "", false
	}

	for _, n := range cg.Nodes {
		resolve(n.ID)
	}

	relSeen := make(map[[2]string]bool)
	var rels []model.Relationship
	for _, e := range cg.Edges {
		fromOwner, fromOK := resolve(e.From)
		toOwner, toOK := resolve(e.To)
		if !fromOK || !toOK || fromOwner == toOwner {
			continue
		}
		key := [2]string{fromOwner, toOwner}
		if relSeen[key] {
			continue
		}
		relSeen[key] = true
		rels = append(rels, model.Relationship{From: fromOwner, To: toOwner})
	}

	elements := make(map[string]model.Element, len(elemIDs))
	for id := range elemIDs {
		if elem, ok := flat[id]; ok {
			elements[id] = *elem
		}
	}

	unmappedList := make([]string, 0, len(unmapped))
	for pkg := range unmapped {
		unmappedList = append(unmappedList, pkg)
	}
	sort.Strings(unmappedList)

	return &ProjectionResult{
		Snapshot:         &model.ModelSnapshot{Elements: elements, Relationships: rels},
		UnmappedPackages: unmappedList,
	}, nil
}

type packageMapping struct {
	elementID string
	prefix    string
}

func buildPackageIndex(flat map[string]*model.Element) []packageMapping {
	var mappings []packageMapping
	for id, elem := range flat {
		if elem.CodeMapping == nil {
			continue
		}
		for _, pkg := range elem.CodeMapping.Packages {
			mappings = append(mappings, packageMapping{elementID: id, prefix: pkg})
		}
	}
	return mappings
}

// resolvePackage returns the element ID whose codeMapping prefix most
// specifically covers pkg (longest match wins when prefixes overlap), or ""
// if no prefix covers it.
func resolvePackage(pkg string, mappings []packageMapping) string {
	best, bestLen := "", -1
	for _, mp := range mappings {
		if matchesPackage(pkg, mp.prefix) && len(mp.prefix) > bestLen {
			best, bestLen = mp.elementID, len(mp.prefix)
		}
	}
	return best
}

// matchesPackage reports whether prefix covers pkg: either an exact match,
// or pkg is a dot-separated subpackage of prefix (ArchUnit's ".." wildcard
// semantics — "com.acme.api" also covers "com.acme.api.internal").
func matchesPackage(pkg, prefix string) bool {
	return pkg == prefix || strings.HasPrefix(pkg, prefix+".")
}

// HasAnyCodeMapping reports whether the model has at least one element with
// a codeMapping — the precondition for a meaningful import-graph run.
func HasAnyCodeMapping(m *model.BausteinsichtModel) (bool, error) {
	flat, err := model.FlattenElements(m)
	if err != nil {
		return false, err
	}
	for _, elem := range flat {
		if elem.CodeMapping != nil {
			return true, nil
		}
	}
	return false, nil
}
