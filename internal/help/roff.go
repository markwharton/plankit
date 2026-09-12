package help

import (
	"fmt"
	"strings"
	"time"

	"github.com/markwharton/plankit/internal/version"
)

// The roff writer: the IR rendered for man(1). It is the third reader of
// the same compiled pages, after the terminal renderer and the site, and
// the only exhaustive one: every switch here ends in a default that
// returns an error, so a node type added to the IR fails the suite
// instead of vanishing from the page.
//
// It does not reuse renderBlock, which wraps to a width and carries an
// ANSI style table. roff fills text itself, so the walk is separate.

// now is stubbed in tests: a man page's date would otherwise change
// with the clock and the output would not be reproducible.
var now = time.Now

// srcWidth is where emitted source lines wrap. roff fills text itself,
// so this is for the file, not the page: mandoc -Tlint reports a source
// line over 80 bytes, and man pages are read as source too.
const srcWidth = 72

// headingMap says what an IR heading level becomes. A single page's
// leading H1 repeats the NAME section, so it is dropped and its lead
// prose becomes DESCRIPTION; in the whole document each page's H1 is
// its own section.
type headingMap struct {
	skipH1 bool
}

// roffWriter carries the state the macros need: whether a section
// heading has just opened a paragraph (a .PP after one is an error),
// and whether a single page still owes its DESCRIPTION heading.
type roffWriter struct {
	sb              *strings.Builder
	hm              headingMap
	afterHeading    bool
	needDescription bool
}

