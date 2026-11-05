package merge

import (
	"errors"
	"slices"
	"testing"

	"github.com/treejadey/lockdeader/kv3"
	"github.com/treejadey/lockdeader/resource"
)

func event(duration float64, files ...string) kv3.Value {
	list := kv3.Value{Type: kv3.TypeArray}

	for _, f := range files {
		list.Array = append(list.Array, kv3.Value{Type: kv3.TypeString, Flag: kv3.FlagResourceName, String: f})
	}

	return kv3.Value{Type: kv3.TypeObject, Object: kv3.Object{
		{Key: "base", Value: kv3.Value{Type: kv3.TypeString, String: "Base.HeroPick.VO.2d"}},
		{Key: "vsnd_files", Value: list},
		{Key: "vsnd_duration", Value: kv3.Value{Type: kv3.TypeFloat64, Float: duration}},
	}}
}

type ev struct {
	key   string
	value kv3.Value
}

// table compiles a table as a mod or the game has it, with a RED2 block
// that says whose it is.
func table(t *testing.T, whose string, events ...ev) []byte {
	t.Helper()

	root := kv3.Value{Type: kv3.TypeObject}

	for _, e := range events {
		root.Object = append(root.Object, kv3.Member{Key: e.key, Value: e.value})
	}

	data, err := kv3.Encode(&kv3.Document{Format: kv3.FormatGeneric, Root: root})
	if err != nil {
		t.Fatal(err)
	}

	return resource.Write(&resource.Resource{Version: 1, Blocks: []resource.Block{
		{Type: "RED2", Data: []byte(whose)},
		{Type: "DATA", Data: data},
	}})
}

func ships(paths ...string) func(string) bool {
	return func(p string) bool { return slices.Contains(paths, p) }
}

// read returns a merged table's events, as text, and its RED2 block.
func read(t *testing.T, data []byte) (map[string]string, []string, string) {
	t.Helper()

	r, err := resource.Read(data)
	if err != nil {
		t.Fatal(err)
	}

	red2, _ := r.Block("RED2")
	block, _ := r.Block("DATA")

	doc, err := kv3.Decode(block)
	if err != nil {
		t.Fatal(err)
	}

	events := map[string]string{}

	var order []string

	for _, e := range doc.Root.Object {
		events[e.Key] = text(e.Value)
		order = append(order, e.Key)
	}

	return events, order, string(red2)
}

// The game's pick events for three heroes, one of them stored as the
// game stores it: 5.577392 as a single-precision float.
var (
	fencer   = event(3.8, "sounds/vo/fencer/fencer_select_01.vsnd")
	deadpack = event(18.047665, "sounds/vo/deadpack/deadpack_select_02.vsnd")
	haze     = event(float64(float32(5.577392)), "sounds/vo/haze/haze_select_01.vsnd")

	newFencer   = event(1.7, "sounds/vo/fencer/fencer_select_01.vsnd")
	newDeadpack = event(17.475918, "sounds/vo/deadpack/deadpack_select_02.vsnd")
)

func gameTable(t *testing.T) []byte {
	return table(t, "game", ev{"Fencer.Pick", fencer}, ev{"Deadpack.Pick", deadpack}, ev{"Haze.Pick", haze})
}

