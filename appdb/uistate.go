package appdb

import (
	"context"
	"fmt"
)

// Keys in the app_state table for UIState.
const (
	keyPage               = "ui.page"
	keyModsView           = "ui.mods_view"
	keyInstalledView      = "ui.installed_view"
	keySection            = "ui.section"
	keyCategoryID         = "ui.category_id"
	keyCategoryPath       = "ui.category_path"
	keyModsTopMod         = "ui.mods_top_mod"
	keyModsTopOffset      = "ui.mods_top_offset"
	keyOpenMod            = "ui.open_mod"
	keyInstalledTop       = "ui.installed_top"
	keyInstalledTopOffset = "ui.installed_top_offset"
	keySettingsY          = "ui.settings_y"
)

// UIState is where the user left the app, so it can reopen there.
type UIState struct {
	Page          int // 0 for mods, 1 for installed mods, 2 for settings
	ModsView      int // 0 for a grid, 1 for a list
	InstalledView int

	// The category browsed on the mods page, as GameBanana numbers it in its
	// section ("Mod" or "Sound"); 0 for the whole section.
	Section    string
	CategoryID int
	// The categories opened to get there, as "Section/id", comma-separated.
	CategoryPath string

	// The first mod on screen, by GameBanana id, and how far past its top the
	// view is scrolled, in pixels. 0 for the top of the list.
	ModsTopMod    int
	ModsTopOffset int
	// The mod open in the details panel; 0 for none.
	OpenMod int

	// The first installed mod on screen, by row, and how far past its top.
	InstalledTop       int
	InstalledTopOffset int
	// In pixels.
	SettingsY int
}

func DefaultUIState() UIState {
	return UIState{Section: "Mod"}
}

// Anything not stored yet, or stored out of range, gets its default.
func (d *DB) UIState(ctx context.Context) (UIState, error) {
	s := DefaultUIState()

	ints := []struct {
		key  string
		dest *int
		max  int // 0 for no maximum
	}{
		{keyPage, &s.Page, 2},
		{keyModsView, &s.ModsView, 1},
		{keyInstalledView, &s.InstalledView, 1},
		{keyCategoryID, &s.CategoryID, 0},
		{keyModsTopMod, &s.ModsTopMod, 0},
		{keyModsTopOffset, &s.ModsTopOffset, 0},
		{keyOpenMod, &s.OpenMod, 0},
		{keyInstalledTop, &s.InstalledTop, 0},
		{keyInstalledTopOffset, &s.InstalledTopOffset, 0},
		{keySettingsY, &s.SettingsY, 0},
	}

	for _, i := range ints {
		var value int

		ok, err := d.state(ctx, i.key, &value)
		if err != nil {
			return DefaultUIState(), err
		}

		if ok && value >= 0 && (i.max == 0 || value <= i.max) {
			*i.dest = value
		}
	}

	var section string

	ok, err := d.state(ctx, keySection, &section)
	if err != nil {
		return DefaultUIState(), err
	}

	if ok && section != "" {
		s.Section = section
	}

	if _, err := d.state(ctx, keyCategoryPath, &s.CategoryPath); err != nil {
		return DefaultUIState(), err
	}

	return s, nil
}

func (d *DB) SetUIState(ctx context.Context, s UIState) error {
	if s.Section == "" {
		return fmt.Errorf("appdb: a UI state needs a section")
	}

	return d.setStates(ctx, map[string]any{
		keyPage:               s.Page,
		keyModsView:           s.ModsView,
		keyInstalledView:      s.InstalledView,
		keySection:            s.Section,
		keyCategoryID:         s.CategoryID,
		keyCategoryPath:       s.CategoryPath,
		keyModsTopMod:         s.ModsTopMod,
		keyModsTopOffset:      s.ModsTopOffset,
		keyOpenMod:            s.OpenMod,
		keyInstalledTop:       s.InstalledTop,
		keyInstalledTopOffset: s.InstalledTopOffset,
		keySettingsY:          s.SettingsY,
	})
}
