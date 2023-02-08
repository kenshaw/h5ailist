package h5ailist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kenshaw/httplog"
	"golang.org/x/net/publicsuffix"
)

// Client is a h5ai client.
type Client struct {
	cl        *http.Client
	URL       string
	UserAgent string
	Transport http.RoundTripper
	Jar       http.CookieJar
	err       error
	mu        sync.Mutex
}

// New creates a new h5ai client.
func New(opts ...Option) *Client {
	cl := &Client{
		Transport: http.DefaultTransport,
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	}
	for _, o := range opts {
		o(cl)
	}
	if cl.cl == nil {
		jar, _ := cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
		cl.cl = &http.Client{
			Jar:       jar,
			Transport: cl.Transport,
		}
	}
	return cl
}

// BuildRequest builds a http request.
func (cl *Client) BuildRequest(method, urlstr string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, urlstr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host)
	if cl.UserAgent != "" {
		req.Header.Set("User-Agent", cl.UserAgent)
	}
	return req, nil
}

// do executes a request against the context and client.
func (cl *Client) do(ctx context.Context, method, urlstr string, request, v interface{}) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(request); err != nil {
		return err
	}
	req, err := cl.BuildRequest(method, urlstr, buf)
	if err != nil {
		return err
	}
	res, err := cl.cl.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode != http.StatusOK:
		return fmt.Errorf("status != %d", http.StatusOK)
	case v == nil:
		return nil
	}
	dec := json.NewDecoder(res.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// init inits the path.
func (cl *Client) init(ctx context.Context, urlstr string) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if err := cl.err; err != nil {
		return err
	}
	if !strings.HasSuffix(urlstr, "/") {
		if i := strings.LastIndexByte(urlstr, '/'); i != -1 {
			urlstr = urlstr[:i+1]
		} else {
			urlstr += "/"
		}
	}
	err := cl.do(
		ctx,
		"POST",
		urlstr,
		map[string]interface{}{
			"action":  "get",
			"langs":   true,
			"options": true,
			"setup":   true,
			"theme":   true,
			"types":   true,
		},
		nil,
	)
	cl.err = err
	return err
}

// Do executes a request against the context and client.
func (cl *Client) Do(ctx context.Context, method, urlstr string, request, v interface{}) error {
	if err := cl.init(ctx, urlstr); err != nil {
		return err
	}
	return cl.do(ctx, method, urlstr, request, v)
}

// list returns the list at the path.
func (cl *Client) list(ctx context.Context, urlstr, href string, filter bool) ([]Item, error) {
	var res struct {
		Items []Item `json:"items,omitempty"`
	}
	if !strings.HasSuffix(urlstr, "/") {
		urlstr += "/"
	}
	if err := cl.Do(ctx, "POST", urlstr, map[string]interface{}{
		"action": "get",
		"items": map[string]interface{}{
			"href": href,
			"what": 1,
		},
	}, &res); err != nil {
		return nil, err
	}
	if !filter {
		return res.Items, nil
	}
	h, err := url.PathUnescape(href)
	if err != nil {
		return nil, err
	}
	var items []Item
	for _, item := range res.Items {
		if item.Href, err = url.PathUnescape(item.Href); err != nil {
			return nil, err
		}
		if item.Href != h && strings.HasPrefix(item.Href, h) {
			items = append(items, item)
		}
	}
	return items, nil
}

// List returns the list at the path.
func (cl *Client) List(ctx context.Context, paths ...string) ([]Item, error) {
	urlstr, href, err := cl.Href(paths...)
	if err != nil {
		return nil, err
	}
	return cl.list(ctx, urlstr, href, false)
}

// Items returns the items at the path.
func (cl *Client) Items(ctx context.Context, paths ...string) ([]Item, error) {
	urlstr, href, err := cl.Href(paths...)
	if err != nil {
		return nil, err
	}
	return cl.list(ctx, urlstr, href, true)
}

