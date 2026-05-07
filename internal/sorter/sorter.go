package sorter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/salemgolemugoo/tgsort/internal/config"
)

type hclItem struct {
	text      string   // raw lines including preceding comments, no trailing newline
	blockType string   // e.g. "dependency"; empty if this is an attribute
	labels    []string // block labels, e.g. ["vpc"]
	attrName  string   // e.g. "inputs"; empty if this is a block
}

// Sort parses src as HCL, sorts top-level blocks/attributes, returns the result.
func Sort(src []byte, cfg *config.Config) ([]byte, error) {
	lines := splitLines(string(src))
	header, footer, items, err := extractItems(src, lines)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return src, nil
	}

	// Sort attributes within items listed in SortAttributesIn.
	for i, item := range items {
		if shouldSortItem(item, cfg) {
			sorted, err := sortItemAttributes(item)
			if err != nil {
				return nil, err
			}
			items[i] = sorted
		}
	}

	sorted := sortItems(items, cfg)
	return []byte(reconstruct(header, footer, sorted)), nil
}

func shouldSortItem(item hclItem, cfg *config.Config) bool {
	name := itemName(item)
	for _, n := range cfg.SortAttributesIn {
		if n == name {
			return true
		}
	}
	return false
}

// sortItemAttributes sorts key-value pairs inside a block or attribute map.
func sortItemAttributes(item hclItem) (hclItem, error) {
	src := []byte(item.text)
	f, diags := hclsyntax.ParseConfig(src, "item.hcl", hcl.Pos{Line: 1, Column: 1, Byte: 0})
	if diags.HasErrors() {
		return item, nil // skip silently if item text can't be parsed independently
	}
	body := f.Body.(*hclsyntax.Body)

	if item.attrName != "" {
		// It's an attribute like `inputs = { ... }` — sort the object literal keys.
		attr, ok := body.Attributes[item.attrName]
		if !ok {
			return item, nil
		}
		objExpr, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			return item, nil // not a literal object, skip
		}
		sorted, err := sortObjectConsExpr(item.text, objExpr)
		if err != nil {
			return item, err
		}
		item.text = sorted
		return item, nil
	}

	// It's a block like `locals { ... }` — sort attributes inside the block body.
	if len(body.Blocks) == 0 {
		return item, nil
	}
	block := body.Blocks[0]
	sorted, err := sortBlockAttributes(item.text, block.Body)
	if err != nil {
		return item, err
	}
	item.text = sorted
	return item, nil
}

// sortObjectConsExpr sorts the key-value pairs of an object literal alphabetically by key.
func sortObjectConsExpr(text string, expr *hclsyntax.ObjectConsExpr) (string, error) {
	if len(expr.Items) == 0 {
		return text, nil
	}

	// Single-line object: can't meaningfully sort in a line-based approach; leave unchanged.
	if expr.SrcRange.Start.Line == expr.SrcRange.End.Line {
		return text, nil
	}

	itemLines := splitLines(text)

	type kvSegment struct {
		key   string
		lines []string
	}

	// Build segments: each segment covers from commentStart of the key line
	// to one line before the next key's commentStart (or the closing line).
	// expr.SrcRange.End.Line is the line of '}' (1-indexed).
	closingLine := expr.SrcRange.End.Line - 1 // convert to 0-indexed

	var segments []kvSegment
	for i, kv := range expr.Items {
		keyVal, diags := kv.KeyExpr.Value(nil)
		if diags.HasErrors() {
			return text, fmt.Errorf("evaluating object key: %s", diags.Error())
		}
		keyStr := keyVal.AsString()

		segStart := commentStart(itemLines, kv.KeyExpr.StartRange().Start.Line)

		var segEnd int
		if i+1 < len(expr.Items) {
			segEnd = commentStart(itemLines, expr.Items[i+1].KeyExpr.StartRange().Start.Line) - 1
		} else {
			segEnd = closingLine - 1
		}

		segLines := itemLines[segStart : segEnd+1]
		// Strip trailing blank lines from each segment.
		for len(segLines) > 0 && strings.TrimSpace(segLines[len(segLines)-1]) == "" {
			segLines = segLines[:len(segLines)-1]
		}
		segments = append(segments, kvSegment{key: keyStr, lines: segLines})
	}

	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].key < segments[j].key
	})

	// Reconstruct: opening line + sorted segments + closing line.
	// Insert a blank line before a segment that starts with a comment.
	openLine := itemLines[expr.SrcRange.Start.Line-1] // the `inputs = {` line
	closeLine := itemLines[closingLine]               // the `}` line

	var sb strings.Builder
	sb.WriteString(openLine)
	sb.WriteString("\n")
	for i, seg := range segments {
		if i > 0 && len(seg.lines) > 0 && strings.HasPrefix(strings.TrimSpace(seg.lines[0]), "#") {
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Join(seg.lines, "\n"))
		sb.WriteString("\n")
	}
	sb.WriteString(closeLine)
	return sb.String(), nil
}

