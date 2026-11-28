package graph

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// ringModel is the model of #730: rings systems, each with services services that
// have dbs databases below them; service j of ring i uses service j of ring i+1,
// so every service number forms a cycle through all rings.
func ringModel(rings, services, dbs int) *model.BausteinsichtModel {
	m := &model.BausteinsichtModel{Model: map[string]model.Element{}}
	for i := 0; i < rings; i++ {
		children := map[string]model.Element{}
		for j := 0; j < services; j++ {
			dbChildren := map[string]model.Element{}
			for k := 0; k < dbs; k++ {
				dbChildren[fmt.Sprintf("db%d", k)] = model.Element{Kind: "db", Title: "db"}
			}
			children[fmt.Sprintf("svc%d", j)] = model.Element{Kind: "svc", Title: "svc", Children: dbChildren}
		}
		m.Model[fmt.Sprintf("sys%d", i)] = model.Element{Kind: "system", Title: "sys", Children: children}
	}
	for i := 0; i < rings; i++ {
		for j := 0; j < services; j++ {
			from := fmt.Sprintf("sys%d.svc%d", i, j)
			m.Relationships = append(m.Relationships,
				model.Relationship{From: from, To: fmt.Sprintf("sys%d.svc%d", (i+1)%rings, j)},
				model.Relationship{From: from, To: from + ".db0"})
		}
	}
	return m
}

// The analysis of a model with some hundreds of elements takes well under a
// second; it took minutes because the centrality ran two searches per pair of
// elements (#730).
func TestAnalyze_LargeModelStaysWithinBudget(t *testing.T) {
	m := ringModel(20, 10, 3) // 820 elements
	start := time.Now()
	result := NewAnalyzer(m).Analyze()
	elapsed := time.Since(start)

	if result.ElementCount != 820 {
		t.Fatalf("setup: want 820 elements, got %d", result.ElementCount)
	}
	if len(result.Cycles) != 10 {
		t.Errorf("want one cycle per service number (10), got %d", len(result.Cycles))
	}
	const budget = 2 * time.Second
	if elapsed > budget {
		t.Errorf("Analyze of %d elements took %v, budget %v", result.ElementCount, elapsed, budget)
	}
	t.Logf("Analyze of %d elements: %v", result.ElementCount, elapsed)
}

// The values of the centrality are what the definition says: betweenness is the
// share of the other elements that can be reached, closeness is 1/(1 + the mean
// shortest distance to them). A direct computation per pair of elements gives the
// reference.
func TestCentrality_MatchesTheDefinition(t *testing.T) {
	m := ringModel(4, 3, 2)
	a := NewAnalyzer(m)
	flat, _ := model.FlattenElements(m)

	byID := map[string]Centrality{}
	for _, c := range a.Analyze().Centrality {
		byID[c.ID] = c
	}
	for id := range flat {
		dist := referenceDistances(a.graph, id)
		reachable, total := 0, 0
		for target, d := range dist {
			if target != id {
				reachable++
				total += d
			}
		}
		wantBetweenness := float64(reachable) / float64(len(flat)-1)
		wantCloseness := 0.0
		if reachable > 0 {
			wantCloseness = 1.0 / (1.0 + float64(total)/float64(reachable))
		}
		got := byID[id]
		if math.Abs(got.Betweenness-wantBetweenness) > 1e-9 || math.Abs(got.Closeness-wantCloseness) > 1e-9 {
			t.Errorf("%s: want betweenness %v closeness %v, got %v %v", id, wantBetweenness, wantCloseness, got.Betweenness, got.Closeness)
		}
		if got.InDegree != len(a.reverse[id]) || got.OutDegree != len(a.graph[id]) {
			t.Errorf("%s: degrees %d/%d", id, got.InDegree, got.OutDegree)
		}
	}
}

// referenceDistances is a plain breadth-first search from src.
func referenceDistances(graph map[string][]string, src string) map[string]int {
	dist := map[string]int{src: 0}
	queue := []string{src}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range graph[cur] {
			if _, seen := dist[next]; !seen {
				dist[next] = dist[cur] + 1
				queue = append(queue, next)
			}
		}
	}
	return dist
}
