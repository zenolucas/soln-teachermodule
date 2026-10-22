package main

// export-demo builds the static showcase: every read-only page and htmx fragment of the
// teacher portal, pre-rendered from the generated showcase data, ready for a static host.
//
//	go run . export-demo -out dist -db-root-dsn 'root@tcp(127.0.0.1:3306)/'
//
// It loads the schema and showcase data into a throwaway database (soln_demo_export, never
// the dev database), serves the real router in-process with DEMO_MODE on, logs in as the
// showcase teacher, and crawls: every <a href>, every hx-get (with the values of the form it
// submits), and the statistics scene picker's options. Each response is written to
// demopath.StaticPath(url)/index.html, and the links in it are rewritten to match; htmx's own
// requests are mapped the same way in the browser by public/js/demo.js. Mutating routes are
// never requested. Any page or fragment that doesn't answer 200 (or a redirect) fails the
// export, which doubles as a broken-link check.

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"soln-teachermodule/demopath"
	"soln-teachermodule/handler"
	"soln-teachermodule/view/errorpage"

	"golang.org/x/net/html"
)

// exportMarker marks a directory as export output, so a rerun may clear it - and so a
// mistyped -out never wipes anything else.
const exportMarker = ".soln-demo-export"

// exportDenied are the routes the crawler must never request, even if a link to one
// turns up: everything that writes, and the game's API.
var exportDenied = []string{"/add/", "/update/", "/delete/", "/createclassroom", "/addstudents", "/logout", "/game/"}

func runExportDemo(args []string) error {
	flags := flag.NewFlagSet("export-demo", flag.ExitOnError)
	out := flags.String("out", "dist", "output directory (created, or replaced if it holds an earlier export)")
	rootDSN := flags.String("db-root-dsn", os.Getenv("SOLN_EXPORT_DB_ROOT_DSN"), "MariaDB DSN allowed to create databases and users (default $SOLN_EXPORT_DB_ROOT_DSN)")
	booking := flags.String("booking-url", os.Getenv("DEMO_BOOKING_URL"), `scheduling page for "Book a demo" (default $DEMO_BOOKING_URL; empty hides the buttons)`)
	download := flags.String("download-url", os.Getenv("DEMO_DOWNLOAD_URL"), "game download link (default $DEMO_DOWNLOAD_URL; empty copies public/downloads/soln.zip into the export)")
	flags.Parse(args)
	if *rootDSN == "" {
		return errors.New("export-demo: set -db-root-dsn or SOLN_EXPORT_DB_ROOT_DSN")
	}
	if err := prepareOutDir(*out); err != nil {
		return err
	}

	// Throwaway secrets: the session only has to outlive this process. Set before
	// loadShowcaseDatabase, whose .env load never overrides a variable that's already set.
	os.Setenv("SESSION_SECRET", randomHex(32))
	os.Setenv("GAME_TOKEN_SECRET", randomHex(32))
	os.Setenv("DEMO_MODE", "true")
	os.Setenv("DEMO_BOOKING_URL", *booking)
	os.Setenv("DEMO_DOWNLOAD_URL", *download)
	if _, err := loadShowcaseDatabase(*rootDSN, showcaseDB{Name: "soln_demo_export", User: "soln_demo_export", Pass: randomHex(16)}); err != nil {
		return err
	}
	if err := handler.InitSessionStore(); err != nil {
		return err
	}
	if err := handler.InitGameTokenStore(); err != nil {
		return err
	}
	handler.InitDemo()

	srv := httptest.NewServer(newRouter())
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	e := &exporter{
		base:    srv.URL,
		out:     *out,
		client:  &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		written: map[string]bool{},
	}
	if err := e.login(); err != nil {
		return err
	}

	start := time.Now()
	for _, seed := range []string{"/", "/home", "/about"} {
		e.enqueue(crawlItem{url: seed, from: "(seed)"})
	}
	if err := e.crawl(); err != nil {
		return err
	}
	if err := e.copyPublic(*download == ""); err != nil {
		return err
	}
	if err := e.writeNotFound(); err != nil {
		return err
	}
	slog.Info("static showcase exported", "out", *out, "pages_and_fragments", len(e.written), "took", time.Since(start).Round(time.Millisecond))
	return nil
}

// prepareOutDir creates out, or empties it if it holds an earlier export. Anything else
// already there is refused rather than deleted.
func prepareOutDir(out string) error {
	entries, err := os.ReadDir(out)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case len(entries) == 0:
	default:
		if _, err := os.Stat(filepath.Join(out, exportMarker)); err != nil {
			return fmt.Errorf("export-demo: %s isn't empty and isn't an earlier export (no %s); refusing to overwrite it", out, exportMarker)
		}
		if err := os.RemoveAll(out); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, exportMarker), nil, 0o644)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type exporter struct {
	base    string
	out     string
	client  *http.Client
	queue   []crawlItem
	written map[string]bool // static paths already queued
}

