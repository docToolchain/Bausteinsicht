// Package xmi parses XMI 2.1 files (e.g. Enterprise Architect exports) and
// converts them to the Bausteinsicht model format.
package xmi

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/docToolchain/Bausteinsicht/internal/importer"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

const (
	// maxDepth caps element nesting to prevent stack/heap exhaustion on pathological inputs.
	// Real-world EA exports (e.g. AUTOSAR) can reach depth 23; 50 gives ample headroom.
	maxDepth = 50

	// maxXMIFileSize is the per-file read cap for XMI import (200 MB).
	// win1252ToUTF8 allocates a second copy, so peak RSS is ~2× the raw file size;
	// the cap prevents runaway memory use on pathological inputs.
	maxXMIFileSize = 200 * 1024 * 1024

	fallbackKind = "element"
)

// defaultKindMap maps UML type local names to Bausteinsicht kind names.
var defaultKindMap = map[string]string{
	"Package":       "package",
	"Component":     "component",
	"Class":         "class",
	"Actor":         "actor",
	"Interface":     "interface",
	"Node":          "node",
	"Artifact":      "artifact",
	"DataType":      "datatype",
	"Enumeration":   "enumeration",
	"Subsystem":     "subsystem",
	"UseCase":       "usecase",
	"Collaboration": "collaboration",
}

// isRelationshipType returns true for UML type local names that denote connectors.
func isRelationshipType(local string) bool {
	switch local {
	case "Dependency", "Association", "Realization", "Usage",
		"Abstraction", "Generalization", "InformationFlow",
		"AssociationClass", "ComponentRealization":
		return true
	}
	return false
}

// ─── Intermediate representation ─────────────────────────────────────────────

type xmiElem struct {
	XMIType    string // value of xmi:type attribute (e.g. "uml:Component")
	XMIID      string // value of xmi:id attribute
	Name       string // value of name attribute
	Stereotype string // stereotype name (from xmi:Extension > stereotype)
	Client     string // relationship source xmi:id
	Supplier   string // relationship target xmi:id
	Children   []*xmiElem
}

// umlLocal returns the local part of the xmi:type (e.g. "Component" from "uml:Component").
func (e *xmiElem) umlLocal() string {
	if i := strings.LastIndex(e.XMIType, ":"); i >= 0 {
		return e.XMIType[i+1:]
	}
	return e.XMIType
}

// ─── XXE protection ──────────────────────────────────────────────────────────

var doctypeRe = regexp.MustCompile(`(?i)<!DOCTYPE`)

// hasDOCTYPE scans the first 4096 bytes for a DOCTYPE declaration.
// The window is intentionally limited: XML declarations and DOCTYPE must appear
// in the document prologue, which precedes any content and is always within the
// first few hundred bytes in practice. A file engineered with >4096 bytes of
// whitespace before DOCTYPE would still be safe because dec.Entity = map[string]string{}
// (set in parseXMI) disables entity expansion regardless of DOCTYPE position —
// hasDOCTYPE is defense-in-depth, not the primary XXE mitigation.
func hasDOCTYPE(data []byte) bool {
	limit := len(data)
	if limit > 4096 {
		limit = 4096
	}
	return doctypeRe.Match(data[:limit])
}

// ─── Charset conversion ───────────────────────────────────────────────────────

// windows1252Table maps bytes 0x80–0x9F to their Unicode codepoints.
// Outside this range, Windows-1252 and ISO-8859-1 are identical to Latin-1
// (bytes 0xA0–0xFF equal Unicode codepoints U+00A0–U+00FF).
var windows1252Table = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

// win1252ToUTF8 converts a Windows-1252 (or ISO-8859-1/Latin-1) byte slice to UTF-8.
func win1252ToUTF8(data []byte) []byte {
	out := make([]byte, 0, len(data)+len(data)/8)
	for _, b := range data {
		if b < 0x80 {
			out = append(out, b)
			continue
		}
		var rn rune
		if b <= 0x9F {
			rn = windows1252Table[b-0x80]
		} else {
			rn = rune(b) // 0xA0–0xFF: same codepoint in Unicode
		}
		out = utf8.AppendRune(out, rn)
	}
	return out
}

