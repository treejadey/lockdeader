// Package merge combines the copies several mods have of one of the game's
// table files, so mods that change different parts of it can be on
// together. See Backlog doc-4, Merging files that mods share.
package merge

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/treejadey/lockdeader/kv3"
	"github.com/treejadey/lockdeader/resource"
)

// A Copy is one mod's copy of a table.
type Copy struct {
	// The compiled file, as the mod has it.
	Data []byte
	// Ships says whether the mod has a file, by its path in lower case.
	Ships func(path string) bool
}

// A Result is a merged table.
type Result struct {
	// The compiled file.
	Data []byte
	// For each copy, in order: the events it changes that the merge left
	// out, either because they don't use the mod's own sounds or because
	// another copy before it in the load order won a conflict over them.
	LeftOut   [][]string
	Conflicts []Conflict
}

// A Conflict is an event that copies change differently, each with sounds
// its mod ships: a real clash, which the merge can't settle, so the first
// of them in the load order wins.
type Conflict struct {
	Event  string
	Copies []int
}

// A CopyError says which copy couldn't be read.
type CopyError struct {
	Copy int
	Err  error
}

func (e *CopyError) Error() string { return fmt.Sprintf("merge: copy %d: %v", e.Copy, e.Err) }
func (e *CopyError) Unwrap() error { return e.Err }

var ErrNotTable = errors.New("merge: not a table of named entries")

// SoundEvents merges copies of a sound event table (a .vsndevts_c file),
// given in load order, over the game's own copy, which is nil for a table
// the game doesn't have. Each event comes from the game, except:
//   - an event a copy changes is taken from it if it refers to a sound file
//     its mod ships, so an out-of-date copy can't revert the game's newer
//     events;
//   - an event a copy adds is taken from it, preferring one that uses its
//     mod's own sounds;
//   - events are compared as KV3 text has them, to six decimals, since
//     mods recompile tables decompiled to text, which shifts every number
//     slightly.
//
// Copies that are all the same are nothing to merge: the result is that
// copy, unchanged.
func SoundEvents(game []byte, copies []Copy) (*Result, error) {
	res := &Result{LeftOut: make([][]string, len(copies))}

	if len(copies) == 0 {
		return nil, errors.New("merge: no copies")
	}

	if allSame(copies) {
		res.Data = copies[0].Data

		return res, nil
	}

	docs := make([]*kv3.Document, len(copies))

	var template *resource.Resource

	for i, c := range copies {
		r, doc, err := load(c.Data)
		if err != nil {
			return nil, &CopyError{Copy: i, Err: err}
		}

		docs[i] = doc

		if template == nil {
			template = r
		}
	}

	// The game's copy is the base, and its file the one written back, with
	// its own RED2 block; for a table the game doesn't have, the first copy.
	base := &kv3.Document{Format: docs[0].Format, Root: kv3.Value{Type: kv3.TypeObject}}

	if game != nil {
		r, doc, err := load(game)
		if err != nil {
			return nil, fmt.Errorf("merge: the game's copy: %w", err)
		}

		template, base = r, doc
	}

	m := &merger{res: res, base: base.Root.Object, picks: map[string][]pick{}}

	for i, doc := range docs {
		m.collect(i, doc.Root.Object, copies[i].Ships)
	}

	merged := make(kv3.Object, 0, len(m.base)+len(m.added))

	for _, e := range m.base {
		if v, ok := m.resolve(e.Key); ok {
			e.Value = v
		}

		merged = append(merged, e)
	}

	for _, key := range m.added {
		v, _ := m.resolve(key)
		merged = append(merged, kv3.Member{Key: key, Value: v})
	}

	data, err := kv3.Encode(&kv3.Document{Format: base.Format, Root: kv3.Value{Type: kv3.TypeObject, Object: merged}})
	if err != nil {
		return nil, fmt.Errorf("merge: %w", err)
	}

	out := &resource.Resource{Version: template.Version}

	for _, b := range template.Blocks {
		if b.Type == "DATA" {
			b.Data = data
		}

		out.Blocks = append(out.Blocks, b)
	}

	res.Data = resource.Write(out)

	return res, nil
}

