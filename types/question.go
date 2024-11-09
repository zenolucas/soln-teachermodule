package types

type MultipleChoiceQuestion struct {
	QuestionID    int      `json:"question_id"`
	QuestionText  string   `json:"question_text"`
	Choices       []Choice `json:"choices"`
}

type Choice struct {
	ChoiceID   int    `json:"choice_id"`
	ChoiceText string `json:"choice_text"`
	IsCorrect bool `json:"is_correct"`
}

type FractionQuestion struct {
	QuestionID            int    `json:"question_id"`
	QuestionText          string `json:"question_text"`
	Fraction1_Numerator   int    `json:"fraction1_numerator"`
	Fraction1_Denominator int    `json:"fraction1_denominator"`
	Fraction2_Numerator   int    `json:"fraction2_numerator"`
	Fraction2_Denominator int    `json:"fraction2_denominator"`
}
