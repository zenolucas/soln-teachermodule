package types

type MultipleChoiceQuestion struct {
	QuestionID     string `json:"question_id"`
	QuestionText   string `json:"question_text"`
	Option1        string `json:"option_1"`
	Option2        string `json:"option_2"`
	Option3        string `json:"option_3"`
	Option4        string `json:"option_4"`
	CorrectAnswer  string `json:"correct_answer"`
}