package handler

import (
	"net/url"
	"reflect"
	"testing"

	"soln-teachermodule/types"
)

func TestParseStudentQuery(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
		want   StudentQuery
	}{
		{
			name:   "empty values default to all/attention/page 1",
			values: url.Values{},
			want:   StudentQuery{Filter: "all", Sort: "attention", Page: 1},
		},
		{
			name:   "recognized filter and sort are kept",
			values: url.Values{"q": {"ann"}, "filter": {"w2"}, "sort": {"name"}, "page": {"3"}},
			want:   StudentQuery{Q: "ann", Filter: "w2", Sort: "name", Page: 3},
		},
		{
			name:   "unrecognized filter falls back to all",
			values: url.Values{"filter": {"bogus"}},
			want:   StudentQuery{Filter: "all", Sort: "attention", Page: 1},
		},
		{
			name:   "unrecognized sort falls back to attention",
			values: url.Values{"sort": {"bogus"}},
			want:   StudentQuery{Filter: "all", Sort: "attention", Page: 1},
		},
		{
			name:   "zero, negative or non-numeric page falls back to 1",
			values: url.Values{"page": {"0"}},
			want:   StudentQuery{Filter: "all", Sort: "attention", Page: 1},
		},
		{
			name:   "negative page falls back to 1",
			values: url.Values{"page": {"-5"}},
			want:   StudentQuery{Filter: "all", Sort: "attention", Page: 1},
		},
		{
			name:   "non-numeric page falls back to 1",
			values: url.Values{"page": {"abc"}},
			want:   StudentQuery{Filter: "all", Sort: "attention", Page: 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStudentQuery(tc.values)
			if got != tc.want {
				t.Errorf("parseStudentQuery(%v) = %+v, want %+v", tc.values, got, tc.want)
			}
		})
	}
}

// studentsFixture is a 6-student fixture spanning every world, a range of flag
// counts, and -1 ("no data") values for quiz/accuracy, used across
// TestApplyStudentQuery's subtests.
func studentsFixture() []types.StudentInsight {
	return []types.StudentInsight{
		{Student: types.Student{Firstname: "Ann", Lastname: "Baker", UserID: "1"}, World: 1, QuizAvgPct: -1, AccuracyPct: -1},
		{Student: types.Student{Firstname: "Ben", Lastname: "Cole", UserID: "2"}, World: 1, QuizAvgPct: 80, AccuracyPct: 60,
			Flags: []types.Flag{{Kind: types.FlagQuizBelow60}}},
		{Student: types.Student{Firstname: "Cara", Lastname: "Baker", UserID: "3"}, World: 2, QuizAvgPct: 40, AccuracyPct: -1,
			Flags: []types.Flag{{Kind: types.FlagQuizBelow60}, {Kind: types.FlagLowAccuracy}}},
		{Student: types.Student{Firstname: "Dan", Lastname: "Diaz", UserID: "4"}, World: 2, QuizAvgPct: 90, AccuracyPct: 90},
		{Student: types.Student{Firstname: "Eve", Lastname: "Ames", UserID: "5"}, World: 3, QuizAvgPct: 60, AccuracyPct: 30,
			Flags: []types.Flag{{Kind: types.FlagLowAccuracy}}},
		{Student: types.Student{Firstname: "Fay", Lastname: "Cole", UserID: "6"}, World: 3, QuizAvgPct: -1, AccuracyPct: 70},
	}
}

func userIDs(students []types.StudentInsight) []string {
	ids := make([]string, len(students))
	for i, s := range students {
		ids[i] = s.UserID
	}
	return ids
}