func TestMergeDifferentHeroes(t *testing.T) {
	flowery := Copy{
		Data:  table(t, "flowery", ev{"Fencer.Pick", newFencer}, ev{"Deadpack.Pick", deadpack}, ev{"Haze.Pick", haze}),
		Ships: ships("sounds/vo/fencer/fencer_select_01.vsnd_c"),
	}
	hatchling := Copy{
		// Recompiled from text, so its copy of Haze's event has 5.577392
		// exactly, which is no change.
		Data:  table(t, "hatchling", ev{"Fencer.Pick", fencer}, ev{"Deadpack.Pick", newDeadpack}, ev{"Haze.Pick", event(5.577392, "sounds/vo/haze/haze_select_01.vsnd")}),
		Ships: ships("sounds/vo/deadpack/deadpack_select_02.vsnd_c"),
	}

	res, err := SoundEvents(gameTable(t), []Copy{flowery, hatchling})
	if err != nil {
		t.Fatal(err)
	}

	events, order, red2 := read(t, res.Data)

	want := map[string]string{"Fencer.Pick": text(newFencer), "Deadpack.Pick": text(newDeadpack), "Haze.Pick": text(haze)}
	for k, v := range want {
		if events[k] != v {
			t.Errorf("%s =\n%s\nwant\n%s", k, events[k], v)
		}
	}

	if !slices.Equal(order, []string{"Fencer.Pick", "Deadpack.Pick", "Haze.Pick"}) {
		t.Errorf("order = %v, want the game's", order)
	}

	if red2 != "game" {
		t.Errorf("RED2 = %q, want the game's", red2)
	}

	if len(res.Conflicts) != 0 || len(res.LeftOut[0]) != 0 || len(res.LeftOut[1]) != 0 {
		t.Errorf("conflicts %v, left out %v; want none", res.Conflicts, res.LeftOut)
	}
}

// An out-of-date copy lacks events the game has and has old versions of
// others; only its changes with its own sounds are taken.
func TestMergeOutOfDateCopy(t *testing.T) {
	oldHaze := event(4.2, "sounds/vo/haze/haze_select_01.vsnd")

	stale := Copy{
		Data:  table(t, "stale", ev{"Fencer.Pick", newFencer}, ev{"Haze.Pick", oldHaze}, ev{"Boho.Pick", event(1, "sounds/vo/boho/boho_select_01.vsnd")}),
		Ships: ships("sounds/vo/fencer/fencer_select_01.vsnd_c"),
	}
	other := Copy{
		Data:  table(t, "other", ev{"Fencer.Pick", fencer}, ev{"Deadpack.Pick", newDeadpack}, ev{"Haze.Pick", haze}),
		Ships: ships("sounds/vo/deadpack/deadpack_select_02.vsnd_c"),
	}

	res, err := SoundEvents(gameTable(t), []Copy{stale, other})
	if err != nil {
		t.Fatal(err)
	}

	events, order, _ := read(t, res.Data)

	if events["Haze.Pick"] != text(haze) {
		t.Errorf("Haze.Pick =\n%s\nwant the game's", events["Haze.Pick"])
	}

	if events["Deadpack.Pick"] != text(newDeadpack) || events["Fencer.Pick"] != text(newFencer) {
		t.Error("the mods' own events weren't taken")
	}

	// Added events are taken, after the game's.
	if !slices.Equal(order, []string{"Fencer.Pick", "Deadpack.Pick", "Haze.Pick", "Boho.Pick"}) {
		t.Errorf("order = %v", order)
	}

	if !slices.Equal(res.LeftOut[0], []string{"Haze.Pick"}) || len(res.LeftOut[1]) != 0 {
		t.Errorf("left out = %v, want only the stale copy's Haze.Pick", res.LeftOut)
	}
}

func TestMergeConflict(t *testing.T) {
	own := ships("sounds/vo/haze/haze_select_01.vsnd_c")
	first := Copy{Data: table(t, "first", ev{"Haze.Pick", event(1, "sounds/vo/haze/haze_select_01.vsnd")}), Ships: own}
	second := Copy{Data: table(t, "second", ev{"Haze.Pick", event(2, "sounds/vo/haze/haze_select_01.vsnd")}), Ships: own}
	same := Copy{Data: table(t, "same", ev{"Haze.Pick", event(1, "sounds/vo/haze/haze_select_01.vsnd")}), Ships: own}

	res, err := SoundEvents(gameTable(t), []Copy{first, second, same})
	if err != nil {
		t.Fatal(err)
	}

	want := []Conflict{{Event: "Haze.Pick", Copies: []int{0, 1}}}
	if !slices.EqualFunc(res.Conflicts, want, func(a, b Conflict) bool { return a.Event == b.Event && slices.Equal(a.Copies, b.Copies) }) {
		t.Errorf("conflicts = %v, want %v", res.Conflicts, want)
	}

	events, _, _ := read(t, res.Data)

	if events["Haze.Pick"] != text(event(1, "sounds/vo/haze/haze_select_01.vsnd")) {
		t.Error("the first copy in the load order didn't win")
	}

	if len(res.LeftOut[0]) != 0 || !slices.Equal(res.LeftOut[1], []string{"Haze.Pick"}) || len(res.LeftOut[2]) != 0 {
		t.Errorf("left out = %v, want only the second copy's", res.LeftOut)
	}
}

