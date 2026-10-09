package banana

import (
	"context"
	"iter"
	"net/url"
	"strconv"
)

// Sort controls the ordering of a mod listing.
type Sort string

const (
	SortDefault        Sort = ""
	SortNewest         Sort = "Generic_Newest"
	SortLatestUpdated  Sort = "Generic_LatestUpdated"
	SortMostLiked      Sort = "Generic_MostLiked"
	SortMostViewed     Sort = "Generic_MostViewed"
	SortMostDownloaded Sort = "Generic_MostDownloaded"
)

// ListModsOptions configures ListMods. Zero values select sensible defaults.
type ListModsOptions struct {
	GameID  int  // defaults to DeadlockGameID
	Page    int  // 1-based; defaults to 1
	PerPage int  // defaults to MaxPerPage; values above MaxPerPage are clamped
	Sort    Sort // defaults to the API's own ordering (newest first)
}

func (o ListModsOptions) withDefaults() ListModsOptions {
	if o.GameID == 0 {
		o.GameID = DeadlockGameID
	}
	if o.Page < 1 {
		o.Page = 1
	}
	if o.PerPage < 1 || o.PerPage > MaxPerPage {
		o.PerPage = MaxPerPage
	}
	return o
}

// ModPage is one page of a mod listing.
type ModPage struct {
	Mods    []Mod
	Page    int  // 1-based page number this result is for
	PerPage int  // page size used for the request
	Total   int  // total number of mods across all pages
	IsLast  bool // true if there are no further pages
}

// TotalPages returns the number of pages in the listing at this page size.
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

// ListMods fetches a single page of mods for a game.
func (c *Client) ListMods(ctx context.Context, opts ListModsOptions) (*ModPage, error) {
	opts = opts.withDefaults()

	q := url.Values{}
	q.Set("_aFilters[Generic_Game]", strconv.Itoa(opts.GameID))
	q.Set("_nPage", strconv.Itoa(opts.Page))
	q.Set("_nPerpage", strconv.Itoa(opts.PerPage))
	if opts.Sort != SortDefault {
		q.Set("_sSort", string(opts.Sort))
	}

	var resp indexResponse
	if err := c.get(ctx, "Mod/Index", q, &resp); err != nil {
		return nil, err
	}

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

// AllMods iterates over every mod starting at opts.Page, fetching pages as needed.
// Iteration stops after the last page, or after yielding the first error.
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
