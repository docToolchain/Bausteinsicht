package sync

import (
	"reflect"
	"testing"
)

// symmetricAdj builds a bidirectional adjacency map from a list of pairs,
// matching the shape buildScopeAdjacency produces.
func symmetricAdj(pairs [][2]string) map[string]map[string]bool {
	adj := make(map[string]map[string]bool)
	ensure := func(id string) {
		if adj[id] == nil {
			adj[id] = make(map[string]bool)
		}
	}
	for _, p := range pairs {
		ensure(p[0])
		ensure(p[1])
		adj[p[0]][p[1]] = true
		adj[p[1]][p[0]] = true
	}
	return adj
}

func TestCountCrossingsBetweenRows_NoCrossing(t *testing.T) {
	// a-a, b-b, c-c: straight, parallel edges, no crossings.
	adj := symmetricAdj([][2]string{{"a", "a2"}, {"b", "b2"}, {"c", "c2"}})
	upper := []string{"a", "b", "c"}
	lower := []string{"a2", "b2", "c2"}

	if got := countCrossingsBetweenRows(upper, lower, adj); got != 0 {
		t.Errorf("expected 0 crossings, got %d", got)
	}
}

func TestCountCrossingsBetweenRows_OneCrossing(t *testing.T) {
	// a-b2, b-a2: the two edges cross (classic X shape).
	adj := symmetricAdj([][2]string{{"a", "b2"}, {"b", "a2"}})
	upper := []string{"a", "b"}
	lower := []string{"a2", "b2"}

	if got := countCrossingsBetweenRows(upper, lower, adj); got != 1 {
		t.Errorf("expected 1 crossing, got %d", got)
	}
}

func TestCountCrossingsBetweenRows_MultipleEdgesPerNode(t *testing.T) {
	// a connects to both a2 and b2 (fan-out) — no crossing among a's own
	// edges (they share an endpoint), only genuinely disjoint edge pairs
	// with an inverted order count.
	adj := symmetricAdj([][2]string{{"a", "a2"}, {"a", "b2"}, {"b", "a2"}})
	upper := []string{"a", "b"}
	lower := []string{"a2", "b2"}
	// Edges: (a->a2)=(0,0), (a->b2)=(0,1), (b->a2)=(1,0)
	// (a->b2) vs (b->a2): u 0<1, l 1>0 -> crosses.
	// (a->a2) vs (b->a2): share lower endpoint (l equal) -> not counted as crossing by strict > / <.
	if got := countCrossingsBetweenRows(upper, lower, adj); got != 1 {
		t.Errorf("expected 1 crossing, got %d", got)
	}
}

func TestSortByBarycenter_OrdersByNeighborPosition(t *testing.T) {
	// refRow fixes positions: x=0, y=1, z=2.
	// row's "b" neighbors only "z" (pos 2), "a" neighbors only "x" (pos 0).
	// Barycenter order should place a before b.
	adj := symmetricAdj([][2]string{{"a", "x"}, {"b", "z"}})
	refRow := []string{"x", "y", "z"}
	row := []string{"b", "a"} // deliberately "wrong" order

	got := sortByBarycenter(row, refRow, adj)
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortByBarycenter() = %v, want %v", got, want)
	}
}

func TestSortByBarycenter_NoNeighborsKeepsOriginalPosition(t *testing.T) {
	// Neither "p" nor "q" connects to anything in refRow — score falls back
	// to original index, so relative order is preserved.
	adj := symmetricAdj(nil)
	refRow := []string{"x", "y"}
	row := []string{"p", "q"}

	got := sortByBarycenter(row, refRow, adj)
	want := []string{"p", "q"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortByBarycenter() = %v, want %v (should keep original order when disconnected)", got, want)
	}
}

// TestMinimizeCrossings_ResolvesKnownCrossing constructs the classic X
// shape (#607's motivating case) and verifies the sweep actually untangles
// it, not just leaves it as-is.
func TestMinimizeCrossings_ResolvesKnownCrossing(t *testing.T) {
	adj := symmetricAdj([][2]string{{"a", "b2"}, {"b", "a2"}})
	rows := [][]string{{"a", "b"}, {"a2", "b2"}}

	result := minimizeCrossings(rows, adj)

	if got := countCrossings(result, adj); got != 0 {
		t.Errorf("expected minimizeCrossings to resolve the crossing, got %d crossings in %v", got, result)
	}
	// Row membership must be unchanged — only order within each row may move.
	for i, row := range result {
		if len(row) != len(rows[i]) {
			t.Fatalf("row %d membership changed: got %v, want same length as %v", i, row, rows[i])
		}
	}
}

// TestMinimizeCrossings_NeverWorseThanInput is the safety property that
// matters most: since barycenter sweeps aren't guaranteed to monotonically
// improve, minimizeCrossings must never return an arrangement with more
// crossings than what it was given.
func TestMinimizeCrossings_NeverWorseThanInput(t *testing.T) {
	cases := []struct {
		name string
		rows [][]string
		adj  map[string]map[string]bool
	}{
		{
			name: "already optimal",
			rows: [][]string{{"a", "b", "c"}, {"a2", "b2", "c2"}},
			adj:  symmetricAdj([][2]string{{"a", "a2"}, {"b", "b2"}, {"c", "c2"}}),
		},
		{
			name: "three rows, dense",
			rows: [][]string{{"a", "b"}, {"m", "n", "o"}, {"x", "y"}},
			adj: symmetricAdj([][2]string{
				{"a", "m"}, {"a", "o"}, {"b", "n"}, {"b", "m"},
				{"m", "x"}, {"n", "y"}, {"o", "x"}, {"o", "y"},
			}),
		},
		{
			name: "single row (no adjacent rows to cross)",
			rows: [][]string{{"a", "b", "c"}},
			adj:  symmetricAdj(nil),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := countCrossings(c.rows, c.adj)
			result := minimizeCrossings(c.rows, c.adj)
			after := countCrossings(result, c.adj)
			if after > before {
				t.Errorf("minimizeCrossings made things worse: before=%d after=%d", before, after)
			}
		})
	}
}

// TestMinimizeCrossings_PreservesRowMembership ensures the function only
// ever reorders elements within a row, never moves an element to a
// different row or drops/duplicates one — that's assignBFSRows's job, not
// minimizeCrossings's.
func TestMinimizeCrossings_PreservesRowMembership(t *testing.T) {
	rows := [][]string{{"a", "b"}, {"c", "d", "e"}, {"f"}}
	adj := symmetricAdj([][2]string{{"a", "d"}, {"b", "c"}, {"c", "f"}, {"e", "f"}})

	result := minimizeCrossings(rows, adj)

	if len(result) != len(rows) {
		t.Fatalf("row count changed: got %d, want %d", len(result), len(rows))
	}
	for i, row := range rows {
		gotSet := make(map[string]bool, len(result[i]))
		for _, id := range result[i] {
			gotSet[id] = true
		}
		if len(gotSet) != len(row) {
			t.Fatalf("row %d: duplicate or missing elements: got %v, want same set as %v", i, result[i], row)
		}
		for _, id := range row {
			if !gotSet[id] {
				t.Errorf("row %d: element %q missing from result %v", i, id, result[i])
			}
		}
	}
}
