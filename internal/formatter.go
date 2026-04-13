package internal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// strikethroughDelimiter implements parser.DelimiterProcessor for ~~ strikethrough.
type strikethroughDelimiter struct{}

func (p *strikethroughDelimiter) IsDelimiter(b byte) bool                   { return b == '~' }
func (p *strikethroughDelimiter) CanOpenCloser(o, c *parser.Delimiter) bool { return o.Char == c.Char }
func (p *strikethroughDelimiter) OnMatch(int) ast.Node                      { return extast.NewStrikethrough() }

// doubleStrikethroughParser requires ~~ (not single ~) for strikethrough.
type doubleStrikethroughParser struct{}

func (s *doubleStrikethroughParser) Trigger() []byte { return []byte{'~'} }

func (s *doubleStrikethroughParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, &strikethroughDelimiter{})
	if node == nil || node.OriginalLength > 2 || before == '~' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *doubleStrikethroughParser) CloseBlock(parent ast.Node, pc parser.Context) {}

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Linkify,
		extension.Table,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithInlineParsers(
			util.Prioritized(&doubleStrikethroughParser{}, 500),
		),
	),
	goldmark.WithRendererOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(extension.NewStrikethroughHTMLRenderer(), 500),
		),
	),
)

var multipleNewlines = regexp.MustCompile(`\n{3,}`)