// charsetReader is the xml.Decoder.CharsetReader implementation.
// It handles Windows-1252 and ISO-8859-1/Latin-1, which are common in EA exports.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	norm := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(charset, "-", ""), "_", ""))
	switch norm {
	case "windows1252", "cp1252", "win1252", "iso88591", "latin1":
		data, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(win1252ToUTF8(data)), nil
	default:
		return nil, fmt.Errorf("unsupported charset %q — convert the file to UTF-8 first", charset)
	}
}

// ─── XML parsing ─────────────────────────────────────────────────────────────

// attrVal returns the value of the first XML attribute with the given local name.
func attrVal(attrs []xml.Attr, local string) string {
	for _, a := range attrs {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// parseXMI decodes XMI bytes into an intermediate element tree.
// Returns (nil, "", nil) if the document is valid XML but not an XMI file.
// The second return value is the xmi:version attribute from the root element.
func parseXMI(data []byte) (*xmiElem, string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Entity = map[string]string{}  // disable entity expansion (XXE mitigation)
	dec.CharsetReader = charsetReader // handle windows-1252 / ISO-8859-1 EA exports

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, "", nil
		}
		if err != nil {
			return nil, "", err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "XMI" {
			return nil, "", nil // not an XMI document
		}
		version := attrVal(se.Attr, "version")
		root, err := parseElem(dec, se, 0)
		return root, version, err
	}
}

// parseElem recursively parses one XML element and its children.
func parseElem(dec *xml.Decoder, se xml.StartElement, depth int) (*xmiElem, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("element hierarchy exceeds maximum depth of %d", maxDepth)
	}

	e := &xmiElem{
		XMIType:  attrVal(se.Attr, "type"),
		XMIID:    attrVal(se.Attr, "id"),
		Name:     attrVal(se.Attr, "name"),
		Client:   attrVal(se.Attr, "client"),
		Supplier: attrVal(se.Attr, "supplier"),
	}

	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("parsing XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// xmi:Extension wraps EA-specific data; extract stereotype, then skip the block
			if t.Name.Local == "Extension" {
				stereo, err := extractStereotype(dec)
				if err != nil {
					return nil, err
				}
				if stereo != "" && e.Stereotype == "" {
					e.Stereotype = stereo
				}
				continue
			}
			// Direct stereotype element (outside Extension, less common)
			if t.Name.Local == "stereotype" {
				// EA XMI 2.1 uses stereotype="…"; other tools use name="…"
				name := attrVal(t.Attr, "name")
				if name == "" {
					name = attrVal(t.Attr, "stereotype")
				}
				if name != "" && e.Stereotype == "" {
					e.Stereotype = name
				}
				if err := dec.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			child, err := parseElem(dec, t, depth+1)
			if err != nil {
				return nil, err
			}
			e.Children = append(e.Children, child)
		case xml.EndElement:
			return e, nil
		}
	}
}

// extractStereotype reads the content of an already-opened xmi:Extension element
// and returns the first stereotype name found inside it.
func extractStereotype(dec *xml.Decoder) (string, error) {
	var stereo string
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("parsing Extension: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if t.Name.Local == "stereotype" && stereo == "" {
				stereo = attrVal(t.Attr, "name")
				if stereo == "" {
					stereo = attrVal(t.Attr, "stereotype") // EA XMI 2.1 variant
				}
			}
		case xml.EndElement:
			if depth == 0 {
				return stereo, nil
			}
			depth--
		}
	}
}

// ─── Public API ───────────────────────────────────────────────────────────────

