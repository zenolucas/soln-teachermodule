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

// insightsFixture is a 4-student fixture shared by TestBuildStudentInsights and
// TestSummarize: student 1 completed the game with high scores, student 2 is
// mid-game with a low_accuracy flag, student 3 is early with a quiz_below_60 flag,
// and student 4 has no rows at all.
func insightsFixture() ([]types.Student, map[int]map[int]bool, []types.QuizScoreRow, []types.FractionAggRow) {
	students := []types.Student{
		{Firstname: "Ann", Lastname: "A", UserID: "1"},
		{Firstname: "Bob", Lastname: "B", UserID: "2"},
		{Firstname: "Cy", Lastname: "C", UserID: "3"},
		{Firstname: "Dee", Lastname: "D", UserID: "4"},
	}

	finished := map[int]map[int]bool{
		1: {1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true},
		2: {1: true, 2: true, 3: true, 4: true, 5: true, 6: true},
		3: {1: true},
		4: {},
	}

	quiz := []types.QuizScoreRow{
		{StudentID: 1, MinigameID: 5, Score: 9, Total: 10},
		{StudentID: 1, MinigameID: 11, Score: 8, Total: 10},
		{StudentID: 1, MinigameID: 12, Score: 10, Total: 10},
		{StudentID: 3, MinigameID: 5, Score: 3, Total: 10},
	}

	frac := []types.FractionAggRow{
		{StudentID: 1, MinigameID: 1, Right: 10, Wrong: 0},
		{StudentID: 1, MinigameID: 2, Right: 10, Wrong: 0},
		{StudentID: 2, MinigameID: 6, Right: 2, Wrong: 8},
	}

	return students, finished, quiz, frac
}

func TestBuildStudentInsights(t *testing.T) {
	students, finished, quiz, frac := insightsFixture()
	got := buildStudentInsights(students, finished, quiz, frac)

	want := []types.StudentInsight{
		{Student: students[0], Current: 12, Completed: true, World: 3, QuizAvgPct: 90, AccuracyPct: 100},
		{Student: students[1], Current: 6, Completed: false, World: 2, QuizAvgPct: -1, AccuracyPct: 20,
			Flags: []types.Flag{{Kind: types.FlagLowAccuracy, MinigameID: 6, Detail: "Waterlogged Room 1 · 20% correct"}}},
		{Student: students[2], Current: 1, Completed: false, World: 1, QuizAvgPct: 30, AccuracyPct: -1,
			Flags: []types.Flag{{Kind: types.FlagQuizBelow60, MinigameID: 5, Detail: "Snekkers Quiz · 3/10"}}},
		{Student: students[3], Current: 1, Completed: false, World: 1, QuizAvgPct: -1, AccuracyPct: -1},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildStudentInsights(...) =\n%+v\nwant\n%+v", got, want)
	}
}

func TestSummarize(t *testing.T) {
	students, finished, quiz, frac := insightsFixture()
	insights := buildStudentInsights(students, finished, quiz, frac)
	room := types.Classroom{ClassroomID: "1", ClassroomName: "Test Class", Section: "A"}

	got := summarize(room, insights)
	want := types.ClassroomSummary{
		Classroom:    room,
		StudentCount: 4,
		QuizAvgPct:   60, // mean of student 1's 90 and student 3's 30
		FlaggedCount: 2,  // students 2 and 3
		ReachedW3:    1,  // student 1
		PerWorld:     [4]int{0, 2, 1, 1},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("summarize(...) = %+v, want %+v", got, want)
	}
}

func TestSummarizeNoQuizzesTaken(t *testing.T) {
	room := types.Classroom{ClassroomID: "2", ClassroomName: "Empty", Section: "B"}
	insights := []types.StudentInsight{
		{Student: types.Student{UserID: "1"}, Current: 1, World: 1, QuizAvgPct: -1, AccuracyPct: -1},
	}
	got := summarize(room, insights)
	if got.QuizAvgPct != -1 {
		t.Errorf("summarize(...).QuizAvgPct = %d, want -1 (no student has taken a quiz)", got.QuizAvgPct)
	}
	if got.FlaggedCount != 0 || got.ReachedW3 != 0 {
		t.Errorf("summarize(...) = %+v, want FlaggedCount 0 and ReachedW3 0", got)
	}
}

