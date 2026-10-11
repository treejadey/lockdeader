package banana

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDownload(t *testing.T) {
	var gotUA string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.UserAgent()
		if r.URL.Path == "/missing.mp3" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("sound"))
	})

	data, err := c.Download(context.Background(), c.BaseURL+"/preview.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sound" {
		t.Errorf("data = %q", data)
	}
	if gotUA != defaultUserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, defaultUserAgent)
	}

	_, err = c.Download(context.Background(), c.BaseURL+"/missing.mp3")
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("err = %v, want an HTTP 404 error", err)
	}
}

func TestOpenDownload(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dl/1":
			http.Redirect(w, r, "/files/mod.zip", http.StatusFound)
		case "/files/mod.zip":
			w.Header().Set("Content-Length", "7")
			w.Write([]byte("archive"))
		case "/slow.zip":
			// Takes longer than the client's time limit for API calls.
			w.Write([]byte("slow"))
			w.(http.Flusher).Flush()
			time.Sleep(150 * time.Millisecond)
			w.Write([]byte("ly"))
		default:
			http.NotFound(w, r)
		}
	})
	c.HTTPClient.Timeout = 50 * time.Millisecond

	body, size, err := c.OpenDownload(context.Background(), c.BaseURL+"/dl/1")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(body)
	body.Close()
	if string(data) != "archive" || size != 7 {
		t.Errorf("got %q, size %d; want \"archive\", size 7", data, size)
	}

	body, _, err = c.OpenDownload(context.Background(), c.BaseURL+"/slow.zip")
	if err != nil {
		t.Fatal(err)
	}
	data, err = io.ReadAll(body)
	body.Close()
	if err != nil || string(data) != "slowly" {
		t.Errorf("slow download: got %q, %v; want \"slowly\"", data, err)
	}

	if _, _, err := c.OpenDownload(context.Background(), c.BaseURL+"/missing.zip"); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("err = %v, want an HTTP 404 error", err)
	}
}