// Two copies that add the same event: one using its mod's own sounds wins
// over one that doesn't, which is most likely an old copy's.
func TestMergeAddedEvents(t *testing.T) {
	mine := event(1, "sounds/vo/new/line.vsnd")
	old := event(2, "sounds/vo/new/line.vsnd")

	stale := Copy{Data: table(t, "stale", ev{"New.Line", old}), Ships: ships()}
	owner := Copy{Data: table(t, "owner", ev{"New.Line", mine}), Ships: ships("sounds/vo/new/line.vsnd_c")}

	res, err := SoundEvents(gameTable(t), []Copy{stale, owner})
	if err != nil {
		t.Fatal(err)
	}

	events, _, _ := read(t, res.Data)

	if events["New.Line"] != text(mine) || len(res.Conflicts) != 0 {
		t.Errorf("New.Line =\n%s\nconflicts %v; want the owner's, no conflict", events["New.Line"], res.Conflicts)
	}

	// The game's events are all still there, as the copies lack them.
	if len(events) != 4 {
		t.Errorf("%d events, want the game's 3 and the new one", len(events))
	}
}

// Copies that are all the same are used as they are, deliberate changes
// that don't use the mod's own sounds and all.
func TestMergeIdenticalCopies(t *testing.T) {
	data := table(t, "mod", ev{"Haze.Pick", event(9, "sounds/vo/haze/haze_select_01.vsnd")})

	res, err := SoundEvents(gameTable(t), []Copy{{Data: data, Ships: ships()}, {Data: data, Ships: ships()}})
	if err != nil {
		t.Fatal(err)
	}

	if string(res.Data) != string(data) || len(res.LeftOut[0])+len(res.LeftOut[1]) != 0 {
		t.Error("identical copies weren't used as they are")
	}
}

// A table the game doesn't have: the copies' events together.
func TestMergeNotInGame(t *testing.T) {
	a := Copy{Data: table(t, "a", ev{"A", event(1, "a.vsnd")}), Ships: ships("a.vsnd_c")}
	b := Copy{Data: table(t, "b", ev{"B", event(2, "b.vsnd")}), Ships: ships("b.vsnd_c")}

	res, err := SoundEvents(nil, []Copy{a, b})
	if err != nil {
		t.Fatal(err)
	}

	events, order, red2 := read(t, res.Data)

	if !slices.Equal(order, []string{"A", "B"}) || events["B"] != text(event(2, "b.vsnd")) || red2 != "a" {
		t.Errorf("events %v, RED2 %q; want A then B, and the first copy's RED2", order, red2)
	}
}

func TestMergeUnreadable(t *testing.T) {
	good := Copy{Data: gameTable(t), Ships: ships()}
	bad := Copy{Data: []byte("not a compiled file"), Ships: ships()}

	var ce *CopyError

	if _, err := SoundEvents(gameTable(t), []Copy{good, bad}); !errors.As(err, &ce) || ce.Copy != 1 {
		t.Errorf("SoundEvents with a broken copy = %v, want a CopyError for copy 1", err)
	}

	if _, err := SoundEvents([]byte("broken"), []Copy{good, {Data: table(t, "x"), Ships: ships()}}); err == nil {
		t.Error("SoundEvents with a broken game copy succeeded")
	}

	// A compiled file whose data isn't a table of named entries.
	list, _ := kv3.Encode(&kv3.Document{Root: kv3.Value{Type: kv3.TypeArray}})
	notTable := resource.Write(&resource.Resource{Blocks: []resource.Block{{Type: "DATA", Data: list}}})

	if _, err := SoundEvents(nil, []Copy{good, {Data: notTable, Ships: ships()}}); !errors.Is(err, ErrNotTable) {
		t.Errorf("SoundEvents with a list = %v, want ErrNotTable", err)
	}
}
