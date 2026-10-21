package demopath

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"
)

func TestStaticPathVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct{ Path, Query, Want string }
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		q, err := url.ParseQuery(v.Query)
		if err != nil {
			t.Fatal(err)
		}
		if got := StaticPath(v.Path, q); got != v.Want {
			t.Errorf("StaticPath(%q, %q) = %q; want %q", v.Path, v.Query, got, v.Want)
		}
	}
}

func TestRewrite(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"/classroom/students?classroom_id=1#add", "/classroom/students/~classroom_id=1/#add", true},
		{"/home", "/home/", true},
		{"/public/styles.css", "/public/styles.css", false},
		{"#by-question", "#by-question", false},
		{"https://cal.example/x", "https://cal.example/x", false},
		{"//cdn.example/x", "//cdn.example/x", false},
	}
	for _, tt := range tests {
		got, ok := Rewrite(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Rewrite(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
