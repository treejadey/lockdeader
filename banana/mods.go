package banana

import (
	"context"
	"iter"
	"net/url"
	"strconv"
)

type Sort string

const (
	SortDefault        Sort = ""
	SortNewest         Sort = "Generic_Newest"
	SortLatestUpdated  Sort = "Generic_LatestUpdated"
	SortMostLiked      Sort = "Generic_MostLiked"
	SortMostViewed     Sort = "Generic_MostViewed"
	SortMostDownloaded Sort = "Generic_MostDownloaded"
)

// A part of GameBanana with its own kind of submissions and its own
// categories. The API calls it a model, as in "Mod/Index".
type Section string

const (
	SectionMods   Section = "Mod"
	SectionSounds Section = "Sound"
)

// Zero values select defaults.
type ListModsOptions struct {
	GameID     int     // defaults to DeadlockGameID
	Section    Section // defaults to SectionMods
	CategoryID int     // only list this category, including its subcategories; 0 for all
	Name       string  // only list mods whose name contains this, ignoring case; "" for all
	Page       int     // 1-based; defaults to 1
	PerPage    int     // defaults to MaxPerPage; values above MaxPerPage are clamped
	Sort       Sort    // defaults to the API's own ordering (newest first)
}

func (o ListModsOptions) withDefaults() ListModsOptions {
	if o.GameID == 0 {
		o.GameID = DeadlockGameID
	}

	if o.Section == "" {
		o.Section = SectionMods
	}

	if o.Page < 1 {
		o.Page = 1
	}

	if o.PerPage < 1 || o.PerPage > MaxPerPage {
		o.PerPage = MaxPerPage
	}

	return o
}

type ModPage struct {
	Mods    []Mod
	Page    int // 1-based
	PerPage int
	Total   int // across all pages
	IsLast  bool
}

func (p ModPage) TotalPages() int {
	if p.PerPage <= 0 {
		return 0
	}

	return (p.Total + p.PerPage - 1) / p.PerPage
}

type indexResponse struct {
	Metadata struct {
		RecordCount int  `json:"_nRecordCount"`
		IsComplete  bool `json:"_bIsComplete"`
		PerPage     int  `json:"_nPerpage"`
	} `json:"_aMetadata"`
	Records []Mod `json:"_aRecords"`
}

func (c *Client) ListMods(ctx context.Context, opts ListModsOptions) (*ModPage, error) {
	opts = opts.withDefaults()

	q := url.Values{}
	q.Set("_aFilters[Generic_Game]", strconv.Itoa(opts.GameID))

	if opts.CategoryID != 0 {
		q.Set("_aFilters[Generic_Category]", strconv.Itoa(opts.CategoryID))
	}

	if opts.Name != "" {
		q.Set("_aFilters[Generic_Name]", "contains,"+opts.Name)
	}

	q.Set("_nPage", strconv.Itoa(opts.Page))
	q.Set("_nPerpage", strconv.Itoa(opts.PerPage))

	if opts.Sort != SortDefault {
		q.Set("_sSort", string(opts.Sort))
	}

	var resp indexResponse

	if err := c.get(ctx, string(opts.Section)+"/Index", q, &resp); err != nil {
		return nil, err
	}

	// Fall back to the page size asked for if the response doesn't give one.
	perPage := resp.Metadata.PerPage
	if perPage == 0 {
		perPage = opts.PerPage
	}

	return &ModPage{
		Mods:    resp.Records,
		Page:    opts.Page,
		PerPage: perPage,
		Total:   resp.Metadata.RecordCount,
		IsLast:  resp.Metadata.IsComplete || len(resp.Records) == 0,
	}, nil
}

// Starts at opts.Page, fetching pages as needed. Iteration stops after the
// last page, or after yielding the first error.
func (c *Client) AllMods(ctx context.Context, opts ListModsOptions) iter.Seq2[Mod, error] {
	return func(yield func(Mod, error) bool) {
		opts := opts.withDefaults()

		for {
			page, err := c.ListMods(ctx, opts)
			if err != nil {
				yield(Mod{}, err)

				return
			}

			for _, m := range page.Mods {
				if !yield(m, nil) {
					return
				}
			}

			if page.IsLast {
				return
			}

			opts.Page++
		}
	}
}
