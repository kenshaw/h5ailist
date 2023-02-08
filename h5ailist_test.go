package h5ailist

import (
	"context"
	"testing"
)

func TestList(t *testing.T) {
	items, err := List(context.Background(), "https://larsjung.de/h5ai/demo/file preview", WithLogf(t.Logf))
	if err != nil {
		t.Fatalf("expected no errors, got: %v", err)
	}
	for i, item := range items {
		t.Logf("%d: %v %q %d", i, item.Time, item.Href, item.FileSize())
	}
}

func TestItems(t *testing.T) {
	items, err := Items(context.Background(), "https://larsjung.de/h5ai/demo/file%20preview/", WithLogf(t.Logf))
	if err != nil {
		t.Fatalf("expected no errors, got: %v", err)
	}
	for i, item := range items {
		t.Logf("%d: %v %q %d", i, item.Time, item.Href, item.FileSize())
	}
}

func TestWalk(t *testing.T) {
	var items []Item
	if err := Walk(context.Background(), "https://larsjung.de/h5ai/demo", func(n string, item *Item, err error) error {
		switch {
		case err != nil:
			return err
		}
		items = append(items, *item)
		return nil
	}, WithLogf(t.Logf)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	var directories, files, size int64
	for i, item := range items {
		sz := item.FileSize()
		t.Logf("%d: %v %q %d", i, item.Time, item.Href, sz)
		if item.IsDir() {
			directories++
		} else {
			files++
		}
		size += sz
	}
	t.Logf("items: %d directories: %d files: %d size: %d", len(items), directories, files, size)
}

func TestGet(t *testing.T) {
	buf, err := Get(context.Background(), "https://larsjung.de/h5ai/demo/file%20preview/text.md", WithLogf(t.Logf))
	if err != nil {
		t.Fatalf("expected no errors, got: %v", err)
	}
	t.Logf("size: %d", len(buf))
}
