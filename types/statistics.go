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
