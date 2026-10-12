// Package banana is a minimal client for the GameBanana API (v11).
package banana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://gamebanana.com/apiv11"

	DeadlockGameID = 20948

	// The largest page size the API accepts.
	MaxPerPage = 50

	// Preview images come from here. Files there never change, since new
	// uploads always get a new name, so cached copies never go stale.
	ImageHost = "images.gamebanana.com"
)

var defaultUserAgent = UserAgent("")

// UserAgent is what the client tells GameBanana it is, with the app's
// version ("" leaves it out).
func UserAgent(version string) string {
	if version == "" {
		return "lockdeader <https://github.com/treejadey/lockdeader>"
	}

	return fmt.Sprintf("lockdeader v%s <https://github.com/treejadey/lockdeader>", version)
}

// The zero value isn't usable; use NewClient.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

func NewClient() *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UserAgent:  defaultUserAgent,
	}
}

// path is relative to BaseURL. Responses other than 2xx are returned as
// *APIError.
func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	u := strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")

	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newAPIError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("banana: decoding %s: %w", path, err)
	}

	// Drain so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// url may be on another host than the API, e.g. a preview image or sound.
// Responses other than 200 OK are errors.
func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	body, _, err := c.open(ctx, hc, url)
	if err != nil {
		return nil, err
	}

	defer body.Close()

	return io.ReadAll(body)
}

// For big files, such as a mod's archive (File.DownloadURL). The caller reads
// and closes the body; the size is in bytes, or -1 if the server didn't say.
// Unlike the client's other requests, it has no time limit, since big files
// take a while on slow connections; ctx can cancel it. Responses other than
// 200 OK are errors.
func (c *Client) OpenDownload(ctx context.Context, url string) (io.ReadCloser, int64, error) {
	hc := http.Client{}

	if c.HTTPClient != nil {
		hc = *c.HTTPClient
	}

	hc.Timeout = 0

	return c.open(ctx, &hc, url)
}

func (c *Client) open(ctx context.Context, hc *http.Client, url string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		return nil, 0, fmt.Errorf("banana: downloading %s: %w", url, &APIError{StatusCode: resp.StatusCode})
	}

	return resp.Body, resp.ContentLength, nil
}