func TestBuildSceneSummaries(t *testing.T) {
	// 4 enrolled students. Scene 1 (fraction): 3 of them have a row. Scene 5 (quiz):
	// 2 of them have a row (i.e. scored it, per database.GetFinishedScenes' contract).
	// Scene 2 has no data at all.
	finished := map[int]map[int]bool{
		1: {1: true, 5: true},
		2: {1: true},
		3: {1: true, 5: true},
		4: {},
	}
	quizRows := []types.QuizScoreRow{
		{StudentID: 1, MinigameID: 5, Score: 8, Total: 10},
		{StudentID: 3, MinigameID: 5, Score: 6, Total: 10},
	}
	counts := map[int]int{1: 5, 5: 10}
	acc := map[int][2]int{1: {30, 10}}

	got := buildSceneSummaries(4, finished, quizRows, counts, acc)

	if len(got) != 12 {
		t.Fatalf("buildSceneSummaries(...) has %d scenes, want 12 (every scene in types.Worlds)", len(got))
	}

	scene1 := types.SceneSummary{MinigameID: 1, QuestionCount: 5, AccuracyPct: 75, QuizAvgPct: -1, CompletionPct: 75, TookCount: 3}
	if got[1] != scene1 {
		t.Errorf("buildSceneSummaries(...)[1] = %+v, want %+v", got[1], scene1)
	}

	scene5 := types.SceneSummary{MinigameID: 5, QuestionCount: 10, AccuracyPct: -1, QuizAvgPct: 70, CompletionPct: 50, TookCount: 2}
	if got[5] != scene5 {
		t.Errorf("buildSceneSummaries(...)[5] = %+v, want %+v", got[5], scene5)
	}

	// No data at all: 0 questions counted, no accuracy or quiz average to compute,
	// but completion is a real 0% (enrolled > 0), not n/a.
	scene2 := types.SceneSummary{MinigameID: 2, QuestionCount: 0, AccuracyPct: -1, QuizAvgPct: -1, CompletionPct: 0, TookCount: 0}
	if got[2] != scene2 {
		t.Errorf("buildSceneSummaries(...)[2] = %+v, want %+v", got[2], scene2)
	}
}

func TestBuildSceneSummariesNoEnrolledStudents(t *testing.T) {
	got := buildSceneSummaries(0, map[int]map[int]bool{}, nil, map[int]int{}, map[int][2]int{})
	if got[1].CompletionPct != -1 {
		t.Errorf("buildSceneSummaries(0, ...)[1].CompletionPct = %d, want -1 (no enrolled students to divide by)", got[1].CompletionPct)
	}
}

func TestSummarizeQuiz(t *testing.T) {
	t.Run("odd count: median is the middle score", func(t *testing.T) {
		scores := []types.StudentScore{{Score: 3}, {Score: 7}, {Score: 7}, {Score: 10}, {Score: 5}}
		got := summarizeQuiz(scores, 10, 5)

		want := types.QuizSummary{
			AvgPct:    64,
			AvgScore:  6.4,
			Median:    7,
			Took:      5,
			Enrolled:  5,
			Below:     2, // 3 and 5 are below 60% of 10
			Histogram: []int{0, 0, 0, 1, 0, 1, 0, 2, 0, 0, 1},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("summarizeQuiz(...) = %+v, want %+v", got, want)
		}
	})

	t.Run("even count: median averages the two middle scores", func(t *testing.T) {
		scores := []types.StudentScore{{Score: 4}, {Score: 6}}
		got := summarizeQuiz(scores, 10, 2)
		if got.Median != 5 {
			t.Errorf("Median = %v, want 5 (average of 4 and 6)", got.Median)
		}
	})

	t.Run("empty: zero stats, but the histogram is still total+1 long", func(t *testing.T) {
		got := summarizeQuiz(nil, 10, 8)
		want := types.QuizSummary{Took: 0, Enrolled: 8, Histogram: make([]int, 11)}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("summarizeQuiz(nil, 10, 8) = %+v, want %+v", got, want)
		}
	})

	t.Run("histogram length is always total+1", func(t *testing.T) {
		for _, total := range []int{0, 1, 6, 10} {
			got := summarizeQuiz([]types.StudentScore{{Score: 0}}, total, 1)
			if len(got.Histogram) != total+1 {
				t.Errorf("total %d: len(Histogram) = %d, want %d", total, len(got.Histogram), total+1)
			}
		}
	})
}

