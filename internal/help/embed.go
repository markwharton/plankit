package help

import (
	"embed"
	"sort"
	"strings"
	"sync"
)

//go:embed data
var dataFS embed.FS

var (
	loadOnce sync.Once
	loadErr  error
	docs     map[string]*Doc
	raws     map[string][]byte
	order    []string
)

// load parses every embedded topic once. A failure here is a build
// pipeline bug (docgen validated these), so it surfaces loudly.
func load() error {
	loadOnce.Do(func() {
		docs = map[string]*Doc{}
		raws = map[string][]byte{}
		entries, err := dataFS.ReadDir("data/ir")
		if err != nil {
			loadErr = err
			return
		}
		for _, e := range entries {
			b, err := dataFS.ReadFile("data/ir/" + e.Name())
			if err != nil {
				loadErr = err
				return
			}
			d, err := Load(b)
			if err != nil {
				loadErr = err
				return
			}
			d.Meta.Command = isCommandDoc(d)
			docs[d.Meta.Name] = d
			raw, err := dataFS.ReadFile("data/raw/" + d.Meta.Name + ".md")
			if err != nil {
				loadErr = err
				return
			}
			raws[d.Meta.Name] = raw
			order = append(order, d.Meta.Name)
		}
		// The overview first, then the other documents, then the commands,
		// each group alphabetical. The kind comes from the page's own
		// heading; the overview is the one page named here, because it
		// is the page a reader meets first.
		sort.Slice(order, func(i, j int) bool {
			return rank(docs[order[i]].Meta) < rank(docs[order[j]].Meta)
		})
	})
	return loadErr
}

// Topics returns the embedded topic metadata in display order.
func Topics() ([]Meta, error) {
	if err := load(); err != nil {
		return nil, err
	}
	metas := make([]Meta, 0, len(order))
	for _, name := range order {
		metas = append(metas, docs[name].Meta)
	}
	return metas, nil
}

// Topic returns one topic's IR and raw authored bytes.
func Topic(name string) (*Doc, []byte, bool) {
	if err := load(); err != nil {
		return nil, nil, false
	}
	d, ok := docs[name]
	if !ok {
		return nil, nil, false
	}
	return d, raws[name], true
}

// isCommandDoc reports whether a page's opening heading is "pk <name>",
// the declaration that it documents a command rather than being a
// document such as the overview.
func isCommandDoc(d *Doc) bool {
	if len(d.Blocks) == 0 || d.Blocks[0].Type != "heading" {
		return false
	}
	var text strings.Builder
	for _, s := range d.Blocks[0].Inlines {
		text.WriteString(s.Text)
	}
	return text.String() == "pk "+d.Meta.Name
}

// Overview is the name of the overview page, the document that leads
// every index. The site's sidebar reads the same constant.
const Overview = "overview"

// rank orders the index: the overview, then documents by name, then
// commands by name.
func rank(m Meta) string {
	switch {
	case m.Name == Overview:
		return "0"
	case !m.Command:
		return "1" + m.Name
	default:
		return "2" + m.Name
	}
}