// Get retrieves a file.
func (cl *Client) Get(ctx context.Context, paths ...string) ([]byte, error) {
	urlstr, _, err := cl.Href(paths...)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(urlstr, "/") {
		return nil, fmt.Errorf("invalid url %s", urlstr)
	}
	if err := cl.init(ctx, urlstr); err != nil {
		return nil, err
	}
	req, err := cl.BuildRequest("GET", urlstr, nil)
	if err != nil {
		return nil, err
	}
	res, err := cl.cl.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("status != %d", http.StatusOK)
	}
	return io.ReadAll(res.Body)
}

// Walk walks the root.
func (cl *Client) Walk(ctx context.Context, root string, fn WalkFunc) error {
	urlstr, href, err := cl.Href()
	if err != nil {
		return err
	}
	if !strings.HasSuffix(urlstr, "/") {
		urlstr += "/"
	}
	if !strings.HasSuffix(href, "/") {
		href += "/"
	}
	if err = cl.init(ctx, urlstr); err != nil {
		err = fn(root, nil, err)
	} else {
		err = cl.walk(ctx, root, &Item{
			Fetched: true,
			Href:    href,
			Managed: true,
			Time:    Time{Time: time.Now()},
		}, fn)
	}
	if err == SkipDir || err == SkipAll {
		return nil
	}
	return err
}

// walk walks the path.
func (cl *Client) walk(ctx context.Context, pathstr string, item *Item, fn WalkFunc) error {
	if !item.IsDir() {
		return fn(item.Href, item, nil)
	}
	urlstr, _, err := cl.Href()
	if err != nil {
		return err
	}
	items, err := cl.list(ctx, urlstr, item.Href, true)
	if e := fn(item.Href, item, err); err != nil || e != nil {
		return e
	}
	for _, item := range items {
		if err := cl.walk(ctx, item.Href, &item, fn); err != nil && err != SkipDir {
			return err
		}
	}
	return nil
}

// Href returns the href path for the URL combined with paths.
func (cl *Client) Href(paths ...string) (string, string, error) {
	u, err := url.Parse(cl.URL)
	if err != nil {
		return "", "", err
	}
	u = u.JoinPath(paths...)
	return u.String(), u.EscapedPath(), nil
}

// Item is a item.
type Item struct {
	Fetched bool   `json:"fetched,omitempty"`
	Href    string `json:"href,omitempty"`
	Managed bool   `json:"managed,omitempty"`
	Size    *int64 `json:"size,omitempty"`
	Time    Time   `json:"time,omitempty"`
}

// IsDir returns true when the item is a directory.
func (item Item) IsDir() bool {
	return item.Managed == true
}

// FileSize returns the file size.
func (item Item) FileSize() int64 {
	if item.Size != nil {
		return *item.Size
	}
	return 0
}

// Time is a wrapped time.
type Time struct {
	time.Time
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(buf []byte) error {
	i, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(i)
	return nil
}

// Option is a client option.
type Option func(*Client)

// WithURL is a client option to set the url.
func WithURL(urlstr string) Option {
	return func(cl *Client) {
		cl.URL = urlstr
	}
}

// WithUserAgent is a client option to set the transport.
func WithUserAgent(userAgent string) Option {
	return func(cl *Client) {
		cl.UserAgent = userAgent
	}
}

// WithTransport is a client option to set the transport.
func WithTransport(transport http.RoundTripper) Option {
	return func(cl *Client) {
		cl.Transport = transport
	}
}

// WithLogf is a client option to set a log handler for http requests and
// responses.
func WithLogf(logf interface{}, opts ...httplog.Option) Option {
	return func(cl *Client) {
		cl.Transport = httplog.NewPrefixedRoundTripLogger(cl.Transport, logf, opts...)
	}
}

// WithJar is a client option to set the cookie jar.
func WithJar(jar http.CookieJar) Option {
	return func(cl *Client) {
		cl.Jar = jar
	}
}

// WithHTTPClient is a client option to set the underlying http.Client used.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(cl *Client) {
		cl.cl = httpClient
	}
}

// Error is a error.
type Error string

const (
	SkipDir Error = "skip dir"
	SkipAll Error = "skip all"
)

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}
