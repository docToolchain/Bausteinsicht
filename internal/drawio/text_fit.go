package drawio

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// Text metrics for the estimate of the number of lines a text needs. They are
// averages of the fonts draw.io uses: a Latin character is about 0.55 of the
// font size wide (0.6 in bold), a CJK character or an emoji a whole em.
const (
	latinCharWidth = 0.55
	boldCharWidth  = 0.6
	wideCharWidth  = 1.0
	spaceWidth     = 0.3
	lineHeightEm   = 1.2
	textPadding    = 4.0 // top and bottom space of a text cell
	sideMargin     = 8.0 // left and right space of a text cell
)

var fontSizeRe = regexp.MustCompile(`(?:^|;)fontSize=([0-9.]+)`)
var fontStyleRe = regexp.MustCompile(`(?:^|;)fontStyle=([0-9]+)`)

// FitElementText makes the title and technology cells of an element hold their
// text (#718). The cells wrap (whiteSpace=wrap) and, where the wrapped text is
// taller than the cell, the cell grows; the cells below move down by the same
// amount and the shape grows with them. Cells never shrink, so the call can be
// repeated and a shape the user has enlarged stays as it is. The description
// cell keeps its size: its text is shortened and the full text is in the
// tooltip.
func (p *Page) FitElementText(obj *etree.Element) {
	root := p.Root()
	if root == nil || obj == nil {
		return
	}
	cells := findChildTextCells(root, obj.SelectAttrValue("id", ""))
	if len(cells) == 0 {
		return
	}
	ordered := make([]*etree.Element, 0, len(cells))
	for _, role := range []string{"title", "tech", "desc"} {
		if c, ok := cells[role]; ok {
			ordered = append(ordered, c)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return geomNum(ordered[i], "y") < geomNum(ordered[j], "y") })

	shift := 0.0
	for _, c := range ordered {
		geom := c.SelectElement("mxGeometry")
		if geom == nil {
			continue
		}
		y, h, w := geomNum(c, "y"), geomNum(c, "height"), geomNum(c, "width")
		grows := strings.HasSuffix(c.SelectAttrValue("id", ""), "-title") || strings.HasSuffix(c.SelectAttrValue("id", ""), "-tech")
		if grows {
			style := c.SelectAttrValue("style", "")
			if updated := setStyleFlags(style, "whiteSpace=wrap"); updated != style {
				setAttr(c, "style", updated)
			}
			if need := neededTextHeight(CellText(c), c.SelectAttrValue("style", ""), w); need > h {
				shift += need - h
				h = need
			}
		}
		if shift != 0 {
			geom.CreateAttr("y", formatFloat(y+shift-(h-geomNum(c, "height"))))
			geom.CreateAttr("height", formatFloat(h))
		}
	}
	if shift == 0 {
		return
	}
	if parent := obj.FindElement("mxCell/mxGeometry"); parent != nil {
		ph, _ := strconv.ParseFloat(parent.SelectAttrValue("height", "0"), 64)
		parent.CreateAttr("height", formatFloat(ph+shift))
	}
}

func geomNum(cell *etree.Element, name string) float64 {
	geom := cell.SelectElement("mxGeometry")
	if geom == nil {
		return 0
	}
	v, _ := strconv.ParseFloat(geom.SelectAttrValue(name, "0"), 64)
	return v
}

// neededTextHeight estimates the height of text in a cell of the given width
// with the given style: the lines of a greedy word wrap, times the line height.
func neededTextHeight(text, style string, width float64) float64 {
	if strings.TrimSpace(text) == "" || width <= 0 {
		return 0
	}
	size := 12.0
	if m := fontSizeRe.FindStringSubmatch(style); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 0 {
			size = v
		}
	}
	bold := false
	if m := fontStyleRe.FindStringSubmatch(style); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			bold = v&1 != 0
		}
	}
	avail := math.Max(width-sideMargin, size)
	lines := wrappedLines(text, size, bold, avail)
	return float64(lines)*size*lineHeightEm + textPadding
}

// wrappedLines counts the lines text needs in avail pixels: words wrap, a word
// wider than a line counts for as many lines as it needs.
func wrappedLines(text string, size float64, bold bool, avail float64) int {
	lines := 0
	for _, paragraph := range strings.Split(text, "\n") {
		lines++
		used := 0.0
		for _, word := range strings.Fields(paragraph) {
			w := textWidth(word, size, bold)
			switch {
			case used == 0:
				used = w
			case used+size*spaceWidth+w <= avail:
				used += size*spaceWidth + w
			default:
				lines++
				used = w
			}
			if used > avail { // a word longer than a line
				extra := int(math.Ceil(used/avail)) - 1
				lines += extra
				used = math.Mod(used, avail)
			}
		}
	}
	return lines
}

func textWidth(s string, size float64, bold bool) float64 {
	narrow := latinCharWidth
	if bold {
		narrow = boldCharWidth
	}
	total := 0.0
	for _, r := range s {
		if r >= 0x1100 {
			total += size * wideCharWidth
		} else {
			total += size * narrow
		}
	}
	return total
}
