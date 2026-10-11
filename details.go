package main

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

// modDetails backs the mod details panel. QML selects a mod by writing its id
// to the "id" key, and Go fills in the other keys: first from the listing, so
// the panel shows something straight away, then from the mod's full profile
// once it has been fetched. An id of 0 means nothing is selected.
//
// QML installs some of the selected mod's files by writing their ids, as a
// list, to the "install" key.
//
// All methods must be called on the Qt main thread.
type modDetails struct {
	*qml.QQmlPropertyMap

	client *banana.Client

	// The listing that QML selects mods from: the section they're in, and
	// mods already loaded into it.
	section func() banana.Section
	lookup  func(id int) (banana.Mod, bool)

	install func(info modInfo, files []banana.File)
	// Called with the id of each mod selected, 0 for none. May be nil.
	onSelected func(id int)

	selected modKey
	profiles map[modKey]*banana.ModProfile
}

// Each section numbers its mods separately.
type modKey struct {
	section banana.Section
	id      int
}

func newModDetails(client *banana.Client, section func() banana.Section, lookup func(id int) (banana.Mod, bool),
	install func(info modInfo, files []banana.File)) *modDetails {
	d := &modDetails{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		client:          client,
		section:         section,
		lookup:          lookup,
		install:         install,
		profiles:        make(map[modKey]*banana.ModProfile),
	}

	d.show("", banana.Submitter{}, "", banana.PreviewMedia{}, "", "", nil)
	d.Select(0)
	d.Insert("install", qt.NewQVariant43(nil))

	// Only fires for changes made from QML, not for our own Inserts.
	d.OnValueChanged(func(key string, value *qt.QVariant) {
		switch key {
		case "id":
			d.Select(value.ToInt())
		case "install":
			var ids []int

			for _, id := range value.ToList() {
				ids = append(ids, id.ToInt())
			}

			// Clear the key, so installing the same files again is still a change.
			d.Insert("install", qt.NewQVariant43(nil))
			d.Install(ids)
		}
	})

	return d
}

// Install does nothing until the mod's profile, which lists its files, has
// arrived.
func (d *modDetails) Install(fileIDs []int) {
	p, ok := d.profiles[d.selected]
	if !ok || len(fileIDs) == 0 {
		return
	}

	var files []banana.File

	for _, f := range p.Files {
		if slices.Contains(fileIDs, f.ID) {
			files = append(files, f)
		}
	}

	if len(files) == 0 {
		return
	}

	info := modInfo{
		section:   d.selected.section,
		id:        p.ID,
		name:      p.Name,
		author:    p.Submitter.Name,
		thumbnail: previewURL(p.PreviewMedia),
		updated:   p.LastUpdated(),
		available: p.Files,
	}

	// Only the listing says whether a mod is NSFW.
	if mod, ok := d.lookup(p.ID); ok {
		info.nsfw = mod.IsNSFW()
	}

	d.install(info, files)
}

// Select looks the mod up in the listing's current section; an id of 0 closes
// the panel. Closing keeps the last mod's details, so the panel can animate
// out with them still showing.
func (d *modDetails) Select(id int) {
	key := modKey{d.section(), id}
	d.selected = key
	d.Insert("id", qt.NewQVariant4(id))

	if d.onSelected != nil {
		d.onSelected(id)
	}

	d.setStatus(false, nil)

	if id == 0 {
		return
	}

	if mod, ok := d.lookup(id); ok {
		d.show(mod.Name, mod.Submitter, modCategory(mod.RootCategory, mod.SubCategory),
			mod.PreviewMedia, mod.ProfileURL, "", nil)
	}

	if p, ok := d.profiles[key]; ok {
		d.showProfile(p)

		return
	}

	d.setStatus(true, nil)

	go func() {
		p, err := d.client.GetModProfile(context.Background(), key.section, id)

		mainthread.Start(func() {
			if err == nil {
				d.profiles[key] = p
			}

			if d.selected != key {
				return // another mod was selected in the meantime
			}

			if err != nil {
				slog.Error("failed to fetch mod profile", "section", key.section, "id", id, "err", err)
				d.setStatus(false, err)

				return
			}

			d.setStatus(false, nil)
			d.showProfile(p)
		})
	}()
}

func (d *modDetails) showProfile(p *banana.ModProfile) {
	description := cleanDescription(p.Text)
	if description == "" {
		description = html.EscapeString(p.Summary)
	}

	d.show(p.Name, p.Submitter, modCategory(p.Category, nil), p.PreviewMedia, p.ProfileURL, description, p.Files)

	if p.SuperCategory != nil {
		d.Insert("category", qt.NewQVariant14(modCategory(*p.SuperCategory, &p.Category)))
	}
}