func TestApplyStudentQueryFilters(t *testing.T) {
	all := studentsFixture()

	tests := []struct {
		name       string
		q          StudentQuery
		wantIDs    []string
		wantTotal  int
		wantCounts map[string]int
	}{
		{
			name:       "all, sorted by name",
			q:          StudentQuery{Filter: "all", Sort: "name", Page: 1},
			wantIDs:    []string{"5", "1", "3", "2", "6", "4"}, // Ames, Baker Ann, Baker Cara, Cole Ben, Cole Fay, Diaz
			wantTotal:  6,
			wantCounts: map[string]int{"all": 6, "attention": 3, "w1": 2, "w2": 2, "w3": 2},
		},
		{
			name:      "attention filter: 2-flag student first, then 1-flag tied by name",
			q:         StudentQuery{Filter: "attention", Sort: "attention", Page: 1},
			wantIDs:   []string{"3", "5", "2"}, // Cara (2 flags), then Eve < Ben alphabetically (1 flag each)
			wantTotal: 3,
		},
		{
			name:      "w2 filter, sorted by name",
			q:         StudentQuery{Filter: "w2", Sort: "name", Page: 1},
			wantIDs:   []string{"3", "4"}, // Baker Cara, Diaz Dan
			wantTotal: 2,
		},
		{
			name:      "w3 filter, sorted by name",
			q:         StudentQuery{Filter: "w3", Sort: "name", Page: 1},
			wantIDs:   []string{"5", "6"}, // Ames Eve, Cole Fay
			wantTotal: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, total, counts := applyStudentQuery(all, tc.q)
			if got := userIDs(page); !reflect.DeepEqual(got, tc.wantIDs) {
				t.Errorf("page IDs = %v, want %v", got, tc.wantIDs)
			}
			if total != tc.wantTotal {
				t.Errorf("total = %d, want %d", total, tc.wantTotal)
			}
			if tc.wantCounts != nil && !reflect.DeepEqual(counts, tc.wantCounts) {
				t.Errorf("counts = %+v, want %+v", counts, tc.wantCounts)
			}
		})
	}
}

func TestApplyStudentQuerySorts(t *testing.T) {
	all := studentsFixture()

	tests := []struct {
		name    string
		sort    string
		wantIDs []string
	}{
		{
			name:    "quiz ascending, -1 last, stable among -1s",
			sort:    "quiz",
			wantIDs: []string{"3", "5", "2", "4", "1", "6"}, // 40,60,80,90, then -1,-1 in original order
		},
		{
			name:    "accuracy ascending, -1 last, stable among -1s",
			sort:    "accuracy",
			wantIDs: []string{"5", "2", "6", "4", "1", "3"}, // 30,60,70,90, then -1,-1 in original order
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, _, _ := applyStudentQuery(all, StudentQuery{Filter: "all", Sort: tc.sort, Page: 1})
			if got := userIDs(page); !reflect.DeepEqual(got, tc.wantIDs) {
				t.Errorf("page IDs = %v, want %v", got, tc.wantIDs)
			}
		})
	}
}

func TestApplyStudentQuerySearch(t *testing.T) {
	all := studentsFixture()

	// "ba" matches "Ann Baker" and "Cara Baker" (both contain "ba" in the surname),
	// not "Ben Cole" - counts reflect the search before the attention pill narrows it
	// further to just Cara (who has flags).
	q := StudentQuery{Q: "ba", Filter: "attention", Sort: "name", Page: 1}
	page, total, counts := applyStudentQuery(all, q)

	if got := userIDs(page); !reflect.DeepEqual(got, []string{"3"}) {
		t.Errorf("page IDs = %v, want [3]", got)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	wantCounts := map[string]int{"all": 2, "attention": 1, "w1": 1, "w2": 1}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Errorf("counts = %+v, want %+v", counts, wantCounts)
	}
}

func TestApplyStudentQueryPageClamping(t *testing.T) {
	// 25 students, all in "all" - 3 pages of size types.StudentsPageSize (10, 10, 5).
	var all []types.StudentInsight
	for i := 0; i < 25; i++ {
		all = append(all, types.StudentInsight{
			Student: types.Student{Firstname: "S", Lastname: "Student", UserID: string(rune('a' + i))},
		})
	}

	tests := []struct {
		name     string
		page     int
		wantLen  int
		wantPage int // can't observe the clamped page directly, so infer it from wantLen/position
	}{
		{name: "page 1 has 10", page: 1, wantLen: 10},
		{name: "page 3 has the remaining 5", page: 3, wantLen: 5},
		{name: "page 4 clamps down to page 3 (5 rows)", page: 4, wantLen: 5},
		{name: "page 0 clamps up to page 1 (10 rows)", page: 0, wantLen: 10},
		{name: "negative page clamps up to page 1 (10 rows)", page: -3, wantLen: 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, total, _ := applyStudentQuery(all, StudentQuery{Filter: "all", Sort: "name", Page: tc.page})
			if len(page) != tc.wantLen {
				t.Errorf("len(page) = %d, want %d", len(page), tc.wantLen)
			}
			if total != 25 {
				t.Errorf("total = %d, want 25", total)
			}
		})
	}
}

func TestApplyStudentQueryEmpty(t *testing.T) {
	page, total, counts := applyStudentQuery(nil, StudentQuery{Filter: "all", Sort: "name", Page: 1})
	if len(page) != 0 {
		t.Errorf("len(page) = %d, want 0", len(page))
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if counts["all"] != 0 {
		t.Errorf(`counts["all"] = %d, want 0`, counts["all"])
	}
}
