package help

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/markwharton/plankit/internal/cli"
)

// codes is the style table for one rendering. The zero value renders
// plain text; the same walker produces both outputs.
type codes struct {
	h      [4]string // by heading level; [0] unused
	strong string
	em     string
	code   string
	block  string // codeblock lines
	url    string // the appended "(url)" token
	rule   string
	reset  string
	run    map[string]string // highlight run kinds
}

// ansiCodes is deliberately conservative: bold, dim, underline, italics,
// and the 16-color palette only, so the output stays readable on light
// and dark themes alike. No 256-color or truecolor absolutes.
func ansiCodes() codes {
	return codes{
		h:      [4]string{"", "\x1b[1;4m", "\x1b[1m", "\x1b[4m"},
		strong: "1",
		em:     "3",
		code:   "36",
		block:  "\x1b[2m",
		url:    "\x1b[2m",
		rule:   "\x1b[2m",
		reset:  "\x1b[0m",
		run: map[string]string{
			"kw":   "\x1b[1m",
			"str":  "\x1b[32m",
			"com":  "\x1b[2m",
			"num":  "\x1b[35m",
			"name": "\x1b[36m",
		},
	}
}

// Render walks the document and produces terminal output. width 0 means
// no wrapping (each paragraph on one line); style selects the table.
func Render(d *Doc, style cli.Style, width int) string {
	c := codes{}
	if style == cli.StyleANSI {
		c = ansiCodes()
	}
	var sb strings.Builder
	renderBlocks(&sb, d.Blocks, c, width, "", "")
	return sb.String()
}

// renderBlocks writes blocks separated by blank lines. firstPrefix and
// contPrefix carry list-bullet and quote indentation into nested content.
func renderBlocks(sb *strings.Builder, blocks []Block, c codes, width int, firstPrefix, contPrefix string) {
	for i, b := range blocks {
		if i > 0 {
			sb.WriteString("\n")
		}
		pfx := contPrefix
		if i == 0 {
			pfx = firstPrefix
		}
		renderBlock(sb, b, c, width, pfx, contPrefix)
	}
}

func renderBlock(sb *strings.Builder, b Block, c codes, width int, firstPrefix, contPrefix string) {
	switch b.Type {
	case "heading":
		open := c.h[b.Level]
		text := flatten(b.Inlines)
		sb.WriteString(firstPrefix + styled(open, text, c.reset) + "\n")
	case "para":
		for _, line := range wrapSpans(b.Inlines, c, width, firstPrefix, contPrefix) {
			sb.WriteString(line + "\n")
		}
	case "codeblock":
		renderCode(sb, b, c, contPrefix)
	case "list":
		for i, item := range b.Items {
			bullet := "- "
			if b.Ordered {
				bullet = strconv.Itoa(i+1) + ". "
			}
			renderBlocks(sb, item, c, width,
				contPrefix+"  "+bullet,
				contPrefix+"  "+strings.Repeat(" ", len(bullet)))
		}
	case "quote":
		renderBlocks(sb, b.Blocks, c, width, contPrefix+"  > ", contPrefix+"  > ")
	case "rule":
		if width > 0 {
			sb.WriteString(contPrefix + styled(c.rule, strings.Repeat("─", min(width-len(contPrefix), 60)), c.reset) + "\n")
		} else {
			sb.WriteString(contPrefix + "---\n")
		}
	}
}

func renderCode(sb *strings.Builder, b Block, c codes, prefix string) {
	indent := prefix + "    "
	emit := func(line string) {
		sb.WriteString(indent + styled(c.block, line, c.reset) + "\n")
	}
	if len(b.Runs) == 0 {
		for _, line := range splitLines(b.Text) {
			emit(line)
		}
		return
	}
	// Highlighted: runs are split at newlines by docgen, so styling per
	// run and per line compose without escape codes crossing lines.
	line := strings.Builder{}
	for _, r := range b.Runs {
		for i, part := range strings.Split(r.Text, "\n") {
			if i > 0 {
				emit(line.String())
				line.Reset()
			}
			if part == "" {
				continue
			}
			if open := c.run[r.Kind]; open != "" {
				line.WriteString(open + part + c.reset)
			} else {
				line.WriteString(part)
			}
		}
	}
	if line.Len() > 0 {
		emit(line.String())
	}
}

