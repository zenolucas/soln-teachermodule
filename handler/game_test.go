package handler

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCleanSavePatch(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    map[string]any
		wantErr bool
	}{
		{
			name: "keeps game keys, drops body student_id",
			body: `{"student_id": 99, "current_floor": 2, "player_badges": {"bowl": true}}`,
			want: map[string]any{"current_floor": 2.0, "player_badges": map[string]any{"bowl": true}},
		},
		{
			name: "drops top-level nulls so they can't delete a key",
			body: `{"current_quest": null, "rock_removed": true}`,
			want: map[string]any{"rock_removed": true},
		},
		{
			name: "empty object is a valid no-op save",
			body: `{}`,
			want: map[string]any{},
		},
		{name: "array rejected", body: `[1, 2]`, wantErr: true},
		{name: "bare null rejected", body: `null`, wantErr: true},
		{name: "not JSON rejected", body: `current_floor=2`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanSavePatch([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotMap map[string]any
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("output isn't JSON: %v", err)
			}
			if !reflect.DeepEqual(gotMap, tt.want) {
				t.Errorf("got %v, want %v", gotMap, tt.want)
			}
		})
	}
}
