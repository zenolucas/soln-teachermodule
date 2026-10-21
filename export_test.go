package main

import (
	"net/url"
	"sort"
	"strings"
	"testing"
)

// rewriteLinks must touch only <a href> and the scene picker's option values, and write
// everything else back byte for byte - templ's inline <style>/<script>, entities, hx-*.
func TestRewriteLinks(t *testing.T) {
	in := `<!DOCTYPE html><html><head><style>.x{width:50%}</style><script>if (a < b && c) {}</script></head><body>` +
		`<a href="/student/score?userID=5&amp;classroomID=1&amp;sort=name" class="btn">View &amp; more</a>` +
		`<a href="/public/downloads/soln.zip">zip</a><a href="https://cal.example/x">book</a><a href="#by-question">tab</a>` +
		`<div hx-get="/getfractions?minigameID=1&amp;classroomID=1" hx-trigger="load"></div>` +
		`<select data-nav-select><option value="/classroom/statistics?classroom_id=1&amp;minigameID=2" selected>Two</option></select>` +
		`<select name="sort"><option value="name">Name</option></select>` +
		`<img src="/public/x.png"/></body></html>`
	out, err := rewriteLinks([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{
		`<a href="/student/score/~classroomID=1/~sort=name/~userID=5/" class="btn">View &amp; more</a>`,
		`<a href="/public/downloads/soln.zip">zip</a><a href="https://cal.example/x">book</a><a href="#by-question">tab</a>`,
		`<div hx-get="/getfractions?minigameID=1&amp;classroomID=1" hx-trigger="load"></div>`,
		`<option value="/classroom/statistics/~classroom_id=1/~minigameID=2/" selected="">Two</option>`,
		`<select name="sort"><option value="name">Name</option></select>`,
		`<style>.x{width:50%}</style><script>if (a < b && c) {}</script>`,
		`<img src="/public/x.png"/>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rewritten HTML is missing\n  %s\ngot:\n  %s", want, got)
		}
	}
}

// The students toolbar expands to every filter x sort (with an empty search box), plus
// a page=all copy per sort for demo.js's search; the pager buttons in each response then
// resubmit that same combination with their page added.
func TestDiscoverStudentsToolbar(t *testing.T) {
	page := `<form id="students-query" hx-get="/students" hx-trigger="load, change">
		<input type="hidden" name="classroomID" value="1"/>
		<input type="text" name="q" value=""/>
		<input type="radio" name="filter" value="all" checked/><input type="radio" name="filter" value="w1"/>
		<select name="sort"><option value="attention">A</option><option value="name">N</option></select>
	</form>
	<form hx-post="/addstudents"><input type="hidden" name="classroomID" value="1"/>
		<input type="search" name="q" hx-get="/unenrolledstudents" hx-include="closest form"/>
		<input type="checkbox" name="userID" value="9"/></form>
	<a href="/home">home</a>`
	items, err := discover([]byte(page), crawlItem{url: "/classroom/students?classroom_id=1"})
	if err != nil {
		t.Fatal(err)
	}
	var urls []string
	var toolbar crawlItem
	for _, it := range items {
		urls = append(urls, it.url)
		if strings.Contains(it.url, "filter=w1") && strings.Contains(it.url, "sort=name") {
			toolbar = it
		}
	}
	sort.Strings(urls)
	want := []string{
		"/home",
		"/students?classroomID=1&filter=all&page=all&q=&sort=attention",
		"/students?classroomID=1&filter=all&page=all&q=&sort=name",
		"/students?classroomID=1&filter=all&q=&sort=attention",
		"/students?classroomID=1&filter=all&q=&sort=name",
		"/students?classroomID=1&filter=w1&q=&sort=attention",
		"/students?classroomID=1&filter=w1&q=&sort=name",
		"/unenrolledstudents?classroomID=1&q=",
	}
	if strings.Join(urls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("discovered:\n%s\nwant:\n%s", strings.Join(urls, "\n"), strings.Join(want, "\n"))
	}

	fragment := `<div>rows</div><div class="join">` +
		`<button type="submit" name="page" value="1" form="students-query">1</button>` +
		`<button type="submit" name="page" value="2" form="students-query">2</button></div>`
	pages, err := discover([]byte(fragment), toolbar)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("pager produced %d requests; want 2", len(pages))
	}
	u, _ := url.Parse(pages[1].url)
	if q := u.Query(); q.Get("page") != "2" || q.Get("filter") != "w1" || q.Get("sort") != "name" || q.Get("classroomID") != "1" {
		t.Errorf("pager request %s lost the toolbar's values", pages[1].url)
	}
}
