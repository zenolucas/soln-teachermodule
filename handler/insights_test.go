package handler

import (
	"reflect"
	"testing"

	"soln-teachermodule/types"
)

func TestEvaluateFlagsQuizBelow60(t *testing.T) {
	tests := []struct {
		name string
		rows []types.QuizScoreRow
		want map[int][]types.Flag
	}{
		{
			name: "exactly 60% is not flagged",
			rows: []types.QuizScoreRow{{StudentID: 1, MinigameID: 5, Score: 6, Total: 10}},
			want: map[int][]types.Flag{},
		},
		{
			name: "below 60% is flagged with scene and score in the detail",
			rows: []types.QuizScoreRow{{StudentID: 1, MinigameID: 11, Score: 4, Total: 10}},
			want: map[int][]types.Flag{
				1: {{Kind: types.FlagQuizBelow60, MinigameID: 11, Detail: "Crab Quiz · 4/10"}},
			},
		},
		{
			name: "a Total 0 quiz is never flagged",
			rows: []types.QuizScoreRow{{StudentID: 1, MinigameID: 5, Score: 0, Total: 0}},
			want: map[int][]types.Flag{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateFlags(tc.rows, nil)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("EvaluateFlags(%+v, nil) = %+v, want %+v", tc.rows, got, tc.want)
			}
		})
	}
}

func TestEvaluateFlagsLowAccuracy(t *testing.T) {
	tests := []struct {
		name string
		rows []types.FractionAggRow
		want map[int][]types.Flag
	}{
		{
			name: "4 attempts at 0% is not flagged (below MinAttemptsForAccuracy)",
			rows: []types.FractionAggRow{{StudentID: 1, MinigameID: 7, Right: 0, Wrong: 4}},
			want: map[int][]types.Flag{},
		},
		{
			name: "5 attempts under 60% is flagged with scene and percent in the detail",
			rows: []types.FractionAggRow{{StudentID: 1, MinigameID: 10, Right: 3, Wrong: 7}},
			want: map[int][]types.Flag{
				1: {{Kind: types.FlagLowAccuracy, MinigameID: 10, Detail: "Rat · 30% correct"}},
			},
		},
		{
			name: "exactly 60% accuracy is not flagged",
			rows: []types.FractionAggRow{{StudentID: 1, MinigameID: 7, Right: 6, Wrong: 4}},
			want: map[int][]types.Flag{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateFlags(nil, tc.rows)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("EvaluateFlags(nil, %+v) = %+v, want %+v", tc.rows, got, tc.want)
			}
		})
	}
}

func TestEvaluateFlagsMultiplePerStudent(t *testing.T) {
	quiz := []types.QuizScoreRow{{StudentID: 1, MinigameID: 5, Score: 2, Total: 10}}
	frac := []types.FractionAggRow{{StudentID: 1, MinigameID: 1, Right: 1, Wrong: 9}}
	got := EvaluateFlags(quiz, frac)
	if len(got[1]) != 2 {
		t.Fatalf("student 1 flags = %+v, want 2 flags (one quiz, one accuracy)", got[1])
	}
}

func TestCurrentScene(t *testing.T) {
	tests := []struct {
		name          string
		played        map[int]bool
		quizScored    map[int]bool
		wantScene     int
		wantCompleted bool
	}{
		{
			name:      "nothing played",
			wantScene: 1,
		},
		{
			name:      "played 1,2,3",
			played:    map[int]bool{1: true, 2: true, 3: true},
			wantScene: 3,
		},
		{
			name:       "played 1-5, quiz 5 scored",
			played:     map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true},
			quizScored: map[int]bool{5: true},
			wantScene:  6,
		},
		{
			name:      "played 1-5, quiz 5 not scored",
			played:    map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true},
			wantScene: 5,
		},
		{
			name: "played through 12, quiz 12 scored",
			played: map[int]bool{
				1: true, 2: true, 3: true, 4: true, 5: true, 6: true,
				7: true, 8: true, 9: true, 10: true, 11: true, 12: true,
			},
			quizScored:    map[int]bool{5: true, 11: true, 12: true},
			wantScene:     12,
			wantCompleted: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotScene, gotCompleted := currentScene(tc.played, tc.quizScored)
			if gotScene != tc.wantScene || gotCompleted != tc.wantCompleted {
				t.Errorf("currentScene(%v, %v) = (%d, %v), want (%d, %v)",
					tc.played, tc.quizScored, gotScene, gotCompleted, tc.wantScene, tc.wantCompleted)
			}
		})
	}
}