// crawlItem is one GET to export. fragment marks an htmx request (sent with HX-Request,
// as htmx would). form is the form that produced it, if any - see expandFragmentButtons.
type crawlItem struct {
	url      string
	from     string
	fragment bool
	form     *crawlForm
}

func (e *exporter) login() error {
	resp, err := e.client.PostForm(e.base+"/login", url.Values{"username": {"teacher"}, "password": {"teacher"}})
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		return fmt.Errorf("export-demo: showcase teacher login answered %d, want 303", resp.StatusCode)
	}
	return nil
}

func (e *exporter) enqueue(item crawlItem) {
	// /public/ assets aren't crawled: copyPublic copies them as they are.
	u, err := url.Parse(item.url)
	if err != nil || !strings.HasPrefix(item.url, "/") || strings.HasPrefix(item.url, "//") || strings.HasPrefix(u.Path, "/public/") {
		return
	}
	for _, prefix := range exportDenied {
		if strings.HasPrefix(u.Path, prefix) {
			return
		}
	}
	key := demopath.StaticPath(u.Path, u.Query())
	if e.written[key] {
		return
	}
	e.written[key] = true
	e.queue = append(e.queue, item)
}

func (e *exporter) crawl() error {
	for len(e.queue) > 0 {
		item := e.queue[0]
		e.queue = e.queue[1:]
		if err := e.export(item); err != nil {
			return err
		}
	}
	return nil
}

