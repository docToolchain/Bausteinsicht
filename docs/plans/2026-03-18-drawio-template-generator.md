# Plan: draw.io Template Generator

## Purpose

Automatically generate a draw.io template file from `spec.elementKinds`. Currently, teams must manually maintain a template file defining shapes, colors, and styles for each element kind. This command generates a ready-to-use template directly from the spec, eliminating manual maintenance and ensuring templates stay in sync with the model spec.

## CLI Interface

```
bausteinsicht generate-template [--model <file>] [--output <template-file>] [--style <style-preset>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `architecture-template.drawio` | Output template file |
| `--style` | `default` | Visual preset: `default`, `c4`, `minimal`, `dark` |

### Example

```bash
bausteinsicht generate-template --style c4 --output templates/my-template.drawio
# → Generated template with shapes for: person, system, container, component, database
```

## Generated Template Structure

A draw.io template is a `.drawio` file (XML) containing one example element per defined kind, arranged in a grid layout. Each element:
- Uses the kind's label as the shape label
- Applies the standard Bausteinsicht fill/stroke colors for that kind
- Includes a subtitle showing the kind name in brackets
- Has standard dimensions (120×60px for services, 60×80px for persons, etc.)

### Example Generated XML (excerpt)

```xml
<mxGraphModel>
  <root>
    <mxCell id="0"/><mxCell id="1" parent="0"/>

    <!-- person kind -->
    <mxCell id="2" value="&lt;b&gt;Person&lt;/b&gt;&lt;br/&gt;[person]"
      style="shape=mxgraph.archimate3.actor;fillColor=#dae8fc;strokeColor=#6c8ebf;fontStyle=1;"
      vertex="1" parent="1">
      <mxGeometry x="40" y="40" width="60" height="80" as="geometry"/>
    </mxCell>

    <!-- service kind -->
    <mxCell id="3" value="&lt;b&gt;Service Name&lt;/b&gt;&lt;br/&gt;[service]"
      style="rounded=1;fillColor=#d5e8d4;strokeColor=#82b366;"
      vertex="1" parent="1">
      <mxGeometry x="160" y="40" width="120" height="60" as="geometry"/>
    </mxCell>

    <!-- database kind -->
    <mxCell id="4" value="&lt;b&gt;Database&lt;/b&gt;&lt;br/&gt;[database]"
      style="shape=mxgraph.flowchart.database;fillColor=#fff2cc;strokeColor=#d6b656;"
      vertex="1" parent="1">
      <mxGeometry x="340" y="40" width="60" height="80" as="geometry"/>
    </mxCell>
  </root>
</mxGraphModel>
```

## Style Presets

### `default`
Standard Bausteinsicht palette — soft colors, rounded rectangles, archimate person shape.

### `c4`
C4 model visual style — blue system boxes, grey person shapes, dashed external system borders.

### `minimal`
White fill, thin grey borders, no icons — optimized for black-and-white printing.

### `dark`
Dark background (`#1e1e1e`), bright fills — suitable for dark-mode presentations.

## Kind → Shape Mapping

```go
var KindShapes = map[string]ShapeConfig{
    "person":    { Shape: "mxgraph.archimate3.actor", Width: 60,  Height: 80 },
    "system":    { Shape: "rounded=1",                Width: 160, Height: 60 },
    "service":   { Shape: "rounded=1",                Width: 120, Height: 60 },
    "container": { Shape: "rounded=1;container=1",    Width: 200, Height: 120 },
    "database":  { Shape: "mxgraph.flowchart.database", Width: 60, Height: 80 },
    "cache":     { Shape: "mxgraph.flowchart.stored_data", Width: 80, Height: 60 },
    "frontend":  { Shape: "rounded=1",                Width: 120, Height: 60 },
    "infra":     { Shape: "mxgraph.cisco.servers.standard_server", Width: 60, Height: 80 },
}
// Unknown kinds fall back to: rounded=1, 120×60
```

Custom shapes can be defined in `spec.elementKinds` via an optional `shape` field.

## Integration with `init`

`bausteinsicht init` gains a `--generate-template` flag:

```bash
bausteinsicht init --generate-template
# → Creates architecture.jsonc + architecture-template.drawio (generated from spec)
```

## Architecture

### New / Modified Files

| File | Change |
|------|--------|
| `internal/template/generator.go` | New: `Generate(spec Specification, style string) string` (returns draw.io XML) |
| `internal/template/shapes.go` | New: kind→shape mapping + preset color tables |
| `internal/template/layout.go` | New: grid layout algorithm for element placement |
| `cmd/bausteinsicht/generate_template.go` | New `generate-template` command |
| `cmd/bausteinsicht/init.go` | Extend with optional `--generate-template` flag |

## Testing

- Unit test: generated XML is valid draw.io format (parse with etree)
- Unit test: all defined kinds appear in output
- Unit test: unknown kind falls back to default shape
- Unit test: all style presets produce different fill colors
- E2E test: `generate-template` → open in draw.io validation (XML schema check)
- Test: kinds defined with custom `shape` field use that shape in output
