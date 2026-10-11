package banana

import "time"

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
	InitialVisibility string       `json:"_sInitialVisibility"` // one of the Visibility constants
	WasFeatured       bool         `json:"_bWasFeatured"`
	LikeCount         int          `json:"_nLikeCount"`
	ViewCount         int          `json:"_nViewCount"`
	PostCount         int          `json:"_nPostCount"`
}

func (m Mod) AddedAt() time.Time { return time.Unix(m.DateAdded, 0) }

// When the mod's page was last edited, which isn't necessarily an update.
func (m Mod) ModifiedAt() time.Time { return time.Unix(m.DateModified, 0) }

// The zero time if the mod has never been updated.
func (m Mod) UpdatedAt() time.Time {
	if m.DateUpdated == 0 {
		return time.Time{}
	}

	return time.Unix(m.DateUpdated, 0)
}

// How GameBanana initially presents a submission, which it decides from the
// submission's content ratings.
const (
	VisibilityShow = "show" // shown as normal, e.g. a mod rated for tobacco use
	VisibilityWarn = "warn" // shown behind a warning, e.g. skimpy attire
	VisibilityHide = "hide" // hidden until clicked, e.g. nudity or sexual content
)

// Mods shown behind a warning count, as well as ones hidden until clicked.
func (m Mod) IsNSFW() bool {
	return m.InitialVisibility == VisibilityWarn || m.InitialVisibility == VisibilityHide
}

// Falls back to when the mod was added if it has never been updated.
func (m Mod) LastUpdated() time.Time {
	if m.DateUpdated == 0 {
		return m.AddedAt()
	}

	return m.UpdatedAt()
}

func (m Mod) Thumbnail() (PreviewImage, bool) { return m.PreviewMedia.First() }

// For a sound, Metadata.AudioURL is a preview of how it sounds.
type PreviewMedia struct {
	Images   []PreviewImage `json:"_aImages"`
	Metadata struct {
		AudioURL string `json:"_sAudioUrl"` // optional
	} `json:"_aMetadata"`
}

func (p PreviewMedia) First() (PreviewImage, bool) {
	if len(p.Images) == 0 {
		return PreviewImage{}, false
	}

	return p.Images[0], true
}

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

func (p PreviewImage) URL() string { return p.join(p.File) }

// The largest variant no wider than maxWidth (100, 220 or 530) that's present,
// or the full-size image if there's none.
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

type Submitter struct {
	ID         int    `json:"_idRow"`
	Name       string `json:"_sName"`
	ProfileURL string `json:"_sProfileUrl"`
	AvatarURL  string `json:"_sAvatarUrl"`
}

type Game struct {
	ID         int    `json:"_idRow"`
	Name       string `json:"_sName"`
	ProfileURL string `json:"_sProfileUrl"`
	IconURL    string `json:"_sIconUrl"`
}

// A category (e.g. "Skins") or subcategory (e.g. a hero's name).
type Category struct {
	Name       string `json:"_sName"`
	ProfileURL string `json:"_sProfileUrl"`
	IconURL    string `json:"_sIconUrl"`
}
