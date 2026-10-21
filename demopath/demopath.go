// Package demopath maps portal URLs to the paths their pre-rendered copies are served at
// in the static showcase (see export.go). Static hosts pick a file by path alone and
// ignore the query string, so the query becomes path segments instead:
//
//	/classroom/statistics?minigameID=3&classroom_id=1 → /classroom/statistics/~classroom_id=1/~minigameID=3/
//
// Every static path ends in "/", which static hosts serve as that directory's index.html.
// public/js/demopath.js is a JS port of StaticPath for the requests htmx builds in the
// browser; testdata/vectors.json pins both to the same answers (demopath_test.go and
// demopath.test.js).
package demopath

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// StaticPath returns the static path for path + query. Empty values are dropped (a form
// sends q="" for an empty search box, and that must land on the same file as no q at
// all), keys are sorted, and each remaining value becomes one "~key=value" segment.
func StaticPath(path string, query url.Values) string {
	path = strings.TrimRight(path, "/")
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(path)
	for _, k := range keys {
		for _, v := range query[k] {
			if v == "" {
				continue
			}
			b.WriteString("/~")
			b.WriteString(encode(k))
			b.WriteString("=")
			b.WriteString(encode(v))
		}
	}
	b.WriteString("/")
	return b.String()
}

// Rewrite maps a link as it appears in the portal's HTML (path, query and maybe a
// #fragment) to its static form. It reports false, leaving raw alone, for anything that
// isn't a portal page or fragment: external URLs, bare #fragments, and /public/ assets,
// which the export copies as they are.
func Rewrite(raw string) (string, bool) {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/public/") {
		return raw, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw, false
	}
	out := StaticPath(u.Path, u.Query())
	if u.Fragment != "" {
		out += "#" + u.Fragment
	}
	return out, true
}

// encode keeps [A-Za-z0-9_-] and writes every other byte as ".xx" (lowercase hex). The
// result is safe as a file name and as a URL path segment with nothing for a host to
// percent-decode, and "." never appears unescaped, so a segment can't be "..".
func encode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, ".%02x", c)
		}
	}
	return b.String()
}