// sortBlockAttributes sorts attributes within a block body alphabetically by name.
func sortBlockAttributes(text string, body *hclsyntax.Body) (string, error) {
	if len(body.Attributes) == 0 || len(body.Blocks) > 0 {
		return text, nil
	}

	itemLines := splitLines(text)

	type attrSegment struct {
		name  string
		lines []string
	}

	// Collect attributes sorted by startLine to find correct segment boundaries.
	type attrNode struct {
		name      string
		startLine int
		endLine   int
	}
	var nodes []attrNode
	for name, a := range body.Attributes {
		nodes = append(nodes, attrNode{
			name:      name,
			startLine: a.NameRange.Start.Line,
			endLine:   a.SrcRange.End.Line,
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].startLine < nodes[j].startLine
	})

	closingLine := body.SrcRange.End.Line - 1 // 0-indexed closing brace line

	var segments []attrSegment
	for i, n := range nodes {
		segStart := commentStart(itemLines, n.startLine)
		var segEnd int
		if i+1 < len(nodes) {
			segEnd = commentStart(itemLines, nodes[i+1].startLine) - 1
		} else {
			segEnd = closingLine - 1
		}
		segLines := itemLines[segStart : segEnd+1]
		for len(segLines) > 0 && strings.TrimSpace(segLines[len(segLines)-1]) == "" {
			segLines = segLines[:len(segLines)-1]
		}
		segments = append(segments, attrSegment{name: n.name, lines: segLines})
	}

	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].name < segments[j].name
	})

	// The block header is everything from line 0 to the first attribute's commentStart.
	firstAttrStart := commentStart(itemLines, nodes[0].startLine)
	header := strings.Join(itemLines[:firstAttrStart], "\n")
	closeLine := itemLines[closingLine]

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	for i, seg := range segments {
		if i > 0 && len(seg.lines) > 0 && strings.HasPrefix(strings.TrimSpace(seg.lines[0]), "#") {
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Join(seg.lines, "\n"))
		sb.WriteString("\n")
	}
	sb.WriteString(closeLine)
	return sb.String(), nil
}

// splitLines splits s on "\n", dropping the trailing empty element when s ends with "\n".
func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type nodeInfo struct {
	startLine int // 1-indexed line of the block type keyword or attribute name
	endLine   int // 1-indexed line of the closing brace or end of value
	blockType string
	labels    []string
	attrName  string
}

func extractItems(src []byte, lines []string) (header, footer string, items []hclItem, err error) {
	f, diags := hclsyntax.ParseConfig(src, "file.hcl", hcl.Pos{Line: 1, Column: 1, Byte: 0})
	if diags.HasErrors() {
		return "", "", nil, fmt.Errorf("%s", diags.Error())
	}
	body := f.Body.(*hclsyntax.Body)

	var nodes []nodeInfo
	for _, b := range body.Blocks {
		nodes = append(nodes, nodeInfo{
			startLine: b.TypeRange.Start.Line,
			endLine:   b.CloseBraceRange.End.Line,
			blockType: b.Type,
			labels:    b.Labels,
		})
	}
	for name, a := range body.Attributes {
		nodes = append(nodes, nodeInfo{
			startLine: a.NameRange.Start.Line,
			endLine:   a.SrcRange.End.Line,
			attrName:  name,
		})
	}
	if len(nodes) == 0 {
		return string(src), "", nil, nil
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].startLine < nodes[j].startLine
	})

	// Determine where each item truly starts (scanning back for comment lines).
	firstItemStart := commentStart(lines, nodes[0].startLine)
	header = strings.Join(lines[:firstItemStart], "\n")
	if header != "" {
		header = strings.TrimRight(header, " \t")
	}

	// Capture footer: everything after the last node's end line.
	lastEndLine := nodes[len(nodes)-1].endLine // 1-indexed
	if lastEndLine < len(lines) {
		footer = strings.TrimLeft(strings.Join(lines[lastEndLine:], "\n"), "\n")
	}

	for _, n := range nodes {
		cs := commentStart(lines, n.startLine)
		end := n.endLine - 1 // convert to 0-indexed
		text := strings.Join(lines[cs:end+1], "\n")
		items = append(items, hclItem{
			text:      text,
			blockType: n.blockType,
			labels:    n.labels,
			attrName:  n.attrName,
		})
	}
	return header, footer, items, nil
}

// commentStart returns the 0-indexed line where this item's text begins,
// which may be above startLine if there are comment lines immediately preceding it.
// It stops scanning when it hits a blank line or a non-comment line.
func commentStart(lines []string, startLine int) int {
	idx := startLine - 1 // convert 1-indexed to 0-indexed
	for idx > 0 {
		prev := strings.TrimSpace(lines[idx-1])
		if strings.HasPrefix(prev, "#") || strings.HasPrefix(prev, "//") {
			idx--
		} else {
			break
		}
	}
	return idx
}

func itemName(item hclItem) string {
	if item.attrName != "" {
		return item.attrName
	}
	return item.blockType
}

func itemPriority(item hclItem, cfg *config.Config) int {
	name := itemName(item)
	for i, bt := range cfg.BlockOrder {
		if bt == name {
			return i
		}
	}
	return len(cfg.BlockOrder)
}

func itemSortKey(item hclItem) string {
	if len(item.labels) > 0 {
		return item.labels[0]
	}
	return ""
}

func sortItems(items []hclItem, cfg *config.Config) []hclItem {
	out := make([]hclItem, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := itemPriority(out[i], cfg), itemPriority(out[j], cfg)
		if pi != pj {
			return pi < pj
		}
		// Same priority: for unlisted blocks, sort by block type name first.
		ni, nj := itemName(out[i]), itemName(out[j])
		if ni != nj {
			return ni < nj
		}
		// Same type: sort by first label.
		return itemSortKey(out[i]) < itemSortKey(out[j])
	})
	return out
}

func reconstruct(header, footer string, items []hclItem) string {
	var parts []string
	if header != "" {
		parts = append(parts, strings.TrimRight(header, "\n"))
	}
	for _, item := range items {
		parts = append(parts, item.text)
	}
	result := strings.Join(parts, "\n\n") + "\n"
	if footer != "" {
		result += "\n" + footer + "\n"
	}
	return result
}
