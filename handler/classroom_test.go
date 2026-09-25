package handler

import (
	"strings"
	"testing"

	"soln-teachermodule/view/ui"
)

func TestValidateCreateClassroom(t *testing.T) {
	tests := []struct {
		name     string
		params   ui.CreateParams
		wantErrs ui.CreateErrors
	}{
		{
			name:   "blank class name",
			params: ui.CreateParams{Classname: "", Section: "1"},
			wantErrs: ui.CreateErrors{
				Classname: "Class name is required.",
			},
		},
		{
			name:   "blank section",
			params: ui.CreateParams{Classname: "Math 6", Section: ""},
			wantErrs: ui.CreateErrors{
				Section: "Section is required.",
			},
		},
		{
			name:   "class name exactly 100 characters is valid",
			params: ui.CreateParams{Classname: strings.Repeat("a", 100), Section: "1"},
		},
		{
			name:   "class name 101 characters is too long",
			params: ui.CreateParams{Classname: strings.Repeat("a", 101), Section: "1"},
			wantErrs: ui.CreateErrors{
				Classname: "Class name must be 100 characters or fewer.",
			},
		},
		{
			name:   "section exactly 100 characters is valid",
			params: ui.CreateParams{Classname: "Math 6", Section: strings.Repeat("a", 100)},
		},
		{
			name:   "section 101 characters is too long",
			params: ui.CreateParams{Classname: "Math 6", Section: strings.Repeat("a", 101)},
			wantErrs: ui.CreateErrors{
				Section: "Section must be 100 characters or fewer.",
			},
		},
		{
			name:   "description exactly 200 characters is valid",
			params: ui.CreateParams{Classname: "Math 6", Section: "1", Description: strings.Repeat("a", 200)},
		},
		{
			name:   "description 201 characters is too long",
			params: ui.CreateParams{Classname: "Math 6", Section: "1", Description: strings.Repeat("a", 201)},
			wantErrs: ui.CreateErrors{
				Description: "Description must be 200 characters or fewer.",
			},
		},
		{
			name:   "all valid, description omitted",
			params: ui.CreateParams{Classname: "Math 6", Section: "St. Anne"},
		},
		{
			name:   "blank name and over-length section both report",
			params: ui.CreateParams{Classname: "", Section: strings.Repeat("a", 101)},
			wantErrs: ui.CreateErrors{
				Classname: "Class name is required.",
				Section:   "Section must be 100 characters or fewer.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateCreateClassroom(tt.params); got != tt.wantErrs {
				t.Errorf("validateCreateClassroom(%+v) = %+v, want %+v", tt.params, got, tt.wantErrs)
			}
		})
	}
}
