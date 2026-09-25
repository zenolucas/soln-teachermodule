package types

import "time"

// Shared thresholds for the insight logic (DEC-20). Views and handlers import these,
// so no literal 60 appears in markup logic. There's deliberately no WrongStreakMin:
// DEC-24 drops the wrong_streak flag until the pending game-client audit resolves V1.
const (
	PassPct                = 60
	MinAttemptsForAccuracy = 5
	MinResponsesForHint    = 5
	StudentsPageSize       = 10
)

// FlagKind identifies why a student was flagged for teacher attention. wrong_streak
// is deliberately absent (DEC-24).
type FlagKind string

const (
	FlagQuizBelow60 FlagKind = "quiz_below_60"
	FlagLowAccuracy FlagKind = "low_accuracy"
)

// Flag is one reason a student needs attention, scoped to a single scene.
type Flag struct {
	Kind       FlagKind
	MinigameID int
	Detail     string
}

// QuizScoreRow is one student's latest attempt at one quiz minigame (DEC-9: latest
// attempt per student per quiz, not best).
type QuizScoreRow struct {
	StudentID  int
	MinigameID int
	Score      int
	Total      int
}

// FractionAggRow is one student's summed right/wrong attempts on one fraction or
// worded minigame.
type FractionAggRow struct {
	StudentID  int
	MinigameID int
	Right      int
	Wrong      int
}

// Activity is one Recent activity event (02 §D1, 01 §1a). Kind is "played" (a
// fraction or worded scene - DEC-24 rules out "finished"/"started" copy) or "scored"
// (a quiz attempt). Score and Total are only meaningful when Kind == "scored".
type Activity struct {
	UserID      int
	First, Last string
	ClassroomID int
	ClassName   string
	MinigameID  int
	Kind        string
	Score       int
	Total       int
	At          time.Time
}