func TestFinishedCount(t *testing.T) {
	tests := []struct {
		name      string
		current   int
		completed bool
		want      int
	}{
		{name: "nothing played, scene 1 current", current: 1, want: 0},
		{name: "played 1,2,3, scene 3 current", current: 3, want: 2},
		{name: "played 1-5 quiz scored, scene 6 current", current: 6, want: 5},
		{name: "played 1-5 quiz not scored, scene 5 current", current: 5, want: 4},
		{name: "completed", current: 12, completed: true, want: 12},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := finishedCount(tc.current, tc.completed)
			if got != tc.want {
				t.Errorf("finishedCount(%d, %v) = %d, want %d", tc.current, tc.completed, got, tc.want)
			}
		})
	}
}

func TestWorldOf(t *testing.T) {
	tests := []struct {
		minigameID int
		want       int
	}{
		{1, 1}, {5, 1}, {6, 2}, {11, 2}, {12, 3}, {99, 0},
	}
	for _, tc := range tests {
		if got := worldOf(tc.minigameID); got != tc.want {
			t.Errorf("worldOf(%d) = %d, want %d", tc.minigameID, got, tc.want)
		}
	}
}

func TestCollapseActivity(t *testing.T) {
	played := func(user, classroom, minigame int) types.Activity {
		return types.Activity{UserID: user, ClassroomID: classroom, MinigameID: minigame, Kind: "played"}
	}
	scored := func(user, classroom, minigame, score, total int) types.Activity {
		return types.Activity{UserID: user, ClassroomID: classroom, MinigameID: minigame, Kind: "scored", Score: score, Total: total}
	}

	tests := []struct {
		name string
		rows []types.Activity
		want []types.Activity
	}{
		{
			name: "consecutive played rows for the same student+scene collapse to one",
			rows: []types.Activity{played(3, 1, 1), played(3, 1, 1), played(3, 1, 1)},
			want: []types.Activity{played(3, 1, 1)},
		},
		{
			name: "played rows for different students don't collapse",
			rows: []types.Activity{played(3, 1, 1), played(4, 1, 1)},
			want: []types.Activity{played(3, 1, 1), played(4, 1, 1)},
		},
		{
			name: "played rows for different scenes don't collapse",
			rows: []types.Activity{played(3, 1, 1), played(3, 1, 2)},
			want: []types.Activity{played(3, 1, 1), played(3, 1, 2)},
		},
		{
			name: "scored rows are never collapsed, even when identical and adjacent",
			rows: []types.Activity{scored(3, 1, 5, 8, 10), scored(3, 1, 5, 8, 10)},
			want: []types.Activity{scored(3, 1, 5, 8, 10), scored(3, 1, 5, 8, 10)},
		},
		{
			name: "a scored row breaks up a run of played rows",
			rows: []types.Activity{played(3, 1, 1), scored(3, 1, 5, 8, 10), played(3, 1, 1)},
			want: []types.Activity{played(3, 1, 1), scored(3, 1, 5, 8, 10), played(3, 1, 1)},
		},
		{
			name: "empty input",
			rows: nil,
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := collapseActivity(tc.rows)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("collapseActivity(%+v) = %+v, want %+v", tc.rows, got, tc.want)
			}
		})
	}
}

func TestFlagLabel(t *testing.T) {
	tests := []struct {
		kind types.FlagKind
		want string
	}{
		{types.FlagQuizBelow60, "Quiz below 60%"},
		{types.FlagLowAccuracy, "Low accuracy"},
	}
	for _, tc := range tests {
		if got := FlagLabel(tc.kind); got != tc.want {
			t.Errorf("FlagLabel(%q) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}