// Import reads the XMI file at path and returns a BausteinsichtModel.
// kindMap maps UML type local names or stereotype names to Bausteinsicht kind values.
// Pass nil for kindMap to use default mappings only.
func Import(path string, kindMap map[string]string) (*importer.ImportResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if info.Size() > maxXMIFileSize {
		return nil, fmt.Errorf("reading %s: file size %d exceeds XMI import limit of %d bytes",
			path, info.Size(), maxXMIFileSize)
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path from CLI flag
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return fromBytes(data, kindMap)
}

func fromBytes(data []byte, kindMap map[string]string) (*importer.ImportResult, error) {
	if hasDOCTYPE(data) {
		return nil, fmt.Errorf("import failed: DOCTYPE declarations are not allowed")
	}

	root, xmiVersion, err := parseXMI(data)
	if err != nil {
		return nil, fmt.Errorf("import failed: invalid XML: %w", err)
	}
	if root == nil {
		return nil, fmt.Errorf("import failed: not a valid XMI document")
	}

	s := &convState{
		kindMap: mergeKinds(kindMap),
		xmiToBS: map[string]string{},
		usedIDs: map[string]int{},
	}

	// XMI 1.1 has a completely different element encoding (type expressed as XML tag names,
	// not xmi:type attributes). Warn early so the user knows the result may be empty.
	if xmiVersion != "2.1" {
		s.warnings = append(s.warnings, fmt.Sprintf(
			"XMI version %q detected; only 2.1 is fully supported — import may be incomplete or empty", xmiVersion))
	}

	return s.convert(root)
}

// ─── Conversion ──────────────────────────────────────────────────────────────

type convState struct {
	kindMap  map[string]string // merged kind map
	xmiToBS  map[string]string // xmi:id → bausteinsicht dot-path ID
	usedIDs  map[string]int    // dot-path → use count (for collision detection)
	warnings []string
}

// mergeKinds returns a kind map with user overrides applied on top of defaults.
func mergeKinds(overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(defaultKindMap)+len(overrides))
	for k, v := range defaultKindMap {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	return merged
}

func (s *convState) convert(root *xmiElem) (*importer.ImportResult, error) {
	bsModel := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{},
		},
		Model: map[string]model.Element{},
		Views: map[string]model.View{},
	}

	// Find children to process — either from uml:Model wrapper or directly from root
	children := s.modelChildren(root)

	var rels []*xmiElem

	// First pass: elements (build ID registry)
	for _, child := range children {
		s.collectElem(child, "", bsModel.Model, bsModel, &rels)
	}

	// Second pass: relationships
	for _, rel := range rels {
		s.collectRel(rel, bsModel)
	}

	return &importer.ImportResult{
		Model:    bsModel,
		Warnings: s.warnings,
	}, nil
}

// modelChildren returns the direct children of the uml:Model element,
// falling back to the root's children if no model wrapper is found.
func (s *convState) modelChildren(root *xmiElem) []*xmiElem {
	for _, c := range root.Children {
		if c.umlLocal() == "Model" {
			return c.Children
		}
	}
	return root.Children
}

// collectElem recursively processes one XMI element and its children.
// Relationships encountered during traversal are appended to rels.
func (s *convState) collectElem(
	e *xmiElem,
	parentPath string,
	target map[string]model.Element,
	bsModel *model.BausteinsichtModel,
	rels *[]*xmiElem,
) {
	local := e.umlLocal()

	// Stash relationships for the second pass
	if isRelationshipType(local) {
		*rels = append(*rels, e)
		return
	}

	// Skip elements without a name or without a recognisable type
	if e.Name == "" || e.XMIType == "" {
		if e.Name != "" && e.XMIType == "" {
			s.warnings = append(s.warnings, fmt.Sprintf("element %q skipped: no xmi:type attribute", e.Name))
		}
		for _, child := range e.Children {
			s.collectElem(child, parentPath, target, bsModel, rels)
		}
		return
	}

	// Resolve element kind
	kind := s.resolveKind(e)

	// Generate dot-path ID
	bsPath := s.makeID(e.Name, parentPath)

	// Register xmi:id → dot-path for relationship resolution
	if e.XMIID != "" {
		s.xmiToBS[e.XMIID] = bsPath
	}

	// Add kind to specification if new
	if _, exists := bsModel.Specification.Elements[kind]; !exists {
		bsModel.Specification.Elements[kind] = model.ElementKind{
			Notation: titleCase(kind),
		}
	}

	// Build element
	elem := model.Element{
		Kind:  kind,
		Title: e.Name,
	}

	// Process children
	childMap := map[string]model.Element{}
	for _, child := range e.Children {
		s.collectElem(child, bsPath, childMap, bsModel, rels)
	}
	if len(childMap) > 0 {
		elem.Children = childMap
		// Mark this kind as a container in the specification — derived from the XMI hierarchy.
		if ks, ok := bsModel.Specification.Elements[kind]; ok && !ks.Container {
			ks.Container = true
			bsModel.Specification.Elements[kind] = ks
		}
	}

	// Store in target map with local key only
	mapKey := bsPath
	if parentPath != "" {
		mapKey = bsPath[len(parentPath)+1:]
	}
	target[mapKey] = elem
}

