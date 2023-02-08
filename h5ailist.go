// Package h5ailist retrieves a directory listing from a remote h5ai list.
package h5ailist

import (
	"context"
)

// WalkFunc is the walk func signature.
type WalkFunc func(path string, item *Item, err error) error

// List retrieves the listing for the url.
func List(ctx context.Context, urlstr string, opts ...Option) ([]Item, error) {
	return New(append([]Option{WithURL(urlstr)}, opts...)...).List(ctx)
}

// Items retrieves the items for the url.
func Items(ctx context.Context, urlstr string, opts ...Option) ([]Item, error) {
	return New(append([]Option{WithURL(urlstr)}, opts...)...).Items(ctx)
}

// Get retrieves the item.
func Get(ctx context.Context, urlstr string, opts ...Option) ([]byte, error) {
	return New(append([]Option{WithURL(urlstr)}, opts...)...).Get(ctx)
}

// Walk walks all items.
func Walk(ctx context.Context, urlstr string, f WalkFunc, opts ...Option) error {
	return New(append([]Option{WithURL(urlstr)}, opts...)...).Walk(ctx, "/", f)
}
