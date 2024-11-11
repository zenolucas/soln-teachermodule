package types

type FractionClassStatistics struct {
	RightAttemptsCount int `json:"num_right_attempts"`
	WrongAttemptsCount int `json:"num_wrong_attempts"`
}

type QuizClassStatistics struct {
	Score int
	Count int
}

type QuizResponseStatistics struct {
	Choice string `json:"choice"`
	Count  string `json:"count"`
}

type StudentQuizScore struct {
	FirstName string
	LastName  string
	Score     int
}

type StudentFractionStatistics struct {
	QuestionText          string `json:"question_text"`
	Fraction1_Numerator   int    `json:"fraction1_numerator"`
	Fraction1_Denominator int    `json:"fraction1_denominator"`
	Fraction2_Numerator   int    `json:"fraction2_numerator"`
	Fraction2_Denominator int    `json:"fraction2_denominator"`
	RightAttemptsCount    int    `json:"num_right_attempts"`
	WrongAttemptsCount    int    `json:"num_wrong_attempts"`
}

type StudentQuizStatistics struct {
	QuestionText  string `json:"question_text"`
	CorrectAnswer string `json:"correct_answer"`
	UserAnswer    string `json:"user_answer"`
	Score         int `json:"score"`
}
