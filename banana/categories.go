package banana

import (
	"context"
	"net/url"
	"strconv"
)

type CategoryEntry struct {
	ID            int    `json:"_idRow"`
	Name          string `json:"_sName"`
	ItemCount     int    `json:"_nItemCount"`     // submissions in it, including its subcategories'
	CategoryCount int    `json:"_nCategoryCount"` // subcategories, including empty ones
	URL           string `json:"_sUrl"`
	IconURL       string `json:"_sIconUrl"` // optional
	IsObsolete    bool   `json:"_bIsObsolete"`
}

// Zero values select defaults.
type ListCategoriesOptions struct {
	GameID   int     // defaults to DeadlockGameID; ignored when ParentID is set
	Section  Section // defaults to SectionMods
	ParentID int     // list this category's subcategories; 0 for the game's top-level categories
}

// The categories come in alphabetical order, and include empty ones.
func (c *Client) ListCategories(ctx context.Context, opts ListCategoriesOptions) ([]CategoryEntry, error) {
	if opts.GameID == 0 {
		opts.GameID = DeadlockGameID
	}

	if opts.Section == "" {
		opts.Section = SectionMods
	}

	q := url.Values{}

	if opts.ParentID != 0 {
		q.Set("_idCategoryRow", strconv.Itoa(opts.ParentID))
	} else {
		q.Set("_idGameRow", strconv.Itoa(opts.GameID))
	}

	// The API's default sort is rejected as invalid, so one has to be given.
	q.Set("_sSort", "a_to_z")
	q.Set("_bShowEmpty", "true")

	var categories []CategoryEntry

	if err := c.get(ctx, string(opts.Section)+"/Categories", q, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}