// collectRel converts one XMI relationship to a Bausteinsicht Relationship.
func (s *convState) collectRel(e *xmiElem, bsModel *model.BausteinsichtModel) {
	if e.Client == "" {
		return
	}
	fromID, ok := s.xmiToBS[e.Client]
	if !ok {
		s.warnings = append(s.warnings, fmt.Sprintf(
			"relationship skipped: client XMI ID %q not found", e.Client))
		return
	}
	toID, ok := s.xmiToBS[e.Supplier]
	if !ok {
		s.warnings = append(s.warnings, fmt.Sprintf(
			"relationship skipped: supplier XMI ID %q not found", e.Supplier))
		return
	}
	bsModel.Relationships = append(bsModel.Relationships, model.Relationship{
		From:  fromID,
		To:    toID,
		Label: e.Name,
	})
}

// resolveKind determines the Bausteinsicht kind for a UML element.
// Priority: stereotype (if in kindMap) > stereotype (raw lowercase) > UML type kindMap > fallback.
func (s *convState) resolveKind(e *xmiElem) string {
	// 1. Stereotype in kindMap
	if e.Stereotype != "" {
		if k, ok := s.kindMap[e.Stereotype]; ok {
			return k
		}
		// Use stereotype directly as kind (lowercase)
		return strings.ToLower(e.Stereotype)
	}

	// 2. UML type in kindMap
	if k, ok := s.kindMap[e.umlLocal()]; ok {
		return k
	}

	// 3. Fallback with warning
	s.warnings = append(s.warnings, fmt.Sprintf(
		"unknown UML type %s — imported with kind %q", e.XMIType, fallbackKind))
	return fallbackKind
}

// makeID generates a unique Bausteinsicht dot-path ID for an element.
func (s *convState) makeID(name, parentPath string) string {
	local := sanitizeID(name)
	fullPath := local
	if parentPath != "" {
		fullPath = parentPath + "." + local
	}

	count := s.usedIDs[fullPath]
	s.usedIDs[fullPath] = count + 1

	if count > 0 {
		suffixed := fmt.Sprintf("%s-%d", local, count+1)
		s.warnings = append(s.warnings, fmt.Sprintf(
			"ID collision: %q sanitizes to %q — using %q", name, local, suffixed))
		local = suffixed
		if parentPath != "" {
			fullPath = parentPath + "." + local
		} else {
			fullPath = local
		}
	}

	return fullPath
}

// sanitizeID converts a name to a valid Bausteinsicht element ID.
// Rules: lowercase, non-[a-z0-9_] runs → single hyphen, trim leading/trailing hyphens.
var nonIDRe = regexp.MustCompile(`[^a-z0-9_]+`)

func sanitizeID(name string) string {
	s := strings.ToLower(name)
	s = nonIDRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return fallbackKind
	}
	return s
}

// titleCase uppercases the first rune of s.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// ParseKindMap parses a comma-separated "Type=kind,Type2=kind2" string.
// Returns an error if any entry is malformed.
func ParseKindMap(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	result := map[string]string{}
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid --kind-map entry %q: expected Type=kind", entry)
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result, nil
}