func (e *exporter) export(item crawlItem) error {
	req, err := http.NewRequest(http.MethodGet, e.base+item.url, nil)
	if err != nil {
		return err
	}
	if item.fragment {
		req.Header.Set("HX-Request", "true")
	} else {
		req.Header.Set("Accept", "text/html")
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	u, _ := url.Parse(item.url)
	dest := filepath.Join(e.out, filepath.FromSlash(demopath.StaticPath(u.Path, u.Query())), "index.html")

	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		target := resp.Header.Get("Location")
		e.enqueue(crawlItem{url: target, from: item.url})
		static, _ := demopath.Rewrite(target)
		stub := fmt.Sprintf("<!DOCTYPE html><meta charset=\"utf-8\"><meta http-equiv=\"refresh\" content=\"0; url=%[1]s\"><link rel=\"canonical\" href=\"%[1]s\">", html.EscapeString(static))
		return writeFile(dest, []byte(stub))
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("export-demo: GET %s (linked from %s) answered %d: %s", item.url, item.from, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	links, err := discover(body, item)
	if err != nil {
		return fmt.Errorf("export-demo: parsing %s: %w", item.url, err)
	}
	for _, link := range links {
		e.enqueue(link)
	}
	rewritten, err := rewriteLinks(body)
	if err != nil {
		return fmt.Errorf("export-demo: rewriting %s: %w", item.url, err)
	}
	return writeFile(dest, rewritten)
}

func writeFile(dest string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, b, 0o644)
}

// discover finds everything a visitor could request from this page or fragment.
func discover(body []byte, item crawlItem) ([]crawlItem, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var found []crawlItem
	walk(doc, func(n *html.Node) {
		switch {
		case n.Data == "a" && attr(n, "href") != "":
			found = append(found, crawlItem{url: attr(n, "href"), from: item.url})
		case n.Data == "option" && hasAttr(closest(n, "select"), "data-nav-select"):
			found = append(found, crawlItem{url: attr(n, "value"), from: item.url})
		}
		get := attr(n, "hx-get")
		if get == "" {
			return
		}
		switch {
		case n.Data == "form":
			found = append(found, formRequests(n, get, item.url)...)
		case attr(n, "hx-include") == "closest form" && closest(n, "form") != nil:
			// The add-students search box: with its box empty it sends the same request as
			// the form's own load-time fetch, and demo.js searches that copy in the browser.
			f := readForm(closest(n, "form"))
			found = append(found, crawlItem{url: withQuery(get, f.base), from: item.url, fragment: true})
		default:
			found = append(found, crawlItem{url: get, from: item.url, fragment: true})
		}
	})

	// A fragment fetched by a form can carry that form's pager: submit buttons that live in
	// the fragment but belong to the form (form="students-query"), each resubmitting it with
	// its own name=value added.
	if item.form != nil {
		walk(doc, func(n *html.Node) {
			if n.Data == "button" && attr(n, "form") == item.form.id && attr(n, "name") != "" {
				v := cloneValues(item.form.values)
				v.Set(attr(n, "name"), attr(n, "value"))
				found = append(found, crawlItem{url: withQuery(item.form.action, v), from: item.url, fragment: true, form: item.form})
			}
		})
	}
	return found, nil
}

// crawlForm is an hx-get form's submission, carried with the fragment it fetched.
type crawlForm struct {
	id, action string
	values     url.Values
}

// formValues is a form's controls as the crawler sees them: base holds every text/hidden
// input's current value; choices holds every radio group's and select's options.
type formValues struct {
	base    url.Values
	choices map[string][]string
}

func readForm(form *html.Node) formValues {
	f := formValues{base: url.Values{}, choices: map[string][]string{}}
	walk(form, func(n *html.Node) {
		name := attr(n, "name")
		if name == "" {
			return
		}
		switch n.Data {
		case "input":
			switch attr(n, "type") {
			case "radio":
				f.choices[name] = append(f.choices[name], attr(n, "value"))
			case "", "text", "hidden", "search":
				f.base.Set(name, attr(n, "value"))
			}
		case "select":
			walk(n, func(o *html.Node) {
				if o.Data == "option" {
					f.choices[name] = append(f.choices[name], attr(o, "value"))
				}
			})
		}
	})
	return f
}

// formRequests is every submission of an hx-get form a visitor can make without typing:
// each combination of its radio groups and selects, with the text inputs as rendered (an
// empty search box). The students toolbar is the only such form: 5 filters x 4 sorts per
// classroom, and its pager is picked up from each response (see discover).
//
// The toolbar also gets one page=all copy per sort with no filter, which demo.js searches
// in the browser instead of the search the static site can't run (handler.HandleGetStudents).
func formRequests(form *html.Node, action, from string) []crawlItem {
	f := readForm(form)
	names := make([]string, 0, len(f.choices))
	for name := range f.choices {
		names = append(names, name)
	}
	sort.Strings(names)

	combos := []url.Values{f.base}
	for _, name := range names {
		var next []url.Values
		for _, combo := range combos {
			for _, choice := range f.choices[name] {
				v := cloneValues(combo)
				v.Set(name, choice)
				next = append(next, v)
			}
		}
		combos = next
	}

	id := attr(form, "id")
	var items []crawlItem
	for _, v := range combos {
		items = append(items, crawlItem{url: withQuery(action, v), from: from, fragment: true, form: &crawlForm{id: id, action: action, values: v}})
		if id == "students-query" && v.Get("filter") == "all" {
			all := cloneValues(v)
			all.Set("page", "all")
			items = append(items, crawlItem{url: withQuery(action, all), from: from, fragment: true})
		}
	}
	return items
}

func withQuery(path string, v url.Values) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := u.Query()
	for k, vals := range v {
		q[k] = vals
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vals := range v {
		out[k] = append([]string(nil), vals...)
	}
	return out
}

// rewriteLinks points every navigation link at its static copy: <a href> and the scene
// picker's option values. hx-* attributes are left alone, since demo.js maps those at
// request time, when it knows the form values htmx adds. A tokenizer rather than a full
// parse + render, so everything else is written back byte for byte.
func rewriteLinks(body []byte) ([]byte, error) {
	z := html.NewTokenizer(bytes.NewReader(body))
	var out bytes.Buffer
	inNavSelect := false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return out.Bytes(), nil
			}
			return nil, z.Err()
		case html.StartTagToken, html.SelfClosingTagToken:
			raw := append([]byte(nil), z.Raw()...)
			tok := z.Token()
			if tok.Data == "select" {
				inNavSelect = hasTokenAttr(tok, "data-nav-select")
			}
			key := ""
			switch {
			case tok.Data == "a":
				key = "href"
			case tok.Data == "option" && inNavSelect:
				key = "value"
			}
			changed := false
			for i, a := range tok.Attr {
				if a.Key == key {
					if static, ok := demopath.Rewrite(a.Val); ok {
						tok.Attr[i].Val = static
						changed = true
					}
				}
			}
			if changed {
				out.WriteString(tok.String())
			} else {
				out.Write(raw)
			}
		case html.EndTagToken:
			if name, _ := z.TagName(); string(name) == "select" {
				inNavSelect = false
			}
			out.Write(z.Raw())
		default:
			out.Write(z.Raw())
		}
	}
}

// copyPublic copies the embedded public/ assets, leaving out the ~55MB game download
// unless the export has no other download link (static hosts like Cloudflare Pages cap
// files at 25MiB, so a deployed showcase links to a release asset instead).
func (e *exporter) copyPublic(includeDownload bool) error {
	return fs.WalkDir(FS, "public", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == "public/downloads" && !includeDownload {
				return fs.SkipDir
			}
			return nil
		}
		b, err := FS.ReadFile(path)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(e.out, filepath.FromSlash(path)), b)
	})
}

// writeNotFound writes the 404.html static hosts serve for unknown paths, styled like the
// portal's own error page.
func (e *exporter) writeNotFound() error {
	var b bytes.Buffer
	if err := errorpage.Error(http.StatusNotFound, "This page isn't part of the demo.").Render(context.Background(), &b); err != nil {
		return err
	}
	return writeFile(filepath.Join(e.out, "404.html"), b.Bytes())
}

func walk(n *html.Node, fn func(*html.Node)) {
	if n.Type == html.ElementNode {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

func closest(n *html.Node, tag string) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == tag {
			return p
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, key string) bool {
	if n == nil {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func hasTokenAttr(t html.Token, key string) bool {
	for _, a := range t.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}
