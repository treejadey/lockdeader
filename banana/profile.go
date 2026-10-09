package banana

import (
	"context"
	"fmt"
)

// ModProfile is everything on a single mod's page, as returned by GetModProfile.
type ModProfile struct {
	ID            int          `json:"_idRow"`
	Name          string       `json:"_sName"`
	ProfileURL    string       `json:"_sProfileUrl"`
	DateAdded     int64        `json:"_tsDateAdded"`
	DateModified  int64        `json:"_tsDateModified"`
	DateUpdated   int64        `json:"_tsDateUpdated"`
	Summary       string       `json:"_sDescription"` // short one-line description
	Text          string       `json:"_sText"`        // full description, as HTML
	Version       string       `json:"_sVersion"`
	PreviewMedia  PreviewMedia `json:"_aPreviewMedia"`
	Submitter     Submitter    `json:"_aSubmitter"`
	Game          Game         `json:"_aGame"`
	Category      Category     `json:"_aCategory"`
	SuperCategory *Category    `json:"_aSuperCategory"` // parent of Category, if it has one
	Files         []File       `json:"_aFiles"`
	DownloadURL   string       `json:"_sDownloadUrl"` // the mod's download page
	DownloadCount int          `json:"_nDownloadCount"`
	LikeCount     int          `json:"_nLikeCount"`
	ViewCount     int          `json:"_nViewCount"`
	PostCount     int          `json:"_nPostCount"`
	IsObsolete    bool         `json:"_bIsObsolete"`
}

// File is a downloadable file attached to a mod.
type File struct {
	ID            int    `json:"_idRow"`
	Name          string `json:"_sFile"`
	Size          int64  `json:"_nFilesize"` // in bytes
	Description   string `json:"_sDescription"`
	DateAdded     int64  `json:"_tsDateAdded"`
	DownloadCount int    `json:"_nDownloadCount"`
	DownloadURL   string `json:"_sDownloadUrl"`
	MD5           string `json:"_sMd5Checksum"`
}

// GetModProfile fetches the full page for the mod with the given id.
func (c *Client) GetModProfile(ctx context.Context, id int) (*ModProfile, error) {
	var p ModProfile
	if err := c.get(ctx, fmt.Sprintf("Mod/%d/ProfilePage", id), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
