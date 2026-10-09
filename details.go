package main

import (
	"context"
	"html"
	"log/slog"
	"regexp"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/mappu/miqt/qt6/qml"

	"github.com/treejadey/lockdeader/banana"
)

// modDetails backs the mod details panel. QML selects a mod by writing its id
// to the "id" key, and Go fills in the other keys: first from the listing, so
// the panel shows something immediately, then from the mod's full profile once
// it has been fetched. An id of 0 means nothing is selected.
// All methods must be called on the Qt main thread.
type modDetails struct {
	*qml.QQmlPropertyMap

	client *banana.Client
	lookup func(id int) (banana.Mod, bool) // finds mods already in the listing

	selected int
	profiles map[int]*banana.ModProfile // profiles fetched so far, by mod id
}

func newModDetails(client *banana.Client, lookup func(id int) (banana.Mod, bool)) *modDetails {
	d := &modDetails{
		QQmlPropertyMap: qml.NewQQmlPropertyMap(),
		client:          client,
		lookup:          lookup,
		profiles:        make(map[int]*banana.ModProfile),
	}
	d.show("", "", "", banana.PreviewMedia{}, "", "")
	d.Select(0)

	// Only fires for changes made from QML, not for our own Inserts.
	d.OnValueChanged(func(key string, value *qt.QVariant) {
		if key == "id" {
			d.Select(value.ToInt())
		}
	})
	return d
}

// Select shows the mod with the given id, or closes the panel if id is 0.
// Closing keeps the last mod's details, so the panel can animate out with
// them still showing.
func (d *modDetails) Select(id int) {
	d.selected = id
	d.Insert("id", qt.NewQVariant4(id))
	d.setStatus(false, "")

	if id == 0 {
		return
	}

	if mod, ok := d.lookup(id); ok {
		d.show(mod.Name, mod.Submitter.Name, modCategory(mod.RootCategory, mod.SubCategory),
			mod.PreviewMedia, mod.ProfileURL, "")
	}
	if p, ok := d.profiles[id]; ok {
		d.showProfile(p)
		return
	}

	d.setStatus(true, "")
	go func() {
		p, err := d.client.GetModProfile(context.Background(), id)

		mainthread.Start(func() {
			if err == nil {
				d.profiles[id] = p
			}
			if d.selected != id {
				return // another mod was selected in the meantime
			}
			if err != nil {
				slog.Error("failed to fetch mod profile", "id", id, "err", err)
				d.setStatus(false, err.Error())
				return
			}
			d.setStatus(false, "")
			d.showProfile(p)
		})
	}()
}

func (d *modDetails) showProfile(p *banana.ModProfile) {
	description := cleanDescription(p.Text)
	if description == "" {
		description = html.EscapeString(p.Summary)
	}
	d.show(p.Name, p.Submitter.Name, modCategory(p.Category, nil), p.PreviewMedia, p.ProfileURL, description)
	if p.SuperCategory != nil {
		d.Insert("category", qt.NewQVariant14(modCategory(*p.SuperCategory, &p.Category)))
	}
}

func (d *modDetails) show(name, author, category string, media banana.PreviewMedia, profileURL, description string) {
	d.Insert("name", qt.NewQVariant14(name))
	d.Insert("author", qt.NewQVariant14(author))
	d.Insert("category", qt.NewQVariant14(category))
	d.Insert("image", qt.NewQVariant14(previewURL(media)))
	d.Insert("images", galleryImages(media))
	d.Insert("profileUrl", qt.NewQVariant14(profileURL))
	d.Insert("description", qt.NewQVariant14(description))
}

func (d *modDetails) setStatus(loading bool, err string) {
	d.Insert("loading", qt.NewQVariant8(loading))
	d.Insert("error", qt.NewQVariant14(err))
}

// galleryThumbnailWidth is the preview variant used for gallery thumbnails.
// GameBanana usually only has 100px variants for anything but the first
// image, which SizedURL falls back to.
const galleryThumbnailWidth = 220

// galleryImages lists a mod's preview images for the gallery and the image
// viewer, as {full, thumbnail, caption} objects.
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

// unwantedHTML matches description content that the details panel shouldn't
// render: images and embeds would overflow it, and scripts and styles would
// show up as text.
var unwantedHTML = regexp.MustCompile(`(?is)<img\b[^>]*>|<iframe\b.*?</iframe>|<script\b.*?</script>|<style\b.*?</style>`)

// cleanDescription prepares a mod's HTML description for a QML rich text label.
func cleanDescription(text string) string {
	return unwantedHTML.ReplaceAllString(text, "")
}
