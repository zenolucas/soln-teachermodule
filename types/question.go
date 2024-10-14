package types

type MultipleChoiceQuestion struct {
	QuestionID    int    `json:"question_id"`
	QuestionText  string `json:"question_text"`
	Option1       string `json:"option_1"`
	Option2       string `json:"option_2"`
	Option3       string `json:"option_3"`
	Option4       string `json:"option_4"`
	CorrectAnswer string `json:"correct_answer"`
}

type FractionQuestion struct {
	QuestionID            int `json:"question_id"`
	Fraction1_Numerator   int `json:"fraction1_numerator"`
	Fraction1_Denominator int `json:"fraction1_denominator"`
	Fraction2_Numerator   int `json:"fraction2_numerator"`
	Fraction2_Denominator int `json:"fraction2_denominator"`
}

type WordedQuestion struct {
	QuestionID            int    `json:"question_id"`
	QuestionText          string `json:"question_text"`
	Fraction1_Numerator   int    `json:"fraction1_numerator"`
	Fraction1_Denominator int    `json:"fraction1_denominator"`
	Fraction2_Numerator   int    `json:"fraction2_numerator"`
	Fraction2_Denominator int    `json:"fraction2_denominator"`
}