// token is one wrappable unit: a word with its style opening. glue
// means no space separates it from the token before it, which is how
// "(`x`)" stays "(x)" and "`pk`'s" stays "pk's" through wrapping.
type token struct {
	word string
	open string
	br   bool
	glue bool
}

// wrapSpans flattens spans to tokens and wraps them greedily by rune
// count. Inline code is atomic; a Br token forces a line break. width 0
// joins everything onto one line.
func wrapSpans(spans []Span, c codes, width int, firstPrefix, contPrefix string) []string {
	toks := tokens(spans, c)
	lines := []string{}
	cur := strings.Builder{}
	curLen := 0
	prefix := firstPrefix

	flush := func() {
		lines = append(lines, prefix+cur.String())
		cur.Reset()
		curLen = 0
		prefix = contPrefix
	}

	for _, t := range toks {
		if t.br {
			flush()
			continue
		}
		wlen := len([]rune(t.word))
		sep := 0
		if curLen > 0 && !t.glue {
			sep = 1
		}
		if width > 0 && curLen > 0 && !t.glue && len(prefix)+curLen+sep+wlen > width {
			flush()
			sep = 0
		}
		if sep == 1 {
			cur.WriteString(" ")
			curLen++
		}
		cur.WriteString(styled(t.open, t.word, c.reset))
		curLen += wlen
	}
	if curLen > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

func tokens(spans []Span, c codes) []token {
	toks := []token{}
	// spaceBefore is whether the next token follows whitespace: true at
	// the start of a paragraph, after a break, and after any text that
	// ends in whitespace. A code span glues to its neighbors unless
	// whitespace stood between them in the source.
	spaceBefore := true
	for _, s := range spans {
		switch s.Type {
		case "br":
			toks = append(toks, token{br: true})
			spaceBefore = true
		case "code":
			toks = append(toks, token{word: s.Text, open: sgr(c.code), glue: !spaceBefore})
			toks = appendURL(toks, s.Text, s.URL, c)
			spaceBefore = false
		case "text":
			open := spanOpen(s, c)
			leading := len(s.Text) > 0 && unicode.IsSpace(rune(s.Text[0]))
			trailing := len(s.Text) > 0 && unicode.IsSpace(rune(s.Text[len(s.Text)-1]))
			words := strings.Fields(s.Text)
			for i, w := range words {
				glue := i == 0 && !leading && !spaceBefore
				toks = append(toks, token{word: w, open: open, glue: glue})
			}
			toks = appendURL(toks, s.Text, s.URL, c)
			if len(words) == 0 {
				spaceBefore = spaceBefore || len(s.Text) > 0
			} else {
				spaceBefore = trailing
			}
		}
	}
	return toks
}

// appendURL adds the "(url)" token after a linked span when the visible
// text is not already the target.
func appendURL(toks []token, text, url string, c codes) []token {
	if url == "" || url == strings.TrimSpace(text) {
		return toks
	}
	return append(toks, token{word: "(" + url + ")", open: c.url})
}

func spanOpen(s Span, c codes) string {
	params := []string{}
	if s.Strong && c.strong != "" {
		params = append(params, c.strong)
	}
	if s.Em && c.em != "" {
		params = append(params, c.em)
	}
	if len(params) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(params, ";") + "m"
}

func sgr(param string) string {
	if param == "" {
		return ""
	}
	return "\x1b[" + param + "m"
}

func styled(open, text, reset string) string {
	if open == "" {
		return text
	}
	return open + text + reset
}

// flatten renders spans as plain text, for headings and the TOC.
func flatten(spans []Span) string {
	var sb strings.Builder
	for _, s := range spans {
		if s.Type == "br" {
			sb.WriteString(" ")
			continue
		}
		sb.WriteString(s.Text)
	}
	return sb.String()
}

func splitLines(text string) []string {
	return strings.Split(strings.TrimRight(text, "\n"), "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
