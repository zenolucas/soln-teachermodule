package handler

import (
	"testing"

	"soln-teachermodule/types"
)

// Two questions, choices A-B. Q1's right answer is choice 11 ("1"), Q2's is choice 22 ("2/3").
var quizQuestions = []types.MultipleChoiceQuestion{
	{QuestionID: 1, QuestionText: "What is 1/2 + 1/2?", Choices: []types.Choice{
		{ChoiceID: 11, ChoiceText: "1", IsCorrect: true}, {ChoiceID: 12, ChoiceText: "1/2"}}},
	{QuestionID: 2, QuestionText: "What is 1/3 + 1/3?", Choices: []types.Choice{
		{ChoiceID: 21, ChoiceText: "1/2"}, {ChoiceID: 22, ChoiceText: "2/3", IsCorrect: true}}},
}

func click(id, q, choice int, at int64) types.QuizClick {
	return types.QuizClick{ResponseID: id, QuestionID: q, ChoiceID: choice, At: at}
}

func TestStudentQuizDetail(t *testing.T) {
	type want struct {
		answers []string // first-try answer per question, "" = unanswered
		rights  []bool
		score   int
		wrong   int
	}
	tests := []struct {
		name   string
		clicks []types.QuizClick
		scores []types.QuizScoreRecord
		want   want
	}{
		{
			name:   "retries in one run: first try counts, not the eventual right answer",
			clicks: []types.QuizClick{click(1, 1, 12, 100), click(2, 1, 11, 101), click(3, 2, 22, 102)},
			scores: []types.QuizScoreRecord{{Score: 1, At: 103}},
			want:   want{answers: []string{"1/2", "2/3"}, rights: []bool{false, true}, score: 1, wrong: 1},
		},
		{
			name: "retake: the latest finished run's first tries",
			clicks: []types.QuizClick{
				click(1, 1, 12, 100), click(2, 1, 11, 101), click(3, 2, 21, 102), click(4, 2, 22, 103), // run 1
				click(5, 1, 11, 200), click(6, 2, 22, 201), // run 2: both right first time
			},
			scores: []types.QuizScoreRecord{{Score: 0, At: 104}, {Score: 2, At: 202}},
			want:   want{answers: []string{"1", "2/3"}, rights: []bool{true, true}, score: 2, wrong: 0},
		},
		{
			name: "an unfinished run after the latest finished one doesn't replace its first tries",
			clicks: []types.QuizClick{
				click(1, 1, 12, 100), click(2, 1, 11, 101), click(3, 2, 22, 102), // run 1
				click(4, 1, 11, 200), click(5, 2, 22, 201), // run 2, finished
				click(6, 1, 12, 300), // run 3, abandoned
			},
			scores: []types.QuizScoreRecord{{Score: 1, At: 103}, {Score: 2, At: 202}},
			want:   want{answers: []string{"1", "2/3"}, rights: []bool{true, true}, score: 2, wrong: 0},
		},
		{
			name:   "seed data: every row shares one timestamp, so fall back to all clicks",
			clicks: []types.QuizClick{click(1, 1, 12, 50), click(2, 2, 22, 50)},
			scores: []types.QuizScoreRecord{{Score: 1, At: 50}, {Score: 1, At: 50}},
			want:   want{answers: []string{"1/2", "2/3"}, rights: []bool{false, true}, score: 1, wrong: 1},
		},
		{
			name:   "never finished, one question unanswered",
			clicks: []types.QuizClick{click(1, 1, 11, 100)},
			scores: nil,
			want:   want{answers: []string{"1", ""}, rights: []bool{true, false}, score: -1, wrong: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, score, wrong := studentQuizDetail(quizQuestions, tt.clicks, tt.scores)
			if score != tt.want.score {
				t.Errorf("score = %d, want %d", score, tt.want.score)
			}
			if len(wrong) != tt.want.wrong {
				t.Errorf("wrong first tries = %d (%v), want %d", len(wrong), wrong, tt.want.wrong)
			}
			if len(rows) != len(quizQuestions) {
				t.Fatalf("got %d rows, want %d", len(rows), len(quizQuestions))
			}
			for i, row := range rows {
				if row.Answer != tt.want.answers[i] || row.Right != tt.want.rights[i] {
					t.Errorf("Q%d: answer %q right %v, want %q right %v", i+1, row.Answer, row.Right, tt.want.answers[i], tt.want.rights[i])
				}
			}
			if rows[0].Correct != "1" || rows[1].Correct != "2/3" {
				t.Errorf("correct answers = %q, %q", rows[0].Correct, rows[1].Correct)
			}
		})
	}
}
