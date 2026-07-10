package xmi_test

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
)

// syntheticElementKinds mirrors the UML types defaultKindMap resolves to
// concrete Bausteinsicht kinds (see xmi.go) — used to generate a realistic
// mix of leaf elements.
var syntheticElementKinds = []string{
	"Component", "Class", "Artifact", "Interface", "Node",
	"DataType", "Enumeration", "Subsystem", "UseCase", "Collaboration",
}

// syntheticStereotypes mirrors common AUTOSAR/EA stereotype names — applied
// to a fraction of generated elements to exercise the stereotype-as-kind path
// (xmi.go's resolveKind checks stereotype before the UML-type kindMap).
var syntheticStereotypes = []string{
	"SWComponent", "BSWModule", "PortInterface", "ECUAbstraction",
}

// stereotypeEvery gates how often a stereotype is attached, keyed off elemIdx
// directly rather than off the same modulus as syntheticElementKinds'
// len(10) cycle. 7 and 10 are coprime, so which of the 10 UML kinds gets a
// stereotype varies across elements instead of always/never pairing the same
// kind with the same stereotype-presence — every kind still gets exercised
// via defaultKindMap's plain UML-type path (not just the stereotype path)
// for most of its occurrences.
const stereotypeEvery = 7

// syntheticWriter wraps a *bufio.Writer and captures the first write error
// instead of requiring every fmt.Fprintf call site to check one — with dozens
// of writes in writeSyntheticXMI, checking each individually would swamp the
// generation logic; sink() below reports whatever was captured once, at the end.
type syntheticWriter struct {
	bw  *bufio.Writer
	err error
}

func (w *syntheticWriter) printf(format string, args ...any) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.bw, format, args...)
}

// writeSyntheticXMI writes a synthetic XMI 2.1 / UML 2.1 document to w that
// mirrors the shape of a real large Enterprise Architect / AUTOSAR export —
// numBranches top-level packages (real exports have several, e.g. AUTOSAR /
// ReadMe / ExportConfiguration / ...), each nested down to depth, with
// numElements leaf elements distributed across the branches' deepest
// packages (mixed types, occasional stereotypes/attributes/comments, and
// relationships between them). Used by TestImport_SyntheticBigData to
// provide scale/depth coverage in every CI job without a network fetch,
// complementing TestImport_BigData's real-file (fetched, #553) coverage.
// Deterministic (fixed seed): repeated runs produce byte-identical output,
// so the generated fixture doesn't introduce non-determinism into the test.
func writeSyntheticXMI(w io.Writer, numElements, depth, numBranches int) error {
	rng := rand.New(rand.NewSource(42)) // #nosec G404 -- deterministic test fixture, not security-sensitive
	sw := &syntheticWriter{bw: bufio.NewWriter(w)}

	genID := func(prefix string) string {
		return fmt.Sprintf("%s_%08X_%04X_%04x_%04X_%012X",
			prefix, rng.Uint32(), rng.Uint32()&0xFFFF, rng.Uint32()&0xFFFF, rng.Uint32()&0xFFFF, rng.Uint64()&0xFFFFFFFFFFFF)
	}

	sw.printf("<?xml version='1.0' encoding='windows-1252' ?>\n")
	sw.printf("<xmi:XMI xmlns:xmi=\"http://schema.omg.org/spec/XMI/2.1\" xmi:version=\"2.1\" xmlns:uml=\"http://schema.omg.org/spec/UML/2.1\">\n")
	sw.printf("\t<xmi:Documentation exporter=\"Bausteinsicht synthetic generator\" exporterVersion=\"1.0\"/>\n")
	sw.printf("\t<uml:Model xmi:type=\"uml:Model\" name=\"EA_Model\" visibility=\"public\">\n")

	generatedIDs := make([]string, 0, numElements)
	elementsPerBranch := numElements / numBranches
	elemIdx := 0

	for b := 0; b < numBranches; b++ {
		// Nested package chain down to `depth` for this branch — mirrors
		// real-world EA/AUTOSAR export nesting (xmi.go's maxDepth comment
		// cites depth 23 as observed) and gives multiple top-level packages,
		// like a real export's AUTOSAR/ReadMe/ExportConfiguration/... roots.
		indent := "\t\t"
		for i := 0; i < depth; i++ {
			sw.printf("%s<packagedElement xmi:type=\"uml:Package\" xmi:id=\"%s\" name=\"Branch%d_Package%d\" visibility=\"public\">\n",
				indent, genID("EAPK"), b, i)
			indent += "\t"
		}

		// Leaf elements at the deepest package level of this branch, plus
		// relationships between them (referencing earlier-generated IDs,
		// matching how EA exports interleave packagedElement and
		// Dependency/Association entries).
		limit := elemIdx + elementsPerBranch
		if b == numBranches-1 {
			limit = numElements // last branch absorbs any remainder
		}
		for ; elemIdx < limit; elemIdx++ {
			kind := syntheticElementKinds[elemIdx%len(syntheticElementKinds)]
			id := genID("EAID")
			generatedIDs = append(generatedIDs, id)

			sw.printf("%s<packagedElement xmi:type=\"uml:%s\" xmi:id=\"%s\" name=\"Element%d\" visibility=\"public\">\n",
				indent, kind, id, elemIdx)

			if elemIdx%3 == 0 {
				sw.printf("%s\t<ownedAttribute xmi:type=\"uml:Property\" xmi:id=\"%s\" name=\"attr%d\" visibility=\"private\"/>\n",
					indent, genID("EAID"), elemIdx)
			}
			if elemIdx%4 == 0 {
				sw.printf("%s\t<ownedComment xmi:type=\"uml:Comment\" xmi:id=\"%s\" body=\"Synthetic comment for element %d, used for scale testing only.\"/>\n",
					indent, genID("EAID"), elemIdx)
			}
			if elemIdx%stereotypeEvery == 0 {
				stereo := syntheticStereotypes[(elemIdx/stereotypeEvery)%len(syntheticStereotypes)]
				sw.printf("%s\t<xmi:Extension extender=\"Bausteinsicht synthetic generator\">\n", indent)
				sw.printf("%s\t\t<stereotype xmi:id=\"%s\" name=\"%s\"/>\n", indent, genID("EAID"), stereo)
				sw.printf("%s\t</xmi:Extension>\n", indent)
			}

			// Every 5th element depends on an earlier one, once enough exist.
			if elemIdx%5 == 0 && len(generatedIDs) > 1 {
				target := generatedIDs[rng.Intn(len(generatedIDs)-1)]
				sw.printf("%s\t<packagedElement xmi:type=\"uml:Dependency\" xmi:id=\"%s\" name=\"dep%d\" client=\"%s\" supplier=\"%s\"/>\n",
					indent, genID("EAID"), elemIdx, id, target)
			}

			sw.printf("%s</packagedElement>\n", indent)
		}

		// Close this branch's package chain in reverse order.
		for i := depth - 1; i >= 0; i-- {
			indent = indent[:len(indent)-1]
			sw.printf("%s</packagedElement>\n", indent)
		}
	}

	sw.printf("\t</uml:Model>\n")
	sw.printf("</xmi:XMI>\n")

	if sw.err != nil {
		return sw.err
	}
	return sw.bw.Flush()
}
