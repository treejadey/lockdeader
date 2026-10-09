package banana

import "time"

// Mod is a single entry from a mod listing.
type Mod struct {
	ID                int          `json:"_idRow"`
	Name              string       `json:"_sName"`
	ProfileURL        string       `json:"_sProfileUrl"`
	DateAdded         int64        `json:"_tsDateAdded"`
	DateModified      int64        `json:"_tsDateModified"`
	DateUpdated       int64        `json:"_tsDateUpdated"`
	HasFiles          bool         `json:"_bHasFiles"`
	Tags              []string     `json:"_aTags"`
	PreviewMedia      PreviewMedia `json:"_aPreviewMedia"`
	Submitter         Submitter    `json:"_aSubmitter"`
	Game              Game         `json:"_aGame"`
	RootCategory      Category     `json:"_aRootCategory"`
	SubCategory       *Category    `json:"_aSubCategory"`
	Version           string       `json:"_sVersion"`
	IsObsolete        bool         `json:"_bIsObsolete"`
	HasContentRatings bool         `json:"_bHasContentRatings"`
	WasFeatured       bool         `json:"_bWasFeatured"`
	LikeCount         int          `json:"_nLikeCount"`
	ViewCount         int          `json:"_nViewCount"`
	PostCount         int          `json:"_nPostCount"`
}

// AddedAt returns when the mod was first submitted.
func (m Mod) AddedAt() time.Time { return time.Unix(m.DateAdded, 0) }

// ModifiedAt returns when the mod's page was last modified.
func (m Mod) ModifiedAt() time.Time { return time.Unix(m.DateModified, 0) }

// UpdatedAt returns when the mod last received an update, or the zero time if it never has.
func (m Mod) UpdatedAt() time.Time {
	if m.DateUpdated == 0 {
		return time.Time{}
	}
	return time.Unix(m.DateUpdated, 0)
}

// LastUpdated returns when the mod last received an update, or when it was
// added if it has never been updated.
func (m Mod) LastUpdated() time.Time {
	if m.DateUpdated == 0 {
		return m.AddedAt()
	}
	return m.UpdatedAt()
}

// Thumbnail returns the first preview image, if any.
func (m Mod) Thumbnail() (PreviewImage, bool) { return m.PreviewMedia.First() }

// PreviewMedia holds a submission's preview images.
type PreviewMedia struct {
	Images []PreviewImage `json:"_aImages"`
}

// First returns the first preview image, if any.
func (p PreviewMedia) First() (PreviewImage, bool) {
	if len(p.Images) == 0 {
		return PreviewImage{}, false
	}
	return p.Images[0], true
}

// PreviewImage is a single preview image, available in several sizes.
// The sized variants (File100/220/530) are not always present.
type PreviewImage struct {
	Type    string `json:"_sType"`
	BaseURL string `json:"_sBaseUrl"`
	File    string `json:"_sFile"`
	Caption string `json:"_sCaption"` // optional

	File100   string `json:"_sFile100"`
	Width100  int    `json:"_wFile100"`
	Height100 int    `json:"_hFile100"`

	File220   string `json:"_sFile220"`
	Width220  int    `json:"_wFile220"`
	Height220 int    `json:"_hFile220"`

	File530   string `json:"_sFile530"`
	Width530  int    `json:"_wFile530"`
	Height530 int    `json:"_hFile530"`
}

// URL returns the full-size image URL.
func (p PreviewImage) URL() string { return p.join(p.File) }

// SizedURL returns the URL of the largest variant whose width is at most maxWidth
// (100, 220 or 530), falling back to smaller variants and finally the full-size image.
func (p PreviewImage) SizedURL(maxWidth int) string {
	candidates := []struct {
		width int
		file  string
	}{{530, p.File530}, {220, p.File220}, {100, p.File100}}
	for _, c := range candidates {
		if c.width <= maxWidth && c.file != "" {
			return p.join(c.file)
		}
	}
	return p.URL()
}

func (p PreviewImage) join(file string) string {
	if file == "" {
		return ""
	}
	return p.BaseURL + "/" + file
}

// Submitter is the member who uploaded a submission.
type Submitter struct {
	ID         int    `json:"_idRow"`
	Name       string `json:"_sName"`
	ProfileURL string `json:"_sProfileUrl"`
	AvatarURL  string `json:"_sAvatarUrl"`
}

// Game identifies the game a submission belongs to.
type Game struct {
	ID         int    `json:"_idRow"`
	Name       string `json:"_sName"`
	ProfileURL string `json:"_sProfileUrl"`
	IconURL    string `json:"_sIconUrl"`
}

// Category is a mod category (e.g. "Skins") or subcategory (e.g. a hero name).
type Category struct {
	Name       string `json:"_sName"`
	ProfileURL string `json:"_sProfileUrl"`
	IconURL    string `json:"_sIconUrl"`
}
