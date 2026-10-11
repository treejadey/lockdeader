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
	// DefaultBaseURL is the root of the GameBanana v11 API.
	DefaultBaseURL = "https://gamebanana.com/apiv11"

	// DeadlockGameID is GameBanana's id for Deadlock.
	DeadlockGameID = 20948

	// MaxPerPage is the largest page size the API accepts.
	MaxPerPage = 50

	// ImageHost serves preview images. A file there never changes: new
	// uploads always get a new name, so cached copies never go stale.
	ImageHost = "images.gamebanana.com"

	ClientVersion = "0.1.0"
)

var (
	defaultUserAgent = fmt.Sprintf("lockdeader v%s <https://github.com/treejadey/lockdeader>", ClientVersion)
)

// Client talks to the GameBanana API. The zero value is not usable; use NewClient.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient returns a Client with default settings.
func NewClient() *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UserAgent:  defaultUserAgent,
	}
}

// get performs a GET request against path (relative to BaseURL) and decodes
// the JSON response body into out. Non-2xx responses are returned as *APIError.
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

// Download fetches the file at url, such as a preview image or sound, which
// may be on another host than the API. Responses other than 200 OK are errors.
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

// OpenDownload starts downloading the file at url, such as a mod's archive
// (File.DownloadURL), and returns its contents for the caller to read and
// close, along with its size in bytes, or -1 if the server didn't say.
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
		return nil, 0, fmt.Errorf("banana: downloading %s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, resp.ContentLength, nil
}
