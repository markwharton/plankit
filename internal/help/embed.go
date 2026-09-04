package help

import (
	"embed"
	"sort"
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
			docs[d.Meta.Name] = d
			raw, err := dataFS.ReadFile("data/raw/" + d.Meta.Name + ".md")
			if err != nil {
				loadErr = err
				return
			}
			raws[d.Meta.Name] = raw
			order = append(order, d.Meta.Name)
		}
		// Alphabetical, with the overview pinned first.
		sort.Slice(order, func(i, j int) bool {
			if order[i] == "plankit" {
				return true
			}
			if order[j] == "plankit" {
				return false
			}
			return order[i] < order[j]
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