func TestBuildJourney(t *testing.T) {
	t.Run("mix of done-pass, done-fail, current and locked", func(t *testing.T) {
		ins := types.StudentInsight{Student: types.Student{UserID: "3"}, Current: 6, Completed: false}
		quiz := []types.QuizScoreRow{
			{StudentID: 3, MinigameID: 5, Score: 8, Total: 10}, // done, 80% -> pass
			{StudentID: 99, MinigameID: 5, Score: 1, Total: 10}, // a different student - must be ignored
		}
		frac := []types.FractionAggRow{
			{StudentID: 3, MinigameID: 1, Right: 2, Wrong: 8}, // done, 20% -> fail
			{StudentID: 99, MinigameID: 1, Right: 9, Wrong: 1}, // a different student - must be ignored
		}

		got := buildJourney(ins, quiz, frac)
		if len(got) != 12 {
			t.Fatalf("len(got) = %d, want 12", len(got))
		}

		want := map[int]struct {
			state string
			score string
		}{
			1:  {"done-fail", "20%"}, // 2/(2+8)
			2:  {"done-pass", ""},    // no row at all - defaults to pass
			3:  {"done-pass", ""},
			4:  {"done-pass", ""},
			5:  {"done-pass", "8/10"},
			6:  {"current", ""},
			7:  {"locked", ""},
			8:  {"locked", ""},
			9:  {"locked", ""},
			10: {"locked", ""},
			11: {"locked", ""},
			12: {"locked", ""},
		}
		for _, tile := range got {
			w, ok := want[tile.Scene.MinigameID]
			if !ok {
				t.Fatalf("unexpected scene %d in journey", tile.Scene.MinigameID)
			}
			if tile.State != w.state || tile.ScoreText != w.score {
				t.Errorf("scene %d: got {State:%q ScoreText:%q}, want {State:%q ScoreText:%q}",
					tile.Scene.MinigameID, tile.State, tile.ScoreText, w.state, w.score)
			}
		}
	})

	t.Run("completed student: every tile is done, none locked or current", func(t *testing.T) {
		ins := types.StudentInsight{Student: types.Student{UserID: "3"}, Current: 12, Completed: true}
		got := buildJourney(ins, nil, nil)
		for _, tile := range got {
			if tile.State == "current" || tile.State == "locked" {
				t.Errorf("scene %d: State = %q, want done-pass or done-fail (completed)", tile.Scene.MinigameID, tile.State)
			}
		}
	})
}

func TestNeighbours(t *testing.T) {
	ids := []string{"3", "7", "9", "2"}

	tests := []struct {
		name     string
		id       string
		wantPrev string
		wantNext string
	}{
		{"first element has no prev", "3", "", "7"},
		{"middle element has both", "7", "3", "9"},
		{"last element has no next", "2", "9", ""},
		{"id not in the list", "99", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prev, next := neighbours(ids, tc.id)
			if prev != tc.wantPrev || next != tc.wantNext {
				t.Errorf("neighbours(%v, %q) = (%q, %q), want (%q, %q)", ids, tc.id, prev, next, tc.wantPrev, tc.wantNext)
			}
		})
	}

	t.Run("single-element list", func(t *testing.T) {
		prev, next := neighbours([]string{"5"}, "5")
		if prev != "" || next != "" {
			t.Errorf("neighbours([5], 5) = (%q, %q), want (\"\", \"\")", prev, next)
		}
	})
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
