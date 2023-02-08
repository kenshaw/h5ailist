# About

Package `h5ailist` is a simple client for retrieving files/directories from a
remote [h5ai](https://larsjung.de/h5ai/) directory list frontend.

## Using

```go
package h5ailist_test

import (
	"context"
	"fmt"

	"github.com/kenshaw/h5ailist"
)

func Example() {
	var items []h5ailist.Item
	if err := h5ailist.Walk(context.Background(), "https://larsjung.de/h5ai/demo/file preview", func(n string, item *h5ailist.Item, err error) error {
		switch {
		case err != nil:
			return err
		}
		items = append(items, *item)
		return nil
	}); err != nil {
		panic(err)
	}
	var directories, files int64
	for i, item := range items {
		fmt.Printf("%d: %q\n", i, item.Href)
		if item.IsDir() {
			directories++
		} else {
			files++
		}
	}
	fmt.Printf("items: %d directories: %d files: %d\n", len(items), directories, files)
	// Output:
	// 0: "/h5ai/demo/file%20preview/"
	// 1: "/h5ai/demo/file preview/class_cli.py"
	// 2: "/h5ai/demo/file preview/image-1.jpg"
	// 3: "/h5ai/demo/file preview/image-2.jpg"
	// 4: "/h5ai/demo/file preview/image-3.jpg"
	// 5: "/h5ai/demo/file preview/modulejs-1.14.0.js"
	// 6: "/h5ai/demo/file preview/options.css"
	// 7: "/h5ai/demo/file preview/text.md"
	// items: 8 directories: 1 files: 7
}
```
