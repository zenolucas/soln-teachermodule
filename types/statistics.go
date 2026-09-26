package types

type StudentFractionStatistics struct {
	// QuestionID lets a caller match this summary row back to the question it's for
	// (see 02 §B3, T3.2) - GetFractionQuestionSummaries/GetWordedQuestionSummaries
	// didn't select it before, so the editor list had no way to link a row to its
	// question beyond position.
	QuestionID            int    `json:"question_id"`
	QuestionText          string `json:"question_text"`
	Fraction1_Numerator   int    `json:"fraction1_numerator"`
	Fraction1_Denominator int    `json:"fraction1_denominator"`
	Fraction2_Numerator   int    `json:"fraction2_numerator"`
	Fraction2_Denominator int    `json:"fraction2_denominator"`
	RightAttemptsCount    int    `json:"num_right_attempts"`
	WrongAttemptsCount    int    `json:"num_wrong_attempts"`
	// MaxWrongAttemptsCount is the highest single-row num_wrong_attempts for this
	// question (02 §C2's wrong_streak input) - only populated by
	// GetStudentFractionStatistics/GetStudentWordedStatistics (T5.6). Not surfaced
	// anywhere yet: DEC-24 drops the wrong_streak flag/stat pending the game-client
	// audit, but the aggregation is cheap to include alongside SUM while fixing X7.
	MaxWrongAttemptsCount int `json:"max_wrong_attempts"`
}

// QuestionAccuracy is a quiz question's per-question right/total breakdown for the
// editor list (02 §B3), from GetQuizQuestionAccuracy.
type QuestionAccuracy struct {
	QuestionID int
	Right      int
	Total      int
}

// QuizClick is one answer click the game posted for a student. The game posts every click, and keeps a
// question in play until it's answered correctly, so a question usually has several clicks.
type QuizClick struct {
	ResponseID int
	QuestionID int
	ChoiceID   int
	At         int64 // UNIX seconds; only compared, never shown
}

// QuizScoreRecord is one posted quiz score. The game posts a score only when a run is won.
type QuizScoreRecord struct {
	Score int
	At    int64 // UNIX seconds
}
