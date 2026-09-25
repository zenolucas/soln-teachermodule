package database

import (
	"context"
	"database/sql"

	"soln-teachermodule/types"
)

// GetFractionQuestion fetches a single fraction or worded question, scoped to
// classroomID (SEC-06 - see X5/T0.1: without this in the WHERE, a teacher could look
// up another classroom's question by id alone). It also returns the question's
// minigame_id, since a caller working from just a questionID (the drawer's edit link,
// T3.4) doesn't otherwise know it. Returns sql.ErrNoRows if no such question exists in
// that classroom.
func GetFractionQuestion(ctx context.Context, questionID, classroomID int) (types.FractionQuestion, int, error) {
	var q types.FractionQuestion
	var minigameID int
	var questionText sql.NullString

	err := db.QueryRowContext(ctx,
		"SELECT question_id, minigame_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM fraction_questions WHERE question_id = ? AND classroom_id = ?",
		questionID, classroomID,
	).Scan(&q.QuestionID, &minigameID, &questionText, &q.Fraction1_Numerator, &q.Fraction1_Denominator, &q.Fraction2_Numerator, &q.Fraction2_Denominator)
	if err != nil {
		return types.FractionQuestion{}, 0, err
	}
	q.QuestionText = questionText.String
	return q, minigameID, nil
}

// GetQuizQuestion fetches a single quiz question with its choices, scoped to
// classroomID (see GetFractionQuestion). It also returns the question's minigame_id.
// Returns sql.ErrNoRows if no such question exists in that classroom.
func GetQuizQuestion(ctx context.Context, questionID, classroomID int) (types.MultipleChoiceQuestion, int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT q.minigame_id, q.question_text, c.choice_id, c.choice_text, c.is_correct
		FROM multiple_choice_questions q
		LEFT JOIN multiple_choice_choices c ON c.question_id = q.question_id
		WHERE q.question_id = ? AND q.classroom_id = ?
		ORDER BY c.choice_id
	`, questionID, classroomID)
	if err != nil {
		return types.MultipleChoiceQuestion{}, 0, err
	}
	defer rows.Close()

	var question types.MultipleChoiceQuestion
	var minigameID int
	found := false

	for rows.Next() {
		var mID int
		var questionText string
		var choiceID sql.NullInt64
		var choiceText sql.NullString
		var isCorrect sql.NullBool

		if err := rows.Scan(&mID, &questionText, &choiceID, &choiceText, &isCorrect); err != nil {
			return types.MultipleChoiceQuestion{}, 0, err
		}

		if !found {
			question.QuestionID = questionID
			question.QuestionText = questionText
			minigameID = mID
			found = true
		}

		// choice_id is NULL only when the question has zero choices (the LEFT JOIN
		// finds no match) - skip appending rather than adding a zero-valued choice
		// (same as GetQuizQuestions).
		if choiceID.Valid {
			question.Choices = append(question.Choices, types.Choice{
				ChoiceID:   int(choiceID.Int64),
				ChoiceText: choiceText.String,
				IsCorrect:  isCorrect.Bool,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return types.MultipleChoiceQuestion{}, 0, err
	}
	if !found {
		return types.MultipleChoiceQuestion{}, 0, sql.ErrNoRows
	}

	return question, minigameID, nil
}

// GetQuizQuestionAccuracy is a quiz scene's per-question right/total breakdown for the
// editor list (02 §B3), keyed by question_id. Scoped to students currently enrolled in
// classroomID (DEC-8), same as every other new aggregate.
func GetQuizQuestionAccuracy(ctx context.Context, classroomID, minigameID int) (map[int]types.QuestionAccuracy, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.question_id, SUM(c.is_correct), COUNT(*)
		FROM multiple_choice_responses r
		JOIN multiple_choice_choices c ON c.choice_id = r.choice_id
		JOIN enrollments e ON e.classroom_id = r.classroom_id AND e.student_id = r.student_id
		WHERE r.classroom_id = ? AND r.minigame_id = ?
		GROUP BY r.question_id
	`, classroomID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]types.QuestionAccuracy)
	for rows.Next() {
		var qa types.QuestionAccuracy
		if err := rows.Scan(&qa.QuestionID, &qa.Right, &qa.Total); err != nil {
			return nil, err
		}
		result[qa.QuestionID] = qa
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
