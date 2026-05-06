package diagram

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// HTMLNode represents a node in the interactive diagram.
type HTMLNode struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Kind        string  `json:"kind"`
	Description string  `json:"description,omitempty"`
	Technology  string  `json:"technology,omitempty"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Fill        string  `json:"fill"`
	Stroke      string  `json:"stroke"`
}

// HTMLEdge is a type alias for relEntry used in HTML diagram output.
type HTMLEdge = relEntry

// HTMLDiagramData is the data structure embedded in the HTML output.
type HTMLDiagramData struct {
	Title string     `json:"title"`
	Nodes []HTMLNode `json:"nodes"`
	Edges []HTMLEdge `json:"edges"`
}

// RenderHTML renders a view as an interactive HTML5 diagram.
func RenderHTML(m *model.BausteinsichtModel, viewKey string) (string, error) {
	view, ok := m.Views[viewKey]
	if !ok {
		return "", fmt.Errorf("view %q not found", viewKey)
	}

	resolved, err := model.ResolveView(m, &view)
	if err != nil {
		return "", err
	}

	flat, _ := model.FlattenElements(m)
	sort.Strings(resolved)

	// Filter elements visible in this view
	elemSet := make(map[string]bool, len(resolved))
	for _, id := range resolved {
		elemSet[id] = true
	}
	if view.Scope != "" {
		elemSet[view.Scope] = true
	}

	// Filter relationships
	rels := filterRelationships(m.Relationships, elemSet)

	// Build node list with simple grid layout
	nodes := []HTMLNode{}
	x, y := 50.0, 50.0
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}

		style := ColorForKind(elem.Kind)
		title := elem.Title
		if title == "" {
			title = id
		}

		nodes = append(nodes, HTMLNode{
			ID:          id,
			Title:       title,
			Kind:        elem.Kind,
			Description: elem.Description,
			Technology:  elem.Technology,
			X:           x,
			Y:           y,
			Fill:        style.Fill,
			Stroke:      style.Stroke,
		})

		x += 200
		if x > 800 {
			x = 50
			y += 150
		}
	}

	// Build edge list from relationships
	edges := make([]HTMLEdge, len(rels))
	for i, r := range rels {
		edges[i] = HTMLEdge(r)
	}

	// Create diagram data
	data := HTMLDiagramData{
		Title: view.Title,
		Nodes: nodes,
		Edges: edges,
	}

	dataJSON, _ := json.Marshal(data)

	// Generate HTML with embedded JavaScript renderer (escape title for HTML safety)
	htmlContent := generateHTMLTemplate(html.EscapeString(view.Title), string(dataJSON))
	return htmlContent, nil
}

func generateHTMLTemplate(title, dataJSON string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Architecture — %s</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; }

    #header {
      background: white;
      padding: 16px;
      border-bottom: 1px solid #ddd;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }

    #header h1 { font-size: 20px; font-weight: 600; color: #333; }

    #controls {
      display: flex;
      gap: 12px;
      margin-top: 12px;
      align-items: center;
    }

    input[type="text"] {
      flex: 1;
      max-width: 300px;
      padding: 8px 12px;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 14px;
    }

    button {
      padding: 8px 16px;
      background: #0066cc;
      color: white;
      border: none;
      border-radius: 4px;
      font-size: 14px;
      cursor: pointer;
    }

    button:hover { background: #0052a3; }

    #canvas {
      flex: 1;
      background: white;
      position: relative;
      overflow: auto;
    }

    svg { display: block; }

    #details {
      width: 300px;
      background: white;
      border-left: 1px solid #ddd;
      padding: 16px;
      overflow-y: auto;
      display: none;
    }

    #details.show { display: block; }

    #details h3 { font-size: 16px; font-weight: 600; color: #333; margin-bottom: 12px; }
    #details p { margin: 8px 0; font-size: 13px; color: #666; word-break: break-word; }
    #details strong { color: #333; }

    .grid { display: flex; }
    #canvas { flex: 1; }

    .node { cursor: pointer; transition: opacity 0.2s; }
    .node:hover { opacity: 0.8; }
    .node.highlighted { filter: drop-shadow(0 0 4px #0066cc); }
    .node.faded { opacity: 0.3; }

    .edge { stroke-width: 2; fill: none; marker-end: url(#arrowhead); }
    .edge.highlighted { stroke: #0066cc; stroke-width: 3; }
    .edge.faded { opacity: 0.2; }

    .edge-label { font-size: 12px; fill: #333; pointer-events: none; }
  </style>
</head>
<body>
  <div id="header">
    <h1>Architecture Diagram</h1>
    <div id="controls">
      <input type="text" id="searchInput" placeholder="Search elements..." />
      <button onclick="resetZoom()">Reset View</button>
    </div>
  </div>

  <div class="grid">
    <div id="canvas"></div>
    <div id="details">
      <h3 id="detailsTitle"></h3>
      <p><strong>Kind:</strong> <span id="detailsKind"></span></p>
      <p id="detailsTech" style="display:none;"><strong>Technology:</strong> <span id="detailsTechVal"></span></p>
      <p id="detailsDesc" style="display:none;"><strong>Description:</strong> <span id="detailsDescVal"></span></p>
    </div>
  </div>

  <script>
    const DIAGRAM_DATA = %s;

    const state = {
      zoom: 1,
      pan: { x: 0, y: 0 },
      selected: null,
      search: ""
    };

    function initDiagram() {
      const canvas = document.getElementById('canvas');
      const width = canvas.clientWidth;
      const height = canvas.clientHeight;

      const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      svg.setAttribute('width', width);
      svg.setAttribute('height', height);

      // Add arrowhead marker definition
      const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
      const marker = document.createElementNS('http://www.w3.org/2000/svg', 'marker');
      marker.setAttribute('id', 'arrowhead');
      marker.setAttribute('markerWidth', '10');
      marker.setAttribute('markerHeight', '10');
      marker.setAttribute('refX', '9');
      marker.setAttribute('refY', '3');
      marker.setAttribute('orient', 'auto');
      const poly = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
      poly.setAttribute('points', '0 0, 10 3, 0 6');
      poly.setAttribute('fill', '#333');
      marker.appendChild(poly);
      defs.appendChild(marker);
      svg.appendChild(defs);

      // Draw edges first (background)
      for (const edge of DIAGRAM_DATA.edges) {
        const fromNode = DIAGRAM_DATA.nodes.find(n => n.id === edge.from);
        const toNode = DIAGRAM_DATA.nodes.find(n => n.id === edge.to);
        if (fromNode && toNode) {
          const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
          line.setAttribute('x1', fromNode.x + 80);
          line.setAttribute('y1', fromNode.y + 40);
          line.setAttribute('x2', toNode.x + 80);
          line.setAttribute('y2', toNode.y + 40);
          line.setAttribute('stroke', '#999');
          line.setAttribute('stroke-width', '2');
          line.setAttribute('marker-end', 'url(#arrowhead)');
          line.classList.add('edge');
          line.dataset.from = edge.from;
          line.dataset.to = edge.to;
          svg.appendChild(line);

          // Label
          if (edge.label) {
            const mid = {
              x: (fromNode.x + toNode.x) / 2 + 80,
              y: (fromNode.y + toNode.y) / 2 + 40
            };
            const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
            text.setAttribute('x', mid.x);
            text.setAttribute('y', mid.y - 5);
            text.setAttribute('text-anchor', 'middle');
            text.setAttribute('font-size', '12');
            text.textContent = edge.label;
            text.classList.add('edge-label');
            svg.appendChild(text);
          }
        }
      }

      // Draw nodes
      for (const node of DIAGRAM_DATA.nodes) {
        const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
        g.classList.add('node');
        g.dataset.id = node.id;

        const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
        rect.setAttribute('x', node.x);
        rect.setAttribute('y', node.y);
        rect.setAttribute('width', '160');
        rect.setAttribute('height', '80');
        rect.setAttribute('fill', node.fill);
        rect.setAttribute('stroke', node.stroke);
        rect.setAttribute('stroke-width', '2');
        rect.setAttribute('rx', '4');

        const title = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        title.setAttribute('x', node.x + 80);
        title.setAttribute('y', node.y + 25);
        title.setAttribute('text-anchor', 'middle');
        title.setAttribute('font-weight', 'bold');
        title.setAttribute('font-size', '13');
        title.textContent = node.title.substring(0, 20);

        const kind = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        kind.setAttribute('x', node.x + 80);
        kind.setAttribute('y', node.y + 50);
        kind.setAttribute('text-anchor', 'middle');
        kind.setAttribute('font-size', '11');
        kind.setAttribute('fill', '#666');
        kind.textContent = '[' + node.kind + ']';

        g.appendChild(rect);
        g.appendChild(title);
        g.appendChild(kind);

        g.onclick = () => selectNode(node);
        svg.appendChild(g);
      }

      canvas.appendChild(svg);

      // Search
      document.getElementById('searchInput').addEventListener('input', (e) => {
        state.search = e.target.value.toLowerCase();
        highlightSearch();
      });
    }

    function selectNode(node) {
      state.selected = node.id;
      updateDetails(node);
      highlightNode(node.id);
    }

    function updateDetails(node) {
      const details = document.getElementById('details');
      document.getElementById('detailsTitle').textContent = node.title;
      document.getElementById('detailsKind').textContent = node.kind;

      const techEl = document.getElementById('detailsTech');
      if (node.technology) {
        techEl.style.display = 'block';
        document.getElementById('detailsTechVal').textContent = node.technology;
      } else {
        techEl.style.display = 'none';
      }

      const descEl = document.getElementById('detailsDesc');
      if (node.description) {
        descEl.style.display = 'block';
        document.getElementById('detailsDescVal').textContent = node.description;
      } else {
        descEl.style.display = 'none';
      }

      details.classList.add('show');
    }

    function highlightNode(nodeId) {
      document.querySelectorAll('.node').forEach(el => {
        if (el.dataset.id === nodeId) {
          el.classList.add('highlighted');
        } else {
          el.classList.remove('highlighted');
        }
      });
    }

    function highlightSearch() {
      if (!state.search) {
        document.querySelectorAll('.node, .edge').forEach(el => el.classList.remove('faded'));
        return;
      }

      document.querySelectorAll('.node').forEach(el => {
        const nodeId = el.dataset.id;
        const node = DIAGRAM_DATA.nodes.find(n => n.id === nodeId);
        const matches = node && (node.id.toLowerCase().includes(state.search) || node.title.toLowerCase().includes(state.search));
        el.classList.toggle('faded', !matches);
      });

      document.querySelectorAll('.edge').forEach(el => {
        const from = el.dataset.from;
        const to = el.dataset.to;
        const matches = from.toLowerCase().includes(state.search) || to.toLowerCase().includes(state.search);
        el.classList.toggle('faded', !matches);
      });
    }

    function resetZoom() {
      state.zoom = 1;
      state.pan = { x: 0, y: 0 };
      state.selected = null;
      document.getElementById('details').classList.remove('show');
      document.querySelectorAll('.node').forEach(el => el.classList.remove('highlighted', 'faded'));
      document.getElementById('searchInput').value = '';
    }

    window.addEventListener('load', initDiagram);
  </script>
</body>
</html>`, title, dataJSON)
}