// FormatTelegramHTML converts Markdown text to Telegram-compatible HTML.
func FormatTelegramHTML(markdown string) string {
	if markdown == "" {
		return ""
	}

	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var buf strings.Builder
	var orderedIndices []int // stack of ordered list counters per nesting level
	var listDepths []bool    // stack of isOrdered for each list level

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Document:
			// no-op

		case *ast.Paragraph:
			if !entering {
				// Don't add trailing newlines if parent is a list item
				// (list item handles its own spacing)
				if _, ok := node.Parent().(*ast.ListItem); !ok {
					buf.WriteString("\n")
					if node.NextSibling() != nil {
						buf.WriteString("\n")
					}
				}
			}

		case *ast.Heading:
			if entering {
				buf.WriteString("<b>")
			} else {
				buf.WriteString("</b>\n")
				if node.NextSibling() != nil {
					// Only add blank line before another heading or blockquote,
					// not before lists or paragraphs (keep headings tight with their content)
					switch node.NextSibling().(type) {
					case *ast.Heading, *ast.Blockquote, *ast.ThematicBreak:
						buf.WriteString("\n")
					}
				}
			}

		case *ast.ThematicBreak:
			if entering {
				buf.WriteString("———\n")
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}

		case *ast.CodeBlock:
			if entering {
				buf.WriteString("<pre>")
				writeCodeBlockLines(&buf, node, source)
				buf.WriteString("</pre>\n")
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}
			return ast.WalkSkipChildren, nil

		case *ast.FencedCodeBlock:
			if entering {
				lang := string(node.Language(source))
				if lang != "" {
					fmt.Fprintf(&buf, `<pre><code class="language-%s">`, escapeHTML(lang))
					writeCodeBlockLines(&buf, node, source)
					buf.WriteString("</code></pre>\n")
				} else {
					buf.WriteString("<pre>")
					writeCodeBlockLines(&buf, node, source)
					buf.WriteString("</pre>\n")
				}
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}
			return ast.WalkSkipChildren, nil

		case *ast.Blockquote:
			if entering {
				buf.WriteString("<blockquote>")
			} else {
				// Trim trailing whitespace inside blockquote
				trimTrailingNewlines(&buf)
				buf.WriteString("</blockquote>\n")
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}

		case *ast.List:
			if entering {
				listDepths = append(listDepths, node.IsOrdered())
				start := node.Start
				if start == 0 {
					start = 1
				}
				orderedIndices = append(orderedIndices, start)
			} else {
				listDepths = listDepths[:len(listDepths)-1]
				orderedIndices = orderedIndices[:len(orderedIndices)-1]
				// Add blank line after top-level list
				if len(listDepths) == 0 {
					if node.NextSibling() != nil {
						buf.WriteString("\n")
					}
				}
			}

		case *ast.ListItem:
			if entering {
				depth := len(listDepths) - 1
				indent := strings.Repeat("    ", depth)
				buf.WriteString(indent)

				// Check if this item has a task checkbox — if so, skip the bullet
				hasCheckbox := false
				for c := node.FirstChild(); c != nil; c = c.NextSibling() {
					for gc := c.FirstChild(); gc != nil; gc = gc.NextSibling() {
						if _, ok := gc.(*extast.TaskCheckBox); ok {
							hasCheckbox = true
							break
						}
					}
					if hasCheckbox {
						break
					}
				}

				if !hasCheckbox {
					if listDepths[len(listDepths)-1] {
						// Ordered list
						fmt.Fprintf(&buf, "%d. ", orderedIndices[len(orderedIndices)-1])
						orderedIndices[len(orderedIndices)-1]++
					} else {
						// Unordered list
						if depth == 0 {
							buf.WriteString("• ")
						} else {
							buf.WriteString("◦ ")
						}
					}
				}
			} else {
				// Avoid double newline when list item contained a nested list
				s := buf.String()
				if len(s) == 0 || s[len(s)-1] != '\n' {
					buf.WriteString("\n")
				}
			}

		case *extast.TaskCheckBox:
			if entering {
				if node.IsChecked {
					buf.WriteString("✅ ")
				} else {
					buf.WriteString("⬜ ")
				}
			}

		case *ast.Text:
			if entering {
				buf.WriteString(escapeHTML(string(node.Segment.Value(source))))
				if node.SoftLineBreak() {
					buf.WriteString("\n")
				}
				if node.HardLineBreak() {
					buf.WriteString("\n")
				}
			}

		case *ast.String:
			if entering {
				buf.WriteString(escapeHTML(string(node.Value)))
			}

		case *ast.CodeSpan:
			if entering {
				buf.WriteString("<code>")
			} else {
				buf.WriteString("</code>")
			}

		case *ast.Emphasis:
			if entering {
				if node.Level == 2 {
					buf.WriteString("<b>")
				} else {
					buf.WriteString("<i>")
				}
			} else {
				if node.Level == 2 {
					buf.WriteString("</b>")
				} else {
					buf.WriteString("</i>")
				}
			}

		case *extast.Strikethrough:
			if entering {
				buf.WriteString("<s>")
			} else {
				buf.WriteString("</s>")
			}

		case *ast.Link:
			if entering {
				fmt.Fprintf(&buf, `<a href="%s">`, escapeHTML(string(node.Destination)))
			} else {
				buf.WriteString("</a>")
			}

		case *ast.AutoLink:
			if entering {
				url := string(node.URL(source))
				label := string(node.Label(source))
				fmt.Fprintf(&buf, `<a href="%s">%s</a>`, escapeHTML(url), escapeHTML(label))
			}

		case *ast.Image:
			if entering {
				dest := string(node.Destination)
				// Collect alt text from children
				alt := extractText(node, source)
				if alt == "" {
					alt = dest
				}
				fmt.Fprintf(&buf, `<a href="%s">%s</a>`, escapeHTML(dest), escapeHTML(alt))
			}
			return ast.WalkSkipChildren, nil

		case *ast.RawHTML:
			if entering {
				for i := range node.Segments.Len() {
					seg := node.Segments.At(i)
					buf.WriteString(escapeHTML(string(seg.Value(source))))
				}
			}

		case *ast.HTMLBlock:
			if entering {
				for i := range node.Lines().Len() {
					seg := node.Lines().At(i)
					buf.WriteString(escapeHTML(string(seg.Value(source))))
				}
				if node.HasClosure() {
					buf.WriteString(escapeHTML(string(node.ClosureLine.Value(source))))
				}
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}

		case *ast.TextBlock:
			if !entering {
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}

		case *extast.Table:
			if entering {
				renderTable(&buf, node, source)
				if node.NextSibling() != nil {
					buf.WriteString("\n")
				}
			}
			return ast.WalkSkipChildren, nil

		case *extast.TableHeader, *extast.TableRow, *extast.TableCell:
			// handled by renderTable
		}

		return ast.WalkContinue, nil
	})

	result := buf.String()
	// Collapse 3+ newlines to 2
	result = multipleNewlines.ReplaceAllString(result, "\n\n")
	result = strings.TrimRight(result, "\n")
	return result
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func writeCodeBlockLines(buf *strings.Builder, n ast.Node, source []byte) {
	lines := n.Lines()
	for i := range lines.Len() {
		seg := lines.At(i)
		line := string(seg.Value(source))
		// Strip trailing newline from the last line (goldmark includes it)
		if i == lines.Len()-1 {
			line = strings.TrimRight(line, "\n")
		}
		buf.WriteString(escapeHTML(line))
	}
}

