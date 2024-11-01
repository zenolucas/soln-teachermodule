package types

type FractionClassStatistics struct {
	Right int
	Wrong int
}

type QuizClassStatistics struct {
	Score int
	Count int
}

type QuizResponseStatistics struct {
	Choice string `json:"choice"`
	Count  string `json:"count"`
}
