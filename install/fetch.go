package install

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/treejadey/lockdeader/appdb"
	"github.com/treejadey/lockdeader/banana"
)

// Stage is what an install is doing.
type Stage int

const (
	Downloading Stage = iota
	// Taking the VPKs out of the downloaded archives.
	Extracting
)

// Status is how an install is going, as Fetch reports it.
type Status struct {
	Stage    Stage
	Received int64 // bytes of every file downloaded so far
	Total    int64 // bytes in every file
	Pace     Pace  // how fast the download is going
}

// Progress returns how much of the download is done, from 0 to 1.
func (s Status) Progress() float64 {
	if s.Stage != Downloading {
		return 1
	}
	if s.Total <= 0 {
		return 0
	}
	return min(1, float64(s.Received)/float64(s.Total))
}

var (
	// ErrChecksum is returned by Fetch for a download that doesn't match the
	// MD5 that GameBanana has for it, e.g. because it was cut short.
	ErrChecksum = errors.New("install: the download is damaged (its checksum doesn't match)")
	// ErrStalled is returned by Fetch for a download that stopped arriving.
	ErrStalled = errors.New("install: the download stopped")
)

// How often Fetch reports progress, at most.
const reportInterval = 100 * time.Millisecond

// A download is given up on once nothing has arrived for this long.
var stallTimeout = 30 * time.Second

// Fetch downloads a mod's files from GameBanana, checks each against its
// MD5, adds the VPKs in them to the store, and returns what the database
// needs to know about them. VPKs are enabled to begin with. Each archive is
// deleted once its VPKs are in the store.
//
// report is called with how it's going: every so often while downloading,
// and once when extraction starts. It's called on Fetch's goroutine.
func Fetch(ctx context.Context, client *banana.Client, store Store, files []banana.File, report func(Status)) ([]appdb.ModFile, error) {
	status := Status{Stage: Downloading}
	for _, f := range files {
		status.Total += max(f.Size, 0)
	}
	var meter SpeedMeter
	lastReport := time.Time{}
	report(status)

	archives := make([]string, len(files))
	defer func() {
		for _, path := range archives {
			if path != "" {
				os.Remove(path)
			}
		}
	}()

	for i, f := range files {
		out, err := store.createPartial(f.ID)
		if err != nil {
			return nil, fmt.Errorf("install: %w", err)
		}
		archives[i] = out.Name()
		before := status.Received
		err = download(ctx, client, f, out, func(received int64) {
			status.Received = before + received
			now := time.Now()
			if now.Sub(lastReport) >= reportInterval {
				lastReport = now
				meter.Add(now, status.Received)
				status.Pace = meter.Pace()
				report(status)
			}
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
	}

	status.Stage = Extracting
	report(status)
	result := make([]appdb.ModFile, len(files))
	for i, f := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		vpks, err := Extract(archives[i], f.Name, store)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		result[i] = appdb.ModFile{
			GameBananaID: f.ID,
			Name:         f.Name,
			Size:         f.Size,
			MD5:          strings.ToLower(f.MD5),
			UploadedAt:   time.Unix(f.DateAdded, 0),
			DownloadedAt: time.Now(),
		}
		for _, v := range vpks {
			result[i].VPKs = append(result[i].VPKs, appdb.VPK{
				Path: v.Path, SHA256: v.SHA256, Size: v.Size, Enabled: true,
			})
		}
	}
	return result, nil
}

// download saves f to out, which it closes, calling progress with how many
// bytes have arrived as they do, and checks it against its MD5.
func download(ctx context.Context, client *banana.Client, f banana.File, out *os.File, progress func(received int64)) error {
	defer out.Close()

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stall := time.AfterFunc(stallTimeout, func() { cancel(ErrStalled) })
	defer stall.Stop()

	body, _, err := client.OpenDownload(ctx, f.DownloadURL)
	if err != nil {
		return cause(ctx, err)
	}
	defer body.Close()

	hash := md5.New()
	w := io.MultiWriter(out, hash)
	buf := make([]byte, 64<<10)
	var received int64
	for {
		n, err := body.Read(buf)
		if n > 0 {
			stall.Reset(stallTimeout)
			if _, err := w.Write(buf[:n]); err != nil {
				return err
			}
			received += int64(n)
			progress(received)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return cause(ctx, err)
		}
	}
	if err := out.Close(); err != nil {
		return err
	}

	if f.MD5 != "" && !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), f.MD5) {
		return ErrChecksum
	}
	return nil
}

// cause returns why ctx was cancelled, if it was, rather than the error that
// cancelling it caused.
func cause(ctx context.Context, err error) error {
	if c := context.Cause(ctx); c != nil {
		return c
	}
	return err
}