func allSame(copies []Copy) bool {
	for _, c := range copies[1:] {
		if !bytes.Equal(c.Data, copies[0].Data) {
			return false
		}
	}

	return true
}

func load(data []byte) (*resource.Resource, *kv3.Document, error) {
	r, err := resource.Read(data)
	if err != nil {
		return nil, nil, err
	}

	block, ok := r.Block("DATA")
	if !ok {
		return nil, nil, fmt.Errorf("%w: no DATA block", ErrNotTable)
	}

	doc, err := kv3.Decode(block)
	if err != nil {
		return nil, nil, err
	}

	if doc.Root.Type != kv3.TypeObject {
		return nil, nil, ErrNotTable
	}

	return r, doc, nil
}

// A pick is a copy's version of an event that's in the running to be
// taken.
type pick struct {
	copy  int
	value kv3.Value
	text  string
	own   bool
}

type merger struct {
	res   *Result
	base  kv3.Object
	picks map[string][]pick
	// Events the game doesn't have, in the order copies first add them.
	added []string
}

func (m *merger) collect(i int, events kv3.Object, ships func(string) bool) {
	for _, e := range events {
		t := text(e.Value)
		g, inGame := m.base.Get(e.Key)

		if inGame && text(*g) == t {
			continue
		}

		own := usesOwnSounds(e.Value, ships)

		if inGame && !own {
			m.leftOut(i, e.Key)

			continue
		}

		if !inGame && m.picks[e.Key] == nil {
			m.added = append(m.added, e.Key)
		}

		m.picks[e.Key] = append(m.picks[e.Key], pick{copy: i, value: e.Value, text: t, own: own})
	}
}

// resolve returns the version of an event the merge takes, if any copy's
// is, and notes what it leaves out and which copies conflict.
func (m *merger) resolve(key string) (kv3.Value, bool) {
	picks := m.picks[key]
	if len(picks) == 0 {
		return kv3.Value{}, false
	}

	// An added event that uses its mod's own sounds wins over ones that
	// don't, which are most likely an old copy's.
	var own []pick

	for _, p := range picks {
		if p.own {
			own = append(own, p)
		}
	}

	if len(own) > 0 && len(own) < len(picks) {
		for _, p := range picks {
			if !p.own {
				m.leftOut(p.copy, key)
			}
		}

		picks = own
	}

	winner := picks[0]
	conflict := Conflict{Event: key, Copies: []int{winner.copy}}

	for _, p := range picks[1:] {
		if p.text != winner.text {
			conflict.Copies = append(conflict.Copies, p.copy)
			m.leftOut(p.copy, key)
		}
	}

	if len(conflict.Copies) > 1 {
		m.res.Conflicts = append(m.res.Conflicts, conflict)
	}

	return winner.value, true
}

func (m *merger) leftOut(i int, key string) {
	m.res.LeftOut[i] = append(m.res.LeftOut[i], key)
}

func text(v kv3.Value) string {
	return (&kv3.Document{Root: v}).Text()
}

// usesOwnSounds says whether an event refers to a sound file the mod ships.
// Events name sounds by their source, sounds/….vsnd, which is compiled to
// sounds/….vsnd_c.
func usesOwnSounds(v kv3.Value, ships func(string) bool) bool {
	switch v.Type {
	case kv3.TypeString:
		s := strings.ToLower(v.String)

		return strings.HasSuffix(s, ".vsnd") && ships(s+"_c")
	case kv3.TypeArray:
		for _, e := range v.Array {
			if usesOwnSounds(e, ships) {
				return true
			}
		}
	case kv3.TypeObject:
		for _, e := range v.Object {
			if usesOwnSounds(e.Value, ships) {
				return true
			}
		}
	}

	return false
}