// RenderMan renders one topic as a man page.
func RenderMan(d *Doc) (string, error) {
	var sb strings.Builder
	writeTH(&sb, manTitle(d.Meta.Name))
	w := &roffWriter{sb: &sb, hm: headingMap{skipH1: true}, needDescription: true}
	w.macro(".SH NAME")
	// Through the writer, not Fprintf: a description runs well past the
	// source width, and mandoc reports the line.
	w.text(escape(d.Meta.Name) + ` \- ` + escape(d.Meta.Description))
	if err := w.blocks(d.Blocks); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// RenderManDocument renders every topic as one man page, each page a
// section.
func RenderManDocument(docs []*Doc) (string, error) {
	var sb strings.Builder
	writeTH(&sb, "PK")
	w := &roffWriter{sb: &sb}
	w.macro(".SH NAME")
	w.text(`plankit \- plan\-driven development for Claude Code`)
	for _, d := range docs {
		if err := w.blocks(d.Blocks); err != nil {
			return "", fmt.Errorf("%s: %w", d.Meta.Name, err)
		}
	}
	return sb.String(), nil
}

func writeTH(sb *strings.Builder, title string) {
	fmt.Fprintf(sb, ".TH %s 1 %q %q %q\n", title, now().Format("2006-01-02"),
		"plankit "+version.Version(), "User Commands")
}

// manTitle is the .TH name: PK-SHIP for a command page, upper case as
// man pages are.
func manTitle(name string) string {
	return strings.ToUpper("pk-" + name)
}

func (w *roffWriter) blocks(blocks []Block) error {
	for _, b := range blocks {
		if err := w.block(b); err != nil {
			return err
		}
	}
	return nil
}

func (w *roffWriter) block(b Block) error {
	if b.Type != "heading" {
		w.openDescription()
	}
	switch b.Type {
	case "heading":
		return w.heading(b)
	case "para":
		text, err := spanRoff(b.Inlines)
		if err != nil {
			return err
		}
		w.paragraph(text)
		return nil
	case "codeblock":
		return w.codeblock(b)
	case "list":
		return w.list(b)
	case "quote":
		w.macro(".RS 4")
		if err := w.blocks(b.Blocks); err != nil {
			return err
		}
		w.macro(".RE")
		return nil
	case "rule":
		w.paragraph(`\l'\n(.lu*2u/3u'`)
		return nil
	default:
		return fmt.Errorf("roff: unknown block type %q", b.Type)
	}
}

// openDescription emits the heading a single page owes its lead prose:
// the H1 was dropped into NAME, and man pages put the opening
// description under DESCRIPTION rather than leaving it in NAME.
func (w *roffWriter) openDescription() {
	if !w.needDescription {
		return
	}
	w.needDescription = false
	w.macro(".SH DESCRIPTION")
	w.afterHeading = true
}

func (w *roffWriter) heading(b Block) error {
	text, err := spanText(b.Inlines)
	if err != nil {
		return err
	}
	switch {
	case b.Level == 1 && w.hm.skipH1:
		return nil // the NAME section already carries it
	case b.Level == 1:
		w.section(".SH " + strings.ToUpper(text))
	case b.Level == 2 && w.hm.skipH1:
		w.section(".SH " + strings.ToUpper(text))
	case b.Level == 2:
		w.section(".SS " + text)
	default:
		// roff has two heading levels; a third becomes a bold lead-in.
		w.paragraph(`\fB` + text + `\fR`)
	}
	return nil
}

// section writes a heading macro and records that it opened a
// paragraph, so the next block does not add a .PP of its own.
func (w *roffWriter) section(macro string) {
	w.needDescription = false
	w.macro(macro)
	w.afterHeading = true
}

// paragraph writes .PP and the text, wrapped. The .PP is omitted
// directly after a section heading, which has already opened one:
// mandoc reports "skipping paragraph macro: PP after SH".
func (w *roffWriter) paragraph(text string) {
	if !w.afterHeading {
		w.macro(".PP")
	}
	w.afterHeading = false
	w.text(text)
}

func (w *roffWriter) codeblock(b Block) error {
	text := b.Text
	if len(b.Runs) > 0 {
		var raw strings.Builder
		for _, r := range b.Runs {
			raw.WriteString(r.Text)
		}
		text = raw.String()
	}
	if !w.afterHeading {
		w.macro(".PP")
	}
	w.afterHeading = false
	w.macro(".RS 4")
	w.macro(".nf")
	for _, line := range splitLines(text) {
		w.sb.WriteString(guard(escape(line)) + "\n")
	}
	w.macro(".fi")
	w.macro(".RE")
	return nil
}

func (w *roffWriter) list(b Block) error {
	w.afterHeading = false
	for i, item := range b.Items {
		bullet := `\(bu`
		if b.Ordered {
			bullet = fmt.Sprintf("%d.", i+1)
		}
		// Quoted by hand: %q would escape the backslash in \(bu, and
		// roff would print it.
		w.macro(fmt.Sprintf(".IP \"%s\" 4", bullet))
		for j, ib := range item {
			// The .IP has opened the paragraph; a .PP here would close
			// the hanging indent, so the item's first paragraph is bare.
			if j == 0 && ib.Type == "para" {
				text, err := spanRoff(ib.Inlines)
				if err != nil {
					return err
				}
				w.text(text)
				continue
			}
			if err := w.block(ib); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *roffWriter) macro(m string) { w.sb.WriteString(m + "\n") }

// text writes filled text, wrapped at srcWidth and guarded so no line
// opens with a character roff reads as a request.
func (w *roffWriter) text(s string) {
	for _, line := range wrapRoff(s, srcWidth) {
		w.sb.WriteString(guard(line) + "\n")
	}
}

// wrapRoff breaks text on spaces. Escape sequences never contain a
// space except inside a \fB..\fR run, where a break is harmless: roff
// carries the font across lines.
func wrapRoff(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	lines := []string{}
	cur := words[0]
	for _, word := range words[1:] {
		if len(cur)+1+len(word) > width {
			lines = append(lines, cur)
			cur = word
			continue
		}
		cur += " " + word
	}
	return append(lines, cur)
}

// spanRoff renders spans with font changes: code and strong bold,
// emphasis italic.
func spanRoff(spans []Span) (string, error) {
	var sb strings.Builder
	for _, s := range spans {
		switch s.Type {
		case "br":
			sb.WriteString(" ")
		case "code":
			sb.WriteString(`\fB` + escape(s.Text) + `\fR`)
		case "text":
			t := escape(s.Text)
			switch {
			case s.Strong:
				t = `\fB` + t + `\fR`
			case s.Em:
				t = `\fI` + t + `\fR`
			}
			sb.WriteString(t)
		default:
			return "", fmt.Errorf("roff: unknown span type %q", s.Type)
		}
	}
	return sb.String(), nil
}

// spanText renders spans as plain text, for headings.
func spanText(spans []Span) (string, error) {
	var sb strings.Builder
	for _, s := range spans {
		switch s.Type {
		case "br":
			sb.WriteString(" ")
		case "code", "text":
			sb.WriteString(escape(s.Text))
		default:
			return "", fmt.Errorf("roff: unknown span type %q", s.Type)
		}
	}
	return sb.String(), nil
}

// escape makes text safe for roff: the backslash first, since the
// replacements below introduce their own, then the hyphen, which man
// renders as a soft hyphen unless escaped.
func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\e`)
	s = strings.ReplaceAll(s, "-", `\-`)
	return s
}

// guard prefixes \& when a line opens with a character roff reads as a
// request: . and ' both start one.
func guard(line string) string {
	if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "'") {
		return `\&` + line
	}
	return line
}
