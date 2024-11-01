package types

type ClassStatistics struct {
	Score int
	Count int
}

type QuizResponseStatistics struct {
	Choice string `json:"choice"`
	Count  string `json:"count"`
}