// description is HTML. Only a mod's full profile has its files.
func (d *modDetails) show(name string, author banana.Submitter, category string, media banana.PreviewMedia, profileURL, description string, files []banana.File) {
	d.Insert("name", qt.NewQVariant14(name))
	d.Insert("author", qt.NewQVariant14(author.Name))
	d.Insert("authorUrl", qt.NewQVariant14(author.ProfileURL))
	d.Insert("category", qt.NewQVariant14(category))
	d.Insert("image", qt.NewQVariant14(previewURL(media)))
	d.Insert("images", galleryImages(media))
	d.Insert("audio", qt.NewQVariant14(media.Metadata.AudioURL))
	d.Insert("profileUrl", qt.NewQVariant14(profileURL))

	if description != "" {
		d.Insert("description", qt.NewQVariant14(codeStyle()+description))
	} else {
		d.Insert("description", qt.NewQVariant14(""))
	}

	d.Insert("plainDescription", qt.NewQVariant14(plainText(description)))
	d.Insert("files", fileList(files))
}

func (d *modDetails) setStatus(loading bool, err error) {
	d.Insert("loading", qt.NewQVariant8(loading))
	insertError(d.QQmlPropertyMap, err)
}

// GameBanana usually only has 100px variants for anything but the first
// image, which SizedURL falls back to.
const galleryThumbnailWidth = 220

func galleryImages(media banana.PreviewMedia) *qt.QVariant {
	list := make([]qt.QVariant, 0, len(media.Images))

	for _, img := range media.Images {
		list = append(list, *qt.NewQVariant20(map[string]qt.QVariant{
			"full":      *qt.NewQVariant14(img.URL()),
			"thumbnail": *qt.NewQVariant14(img.SizedURL(galleryThumbnailWidth)),
			"caption":   *qt.NewQVariant14(img.Caption),
		}))
	}

	return qt.NewQVariant43(list)
}

// Images and embeds would overflow the details panel, and scripts and styles
// would show up as text.
var unwantedHTML = regexp.MustCompile(`(?is)<img\b[^>]*>|<iframe\b.*?</iframe>|<script\b.*?</script>|<style\b.*?</style>`)

func cleanDescription(text string) string {
	return unwantedHTML.ReplaceAllString(text, "")
}

// added is when the file was uploaded, in Unix seconds.
func fileList(files []banana.File) *qt.QVariant {
	list := make([]qt.QVariant, 0, len(files))

	for _, f := range files {
		list = append(list, *qt.NewQVariant20(map[string]qt.QVariant{
			"id":          *qt.NewQVariant4(f.ID),
			"name":        *qt.NewQVariant14(f.Name),
			"size":        *qt.NewQVariant14(formatSize(f.Size)),
			"added":       *qt.NewQVariant6(f.DateAdded),
			"description": *qt.NewQVariant14(f.Description),
		}))
	}

	return qt.NewQVariant43(list)
}

// formatSize matches GameBanana: binary units (a KB is 1024 bytes), with one
// decimal place below 100, unless it's 0, e.g. "450 KB" or "23.5 MB".
func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	size := float64(bytes)
	unit := ""

	for _, u := range []string{"KB", "MB", "GB", "TB"} {
		size /= 1024
		unit = u

		if size < 1024 {
			break
		}
	}

	text := fmt.Sprintf("%.1f", size)
	if size >= 100 || strings.HasSuffix(text, ".0") {
		text = fmt.Sprintf("%.0f", size)
	}

	return text + " " + unit
}

var (
	// Tags that end a line or a paragraph.
	lineBreakHTML = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|li|h[1-6])>`)
	// Content that isn't text at all.
	nonTextHTML = regexp.MustCompile(`(?is)<(script|style)\b.*?</(script|style)>`)
	anyTag      = regexp.MustCompile(`(?s)<[^>]*>`)
	spaces      = regexp.MustCompile(`[ \t\r\f\v\x{a0}]+`)
	blankLines  = regexp.MustCompile(`\n\s*\n\s*\n+`)
)

func plainText(description string) string {
	text := nonTextHTML.ReplaceAllString(description, "")
	text = lineBreakHTML.ReplaceAllString(text, "\n")
	text = html.UnescapeString(anyTag.ReplaceAllString(text, ""))
	text = spaces.ReplaceAllString(text, " ")

	lines := strings.Split(text, "\n")

	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	text = strings.Join(lines, "\n")

	return strings.TrimSpace(blankLines.ReplaceAllString(text, "\n\n"))
}