func trimTrailingNewlines(buf *strings.Builder) {
	s := buf.String()
	trimmed := strings.TrimRight(s, "\n")
	buf.Reset()
	buf.WriteString(trimmed)
}

// extractText collects plain text from all descendant text nodes.
func extractText(n ast.Node, source []byte) string {
	var buf strings.Builder
	ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
		if s, ok := child.(*ast.String); ok {
			buf.Write(s.Value)
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

func renderTable(buf *strings.Builder, table *extast.Table, source []byte) {
	var rows []ast.Node
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		rows = append(rows, row)
	}

	if len(rows) < 2 {
		return
	}

	const sep = "───"

	var headers []ast.Node
	for cell := rows[0].FirstChild(); cell != nil; cell = cell.NextSibling() {
		headers = append(headers, cell)
	}

	buf.WriteString(sep)
	buf.WriteString("\n")
	for _, row := range rows[1:] {
		var cells []ast.Node
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, cell)
		}
		for i, header := range headers {
			buf.WriteString("<b>")
			renderCellInline(buf, header, source)
			buf.WriteString("</b>: ")
			if i < len(cells) {
				renderCellInline(buf, cells[i], source)
			}
			buf.WriteString("\n")
		}
		buf.WriteString(sep)
		buf.WriteString("\n")
	}
}

// renderCellInline writes the inline children of a table cell, preserving
// inline markdown formatting (bold, italic, code, strikethrough, links).
func renderCellInline(buf *strings.Builder, cell ast.Node, source []byte) {
	for child := cell.FirstChild(); child != nil; child = child.NextSibling() {
		renderInlineNode(buf, child, source)
	}
}

func renderInlineNode(buf *strings.Builder, n ast.Node, source []byte) {
	switch node := n.(type) {
	case *ast.Text:
		buf.WriteString(escapeHTML(string(node.Segment.Value(source))))
	case *ast.String:
		buf.WriteString(escapeHTML(string(node.Value)))
	case *ast.CodeSpan:
		buf.WriteString("<code>")
		renderCellInline(buf, node, source)
		buf.WriteString("</code>")
	case *ast.Emphasis:
		open, close := "<i>", "</i>"
		if node.Level == 2 {
			open, close = "<b>", "</b>"
		}
		buf.WriteString(open)
		renderCellInline(buf, node, source)
		buf.WriteString(close)
	case *extast.Strikethrough:
		buf.WriteString("<s>")
		renderCellInline(buf, node, source)
		buf.WriteString("</s>")
	case *ast.Link:
		fmt.Fprintf(buf, `<a href="%s">`, escapeHTML(string(node.Destination)))
		renderCellInline(buf, node, source)
		buf.WriteString("</a>")
	case *ast.AutoLink:
		url := string(node.URL(source))
		label := string(node.Label(source))
		fmt.Fprintf(buf, `<a href="%s">%s</a>`, escapeHTML(url), escapeHTML(label))
	case *ast.Image:
		dest := string(node.Destination)
		alt := extractText(node, source)
		if alt == "" {
			alt = dest
		}
		fmt.Fprintf(buf, `<a href="%s">%s</a>`, escapeHTML(dest), escapeHTML(alt))
	case *ast.RawHTML:
		for i := range node.Segments.Len() {
			seg := node.Segments.At(i)
			buf.WriteString(escapeHTML(string(seg.Value(source))))
		}
	}
}
