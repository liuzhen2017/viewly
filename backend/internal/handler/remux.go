package handler

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	remuxCDNBase string
	remuxRegion  string
	remuxBucket  string
)

// remuxQueue serializes faststart jobs (disk + CPU friendly on the small box).
var remuxQueue = make(chan remuxJob, 32)

type remuxJob struct{ key string }

func init() {
	go func() {
		for j := range remuxQueue {
			remuxFaststart(j.key)
		}
	}()
}

// enqueueRemux moves the mp4 index (moov atom) to the head of the file so
// browsers can start playback immediately instead of fetching the tail of a
// 200MB file first. Stream copy — no re-encode, no quality loss.
func enqueueRemux(key string) {
	if strings.HasSuffix(key, ".mp4") {
		select {
		case remuxQueue <- remuxJob{key: key}:
		default: // queue full: skip, file stays playable-but-slow
		}
	}
}

func remuxFaststart(key string) {
	tmp, err := os.MkdirTemp("/tmp", "remux-*")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmp)
	in := filepath.Join(tmp, "in.mp4")
	out := filepath.Join(tmp, "out.mp4")

	// fetch current object via CDN (bucket policy only allows CF IPs)
	url := strings.TrimRight(remuxCDNBase, "/") + "/" + key
	resp, gerr := http.Get(url)
	err = gerr
	if err != nil || resp.StatusCode != 200 {
		return
	}
	f, err := os.Create(in)
	if err != nil {
		resp.Body.Close()
		return
	}
	_, _ = io.Copy(f, resp.Body)
	f.Close()
	resp.Body.Close()

	// remux: copy streams, put moov first
	if err := exec.Command("ffmpeg", "-nostdin", "-y", "-loglevel", "error",
		"-i", in, "-c", "copy", "-movflags", "+faststart", out).Run(); err != nil {
		return
	}
	inF, err := os.Open(out)
	if err != nil {
		return
	}
	defer inF.Close()
	st, _ := inF.Stat()

	ctx, cancel := context.WithTimeout(context.Background(), 15*60*1e9)
	defer cancel()
	cfg, _ := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(remuxRegion))
	_, _ = s3.NewFromConfig(cfg).PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(remuxBucket),
		Key:         aws.String(key),
		Body:        inF,
		ContentType: aws.String("video/mp4"),
	})
	_ = st
}
