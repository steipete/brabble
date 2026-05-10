package control

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFileWritesDestination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() != nil {
			t.Fatalf("request context already canceled: %v", r.Context().Err())
		}
		_, _ = w.Write([]byte("model-bytes"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "models", "model.bin")
	if err := downloadFile(context.Background(), srv.URL, dest); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != "model-bytes" {
		t.Fatalf("dest contents=%q", got)
	}
	if _, err := os.Stat(dest + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("part file stat err=%v", err)
	}
}

func TestDownloadFileRemovesPartFileOnCopyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = w.Write([]byte("short"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "models", "model.bin")
	if err := downloadFile(context.Background(), srv.URL, dest); err == nil {
		t.Fatal("expected copy error")
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dest stat err=%v", err)
	}
	if _, err := os.Stat(dest + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("part file stat err=%v", err)
	}
}

func TestDownloadFileUsesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := filepath.Join(t.TempDir(), "models", "model.bin")
	if err := downloadFile(ctx, "http://example.invalid/model.bin", dest); err == nil {
		t.Fatal("expected context cancellation error")
	}
	if _, err := os.Stat(dest + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("part file stat err=%v", err)
	}
}
